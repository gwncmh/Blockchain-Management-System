package ed25519

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto" // Cần cài: go get github.com/ethereum/go-ethereum
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// --- CONFIGURATION ---

const (
	KdfMemory      = 64 * 1024 // 64 MB
	KdfTime        = 4         // 4 Iteration
	KdfParallelism = 4         // 4 Threads
	SaltSize       = 16        // 16 bytes salt
)

// --- STRUCTURES ---

// KeyRing là chùm chìa khóa nằm bên trong lớp mã hóa
type KeyRing struct {
	Ed25519Seed []byte `json:"ed_seed"` // Seed cho ký file (32 bytes)
	EthPrivKey  []byte `json:"eth_key"` // Private Key cho Blockchain (32 bytes)
}

// KeyStore là cấu trúc JSON lưu trữ file bên ngoài (Public wrapper)
type KeyStore struct {
	Version      int          `json:"version"`
	ID           string       `json:"id"`
	KDF          string       `json:"kdf"`
	KDFParams    KDFParams    `json:"kdf_params"`
	Cipher       string       `json:"cipher"`
	CipherParams CipherParams `json:"cipher_params"`
	Ciphertext   string       `json:"ciphertext"` // Chứa KeyRing đã mã hóa
	Meta         string       `json:"meta"`
}

type KDFParams struct {
	Memory      uint32 `json:"m"`
	Time        uint32 `json:"t"`
	Parallelism uint8  `json:"p"`
	Salt        string `json:"salt"` // Hex
}

type CipherParams struct {
	Nonce string `json:"nonce"` // Hex
}

// --- CORE FUNCTIONS ---
// [NEW] encryptKeyRingLogic: Hàm nội bộ dùng chung để đóng gói KeyRing thành KeyStore v2
func encryptKeyRingLogic(password string, meta string, keyRing *KeyRing) (*KeyStore, error) {
	// 1. Serialize KeyRing ra JSON
	keyRingJSON, err := json.Marshal(keyRing)
	if err != nil { return nil, err }

	// 2. Chuẩn bị Salt & Key (Argon2id)
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil { return nil, err }

	key := argon2.IDKey([]byte(password), salt, KdfTime, KdfMemory, KdfParallelism, 32)
	
    // Xóa key khỏi RAM khi xong
    defer func() {
		for i := range key { key[i] = 0 }
	}()

	// 3. Mã hóa (XChaCha20-Poly1305)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil { return nil, err }

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil { return nil, err }

	ciphertext := aead.Seal(nil, nonce, keyRingJSON, nil)

	// 4. Tạo struct KeyStore v2
    // Lưu ý: Sau này muốn lên v3, bạn chỉ cần sửa số 2 thành 3 ở dòng dưới
	idBytes := make([]byte, 4); rand.Read(idBytes)
	
    ks := &KeyStore{
		Version:      2, // <--- LUÔN LÀ VERSION MỚI NHẤT
		ID:           hex.EncodeToString(idBytes),
		KDF:          "argon2id",
		KDFParams:    KDFParams{ Memory: KdfMemory, Time: KdfTime, Parallelism: KdfParallelism, Salt: hex.EncodeToString(salt) },
		Cipher:       "xchacha20-poly1305",
		CipherParams: CipherParams{ Nonce: hex.EncodeToString(nonce) },
		Ciphertext:   hex.EncodeToString(ciphertext),
		Meta:         meta,
	}
	return ks, nil
}

// EncryptKeyStore tạo KeyStore mới.
// EncryptKeyStore: Tạo khóa MỚI TINH (Generate New)
func EncryptKeyStore(password string, meta string) (*KeyStore, error) {
	if len(password) < 8 { return nil, errors.New("mật khẩu quá yếu") }

	// 1. Sinh ngẫu nhiên
	edSeed := make([]byte, 32); rand.Read(edSeed)
	ethKey, _ := crypto.GenerateKey()
	
	keyRing := &KeyRing{
		Ed25519Seed: edSeed,
		EthPrivKey:  crypto.FromECDSA(ethKey),
	}

	// 2. Gọi hàm đóng gói chung
	return encryptKeyRingLogic(password, meta, keyRing)
}

