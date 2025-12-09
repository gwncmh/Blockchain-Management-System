package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"EdSigner/pkg/ed25519"
	"EdSigner/pkg/blockchain" // Import package blockchain vừa viết

	"github.com/ethereum/go-ethereum/crypto" // Để tính Keccak256
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// --- BLOCKCHAIN CONFIG ---
// Bạn sẽ thay đổi 2 giá trị này sau khi Deploy
const (
    RPC_URL          = "http://127.0.0.1:7545" // Ganache mặc định
    CONTRACT_ADDRESS = "0x31d1a7863aC277a7E48a2f8F7eA6bC5bE3c2700c"                 // Sẽ điền sau khi deploy
)

// Thời gian Timeout (5 phút)
const InactivityTimeout = 5 * time.Minute

type SignMetadata struct {
	Version   string          `json:"ver"`
	ID        string          `json:"id"`
	Timestamp string          `json:"ts"`
	Type      string          `json:"type"`
	App       string          `json:"app"`
	Data      json.RawMessage `json:"data,omitempty"`
	Note      string          `json:"note"`
}

// BlockchainProofInfo để trả về Frontend
type BlockchainProofInfo struct {
	Exists    bool     `json:"exists"`    // File có tồn tại trên blockchain không
	Signer    string   `json:"signer"`    // Địa chỉ người ký
	Timestamp int64    `json:"timestamp"` // Thời gian ký (Unix)
	TxHash    string   `json:"tx_hash"`   // Transaction Hash
	BlockNum  uint64   `json:"block_num"` // Block Number
}

type App struct {
	ctx            context.Context
	
	// --- STATE ---
	currentPrivKey []byte // Ed25519 Seed (Dùng để ký File)
	currentEthKey  []byte // Ethereum Private Key (Dùng để ký Blockchain) - [NEW]
	currentPubKey  []byte // Ed25519 Public Key (Để hiển thị/verify)

	// Timer cho Auto-Lock
	logoutTimer    *time.Timer
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// --- LOGIC AUTO-LOCK (TỰ ĐỘNG KHÓA) ---

func (a *App) resetActivity() {
	if a.logoutTimer != nil {
		a.logoutTimer.Stop()
	}

	a.logoutTimer = time.AfterFunc(InactivityTimeout, func() {
		a.ClearKey()
		runtime.EventsEmit(a.ctx, "auto-lock-triggered")
	})
}

// --- LOGIC METADATA ---
func validateAndFormatMetadata(input string) ([]byte, error) {
	var meta SignMetadata
	if err := json.Unmarshal([]byte(input), &meta); err == nil {
		if meta.Version == "" { meta.Version = "1.0" }
		if meta.ID == "" { meta.ID = uuid.New().String() }
		if meta.Timestamp == "" { meta.Timestamp = time.Now().Format(time.RFC3339) }
		return json.Marshal(meta)
	}

	newMeta := SignMetadata{
		Version:   "1.0",
		Type:      "generic",
		ID:        uuid.New().String(),
		Note:      input,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	return json.Marshal(newMeta)
}

// --- BRIDGE METHODS ---

func (a *App) SelectFile() (string, error) {
	a.resetActivity()
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Chọn tập tin"})
}

func (a *App) SelectSaveFile(defaultName string) (string, error) {
	a.resetActivity()
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Chọn nơi lưu khóa",
		DefaultFilename:  defaultName,
		CanCreateDirectories: true,
	})
}

// GenerateAndSaveKey: Đã cập nhật để tương thích Keystore v2 (Dual Keys)
func (a *App) GenerateAndSaveKey(password string, meta string, savePath string) (string, error) {
	// 1. Tạo KeyStore (KeyRing sinh ngẫu nhiên bên trong hàm này)
	ks, err := ed25519.EncryptKeyStore(password, meta)
	if err != nil { return "", fmt.Errorf("lỗi tạo khóa: %v", err) }

	// 2. Lưu xuống đĩa
	if err := ed25519.SaveKeyStore(savePath, ks); err != nil {
		return "", fmt.Errorf("lỗi ghi file: %v", err)
	}

	// 3. Auto-Login: Giải mã ngay lập tức để lấy key vào RAM sử dụng luôn
	keyRing, err := ed25519.DecryptKeyStore(password, ks)
	if err != nil { return "", fmt.Errorf("lỗi giải mã (internal): %v", err) }

	// 4. Cập nhật State
	a.currentPrivKey = keyRing.Ed25519Seed
	a.currentEthKey = keyRing.EthPrivKey // [NEW] Lưu Eth Key vào RAM
	
	// Tạo Public Key Ed25519 để hiển thị
	pubKey, _ := ed25519.GenerateKey(a.currentPrivKey)
	a.currentPubKey = pubKey
	
	a.resetActivity()
	return hex.EncodeToString(pubKey), nil
}