// DecryptKeyStore giải mã KeyStore.
// Trả về struct KeyRing chứa cả 2 loại key.
func DecryptKeyStore(password string, ks *KeyStore) (*KeyRing, error) {
	if ks.KDF != "argon2id" || ks.Cipher != "xchacha20-poly1305" {
		return nil, errors.New("thuật toán không hỗ trợ")
	}

	salt, err := hex.DecodeString(ks.KDFParams.Salt)
	if err != nil { return nil, err }
	nonce, err := hex.DecodeString(ks.CipherParams.Nonce)
	if err != nil { return nil, err }
	encryptedBytes, err := hex.DecodeString(ks.Ciphertext)
	if err != nil { return nil, err }

	key := argon2.IDKey([]byte(password), salt, ks.KDFParams.Time, ks.KDFParams.Memory, ks.KDFParams.Parallelism, 32)
	
	// Xóa key khỏi RAM ngay khi xong
	defer func() {
		for i := range key { key[i] = 0 }
	}()

	aead, err := chacha20poly1305.NewX(key)
	if err != nil { return nil, err }

	plaintext, err := aead.Open(nil, nonce, encryptedBytes, nil)
	if err != nil {
		return nil, errors.New("sai mật khẩu hoặc dữ liệu bị hỏng")
	}

	// Logic Migration: Xử lý tương thích ngược
	// Nếu Version 1 (Cũ): plaintext chính là seed (32 bytes)
	// Nếu Version 2 (Mới): plaintext là JSON KeyRing
	
	keyRing := &KeyRing{}

	if ks.Version == 1 || len(plaintext) == 32 {
        // Đây là file cũ, chỉ có Ed25519 Seed (32 bytes)
        keyRing.Ed25519Seed = plaintext
        
        // [FIX] Thay vì random, ta dùng Hash của Seed để làm Key Ethereum.
        // Điều này đảm bảo tính Deterministic: Cùng Seed -> Luôn ra cùng Key Eth.
        // Ta dùng Keccak256 (Hash chuẩn của Ethereum) để băm seed.
        ethKeyBytes := crypto.Keccak256(keyRing.Ed25519Seed)

        // Kiểm tra xem bytes này có tạo thành Private Key hợp lệ không
        // (Về mặt toán học xác suất lỗi cực thấp, nhưng vẫn nên check)
        privKey, err := crypto.ToECDSA(ethKeyBytes)
        if err != nil {
             return nil, errors.New("không thể dẫn xuất khóa Ethereum từ seed cũ")
        }

        keyRing.EthPrivKey = crypto.FromECDSA(privKey)
    } else {
		// File mới, unmarshal JSON
		if err := json.Unmarshal(plaintext, keyRing); err != nil {
			return nil, errors.New("lỗi cấu trúc dữ liệu khóa")
		}
	}

	return keyRing, nil
}

// SaveKeyStore & LoadKeyStore giữ nguyên
func SaveKeyStore(filename string, ks *KeyStore) error {
	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil { return err }
	return ioutil.WriteFile(filename, data, 0600)
}

func LoadKeyStore(filename string) (*KeyStore, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil { return nil, err }
	ks := &KeyStore{}
	if err := json.Unmarshal(data, ks); err != nil { return nil, err }
	return ks, nil
}

// [NEW] MigrateKeyStore: Nâng cấp file keystore lên phiên bản mới nhất
func MigrateKeyStore(filePath string, password string) error {
    // 1. Load file từ đĩa
    ks, err := LoadKeyStore(filePath)
    if err != nil { return err }

    // Kiểm tra: Nếu đã là version mới nhất thì không làm gì cả
    if ks.Version >= 2 { 
        return nil // Already up-to-date
    }

    // 2. Giải mã (Dùng logic DecryptKeyStore đã sửa ở bước trước)
    // Hàm này sẽ tự xử lý v1 -> KeyRing chuẩn (với Eth Key từ hash)
    keyRing, err := DecryptKeyStore(password, ks)
    if err != nil { return fmt.Errorf("không thể giải mã để nâng cấp: %v", err) }

    // 3. Đóng gói lại theo chuẩn mới nhất (v2)
    // Giữ nguyên Metadata cũ
    newKs, err := encryptKeyRingLogic(password, ks.Meta, keyRing)
    if err != nil { return err }

    // 4. Backup file cũ (An toàn là trên hết)
    backupPath := filePath + ".bak.v1"
    os.Rename(filePath, backupPath)

    // 5. Ghi file mới đè lên
    if err := SaveKeyStore(filePath, newKs); err != nil {
        // Nếu lỗi, khôi phục file backup
        os.Rename(backupPath, filePath)
        return err
    }
    
    // (Tuỳ chọn) Xóa file backup nếu thành công
    // os.Remove(backupPath) 

    return nil
}