// UnlockKey: Đã cập nhật để đọc KeyRing
func (a *App) UnlockKey(filePath string, password string) (string, error) {
	// 1. Gọi thử Migrate trước
    // Nếu là file cũ, nó sẽ tự nâng cấp lên v2 rồi lưu xuống đĩa.
    // Nếu là file mới, nó bỏ qua.
    if err := ed25519.MigrateKeyStore(filePath, password); err != nil {
        // Nếu migrate lỗi (ví dụ sai pass), ta chưa return vội, để hàm Decrypt bên dưới xử lý báo lỗi chuẩn
        fmt.Println("Warning: Migration failed or skipped:", err)
    }

	ks, err := ed25519.LoadKeyStore(filePath)
	if err != nil { return "", fmt.Errorf("lỗi đọc file: %v", err) }

	wasV1 := false
    if ks.Version < 2 {
        wasV1 = true
        // Gọi hàm migrate mà ta đã viết
        if err := ed25519.MigrateKeyStore(filePath, password); err != nil {
             return "", fmt.Errorf("lỗi nâng cấp khóa: %v", err)
        }
    }

	// 1. Giải mã lấy KeyRing
	keyRing, err := ed25519.DecryptKeyStore(password, ks)
	if err != nil { return "", fmt.Errorf("sai mật khẩu") }

	// 2. Cập nhật State
	a.currentPrivKey = keyRing.Ed25519Seed
	a.currentEthKey = keyRing.EthPrivKey // [NEW]
	
	pubKey, _ := ed25519.GenerateKey(a.currentPrivKey)
	a.currentPubKey = pubKey
	
	if wasV1 {
        runtime.EventsEmit(a.ctx, "migration-success")
    }

	a.resetActivity()
	return hex.EncodeToString(pubKey), nil
}

func (a *App) SignFileAction(filePath string, infoInput string, mode int) error {
	a.resetActivity()

	if len(a.currentPrivKey) != 32 {
		return fmt.Errorf("phiên làm việc hết hạn, vui lòng mở khóa lại")
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".exe" || ext == ".dll" || ext == ".bat" || ext == ".cmd" || ext == ".sh" {
		return fmt.Errorf("bảo mật: không cho phép ký vào file thực thi (%s)", ext)
	}

	jsonInfo, err := validateAndFormatMetadata(infoInput)
	if err != nil { return fmt.Errorf("lỗi metadata: %v", err) }

	onProgress := func(p float64) {
		runtime.EventsEmit(a.ctx, "sign-progress", p)
	}
	
	switch mode {
	case 0: // Embedded Only
		err = ed25519.SignFile(filePath, a.currentPrivKey, a.currentPubKey, jsonInfo, onProgress)
	
	case 1: // Detached Only
		err = ed25519.SignFileDetached(filePath, a.currentPrivKey, a.currentPubKey, jsonInfo, onProgress)

	case 2: // Hybrid
		err = ed25519.SignFile(filePath, a.currentPrivKey, a.currentPubKey, jsonInfo, func(p float64) {
			onProgress(p * 0.5)
		})
		if err != nil { return err }

		err = ed25519.SignFileDetached(filePath, a.currentPrivKey, a.currentPubKey, jsonInfo, func(p float64) {
			onProgress(50 + p*0.5)
		})
	}

	if err != nil { return fmt.Errorf("ký thất bại: %v", err) }
	return nil
}

func (a *App) VerifyFileAction(filePath string, sigPath string) ([]ed25519.VerificationResult, error) {
	a.resetActivity()
	onProgress := func(p float64) { runtime.EventsEmit(a.ctx, "verify-progress", p) }

	var allResults []ed25519.VerificationResult

	if sigPath != "" {
		detResults, err := ed25519.VerifyFileDetached(filePath, sigPath, onProgress)
		if err != nil { return nil, fmt.Errorf("Lỗi file .sig: %v", err) }
		allResults = append(allResults, detResults...)

		embResults, _ := ed25519.VerifyFile(filePath, nil)
		if len(embResults) > 0 {
			startIdx := len(allResults)
			for i := range embResults {
				embResults[i].Index = startIdx + i
			}
			allResults = append(allResults, embResults...)
		}
		
		return allResults, nil
	}

	return ed25519.VerifyFile(filePath, onProgress)
}

func (a *App) RemoveSignatureAction(filePath string) error {
	ext := filepath.Ext(filePath)
	nameWithoutExt := filePath[0 : len(filePath)-len(ext)]
	if strings.HasSuffix(nameWithoutExt, ".signed") {
		nameWithoutExt = strings.TrimSuffix(nameWithoutExt, ".signed")
	}
	newPath := nameWithoutExt + ".original" + ext
	err := ed25519.RestoreOriginalFile(filePath, newPath)
	if err != nil {
		return fmt.Errorf("lỗi trích xuất: %v", err)
	}
	return nil
}

func (a *App) GetCurrentKeyInfo() (string, bool) {
	a.resetActivity()
	if len(a.currentPubKey) == 32 {
		return hex.EncodeToString(a.currentPubKey), true
	}
	return "", false
}

// ClearKey: Xóa sạch 2 chìa khóa khỏi RAM
func (a *App) ClearKey() {
	if a.logoutTimer != nil {
		a.logoutTimer.Stop()
	}
	
	// Wipe Ed25519 Key
	if a.currentPrivKey != nil {
		for i := range a.currentPrivKey { a.currentPrivKey[i] = 0 }
	}
	a.currentPrivKey = nil

	// Wipe Eth Key [NEW]
	if a.currentEthKey != nil {
		for i := range a.currentEthKey { a.currentEthKey[i] = 0 }
	}
	a.currentEthKey = nil

	a.currentPubKey = nil
}

// RegisterFileOnChain: Đưa "Bằng chứng tồn tại" lên Blockchain
func (a *App) RegisterFileOnChain(filePath string) (string, error) {
    a.resetActivity()

    // 1. Kiểm tra Auth (Phải có Key Ethereum trong RAM)
    if len(a.currentEthKey) == 0 {
        return "", fmt.Errorf("bạn chưa đăng nhập hoặc khóa Ethereum không tồn tại")
    }
    
    // Check địa chỉ Contract (nếu chưa cấu hình)
    if CONTRACT_ADDRESS == "0x..." {
        return "", fmt.Errorf("hệ thống chưa cấu hình địa chỉ Smart Contract")
    }

    // 2. Hash File (SHA-512) - Lấy dấu vân tay file gốc
    // Dù file là PDF thường hay PDF đã ký nhúng, hàm này đều lấy đúng Hash của nội dung gốc.
    sha512Hash, err := ed25519.HashFile(filePath)
    if err != nil {
        return "", fmt.Errorf("lỗi đọc file: %v", err)
    }

    // 3. Hash Agility (Chuyển đổi hệ Hash)
    // Smart Contract dùng bytes32 (Keccak-256). Ta hash chuỗi SHA-512 một lần nữa.
    // Kết quả: keccak256(sha512(file))
    evmHash := crypto.Keccak256Hash(sha512Hash)
    
    // 4. Khởi tạo Blockchain Client
    client, err := blockchain.NewEthereumClient(RPC_URL, CONTRACT_ADDRESS)
    if err != nil {
        return "", fmt.Errorf("lỗi kết nối Blockchain: %v", err)
    }

    // 5. Gửi Transaction (Ký Offline bằng Private Key trong RAM)
    txHash, err := client.StoreProof(a.currentEthKey, evmHash)
    if err != nil {
        return "", fmt.Errorf("lỗi ghi Blockchain: %v", err)
    }

    return txHash, nil
}

// CheckFileOnBlockchain kiểm tra file có trên blockchain không và lấy thông tin
func (a *App) CheckFileOnBlockchain(filePath string) ([]BlockchainProofInfo, error) {
	a.resetActivity()

	// 1. Tính Hash của file (giống như khi đăng ký)
	sha512Hash, err := ed25519.HashFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("lỗi đọc file: %v", err)
	}

	// 2. Hash Agility (chuyển sang Keccak256)
	evmHash := crypto.Keccak256Hash(sha512Hash)
	var fileHash [32]byte
	copy(fileHash[:], evmHash[:])

	// 3. Kết nối Blockchain
	if CONTRACT_ADDRESS == "0x..." {
		return nil, fmt.Errorf("hệ thống chưa cấu hình Smart Contract")
	}

	client, err := blockchain.NewEthereumClient(RPC_URL, CONTRACT_ADDRESS)
	if err != nil {
		return nil, fmt.Errorf("lỗi kết nối Blockchain: %v", err)
	}

	// 4. Kiểm tra tồn tại
	exists, err := client.QueryProof(fileHash)
	if err != nil {
		return nil, fmt.Errorf("lỗi truy vấn: %v", err)
	}

	if !exists {
		// File chưa được đăng ký
		return []BlockchainProofInfo{{Exists: false}}, nil
	}

	// 5. Lấy lịch sử (có thể có nhiều lần ký từ nhiều người)
	records, err := client.GetProofHistory(fileHash)
	if err != nil {
		return nil, fmt.Errorf("lỗi lấy lịch sử: %v", err)
	}

	// 6. Convert sang struct để trả về Frontend
	results := make([]BlockchainProofInfo, 0, len(records))
	for _, rec := range records {
		results = append(results, BlockchainProofInfo{
			Exists:    true,
			Signer:    rec.Signer.Hex(),
			Timestamp: int64(rec.Timestamp),
			TxHash:    rec.TxHash,
			BlockNum:  rec.BlockNum,
		})
	}

	return results, nil
}

// [NEW] Helper trả về địa chỉ ví Ethereum công khai để hiển thị UI
func (a *App) GetEthAddress() string {
	if len(a.currentEthKey) == 0 {
		return ""
	}
	// Chuyển private key -> public address
	privKey, err := crypto.ToECDSA(a.currentEthKey)
	if err != nil { return "" }
	
	address := crypto.PubkeyToAddress(privKey.PublicKey)
	return address.Hex() // Trả về chuỗi "0x123..."
}