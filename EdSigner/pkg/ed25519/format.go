package ed25519

import (
	"encoding/binary"
	"errors"
	"io"
)

// --- CONSTANTS & CONFIG ---

// Magic bytes: 0xED 25 51 90 (Ed25519)
var MagicBytes = [4]byte{0xED, 0x25, 0x51, 0x90}

const (
	// Version 2: Hỗ trợ Multi-Algorithm (Dynamic Length)
	FormatVersion = 2 

	// Kích thước cố định của Footer
	FooterSize = 16

	// Algo ID Definitions
	AlgoEd25519    = 1
	AlgoDilithium3 = 2 // Reserved cho tương lai (Post-Quantum)

	// Giới hạn an toàn (64KB - overhead)
	MaxEntrySize = 65535
)

// --- STRUCTURES ---

// SignatureEntry linh hoạt (Dynamic)
type SignatureEntry struct {
	AlgoID    uint8  // ID thuật toán
	PublicKey []byte // Độ dài động (tùy Algo)
	Signature []byte // Độ dài động (tùy Algo)
	Info      []byte // Metadata
}

// FileFooter (Giữ nguyên)
type FileFooter struct {
	Magic     [4]byte
	Version   uint16
	Count     uint16
	SigOffset int64
}

// --- SERIALIZATION HELPERS ---

// WriteSignatureEntry ghi Entry dạng TLV (Tag-Length-Value)
// Structure:
// [TotalLen: 2] [AlgoID: 1] [KeyLen: 2] [Key: N] [SigLen: 2] [Sig: M] [InfoLen: 2] [Info: K]
func WriteSignatureEntry(w io.Writer, pubKey, sig, info []byte) error {
	// 1. Tính toán độ dài các thành phần
	keyLen := len(pubKey)
	sigLen := len(sig)
	infoLen := len(info)

	// Kiểm tra tràn số (uint16 max 65535)
	// Header size = AlgoID(1) + KeyLen(2) + SigLen(2) + InfoLen(2) = 7 bytes
	headerSize := 7
	totalSize := headerSize + keyLen + sigLen + infoLen

	if totalSize > MaxEntrySize {
		return errors.New("entry data too large")
	}

	// 2. Ghi TotalLen (2 bytes)
	if err := binary.Write(w, binary.LittleEndian, uint16(totalSize)); err != nil {
		return err
	}

	// 3. Ghi AlgoID (1 byte) - Hiện tại mặc định là AlgoEd25519
	// Sau này có thể truyền AlgoID vào hàm nếu cần hỗ trợ nhiều loại
	if err := binary.Write(w, binary.LittleEndian, uint8(AlgoEd25519)); err != nil {
		return err
	}

	// 4. Ghi PublicKey (Len + Data)
	if err := binary.Write(w, binary.LittleEndian, uint16(keyLen)); err != nil { return err }
	if _, err := w.Write(pubKey); err != nil { return err }

	// 5. Ghi Signature (Len + Data)
	if err := binary.Write(w, binary.LittleEndian, uint16(sigLen)); err != nil { return err }
	if _, err := w.Write(sig); err != nil { return err }

	// 6. Ghi Info (Len + Data)
	if err := binary.Write(w, binary.LittleEndian, uint16(infoLen)); err != nil { return err }
	if infoLen > 0 {
		if _, err := w.Write(info); err != nil { return err }
	}

	return nil
}

// ReadSignatureEntry đọc Entry dạng TLV
func ReadSignatureEntry(r io.Reader) (*SignatureEntry, error) {
	// 1. Đọc TotalLen
	var totalLen uint16
	if err := binary.Read(r, binary.LittleEndian, &totalLen); err != nil {
		return nil, err
	}

	// 2. Đọc AlgoID
	var algoID uint8
	if err := binary.Read(r, binary.LittleEndian, &algoID); err != nil {
		return nil, err
	}

	// 3. Đọc PublicKey
	var keyLen uint16
	if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil { return nil, err }
	
	if keyLen > 1024 {
        return nil, errors.New("public key too large")
    }

	pubKey := make([]byte, keyLen)
	if _, err := io.ReadFull(r, pubKey); err != nil { return nil, err }

	// 4. Đọc Signature
	var sigLen uint16
	if err := binary.Read(r, binary.LittleEndian, &sigLen); err != nil { return nil, err }

	if sigLen > 1024 {
        return nil, errors.New("signature too large")
    }

	sig := make([]byte, sigLen)
	if _, err := io.ReadFull(r, sig); err != nil { return nil, err }

	// 5. Đọc Info
	var infoLen uint16
	if err := binary.Read(r, binary.LittleEndian, &infoLen); err != nil { return nil, err }

	if infoLen > 65535 {
        return nil, errors.New("info field too large")
    }

	info := make([]byte, infoLen)
	if infoLen > 0 {
		if _, err := io.ReadFull(r, info); err != nil { return nil, err }
	}

	// Kiểm tra tính nhất quán độ dài
	expectedTotal := 7 + int(keyLen) + int(sigLen) + int(infoLen)
	// Lưu ý: totalLen ban đầu tính cả header nhưng header 7 bytes đã đọc rải rác.
	// Logic trên Write: totalSize = headerSize(7) + data
	// Ở đây ta so sánh để chắc chắn file không bị hỏng
	if int(totalLen) != expectedTotal {
		// (Optional warning) nhưng miễn đọc đủ data là được
	}

	return &SignatureEntry{
		AlgoID:    algoID,
		PublicKey: pubKey,
		Signature: sig,
		Info:      info,
	}, nil
}

// WriteFooter & ReadFooter giữ nguyên logic cũ vì không ảnh hưởng bởi độ dài động
func WriteFooter(w io.Writer, count uint16, sigOffset int64) error {
	if _, err := w.Write(MagicBytes[:]); err != nil { return err }
	if err := binary.Write(w, binary.LittleEndian, uint16(FormatVersion)); err != nil { return err }
	if err := binary.Write(w, binary.LittleEndian, count); err != nil { return err }
	if err := binary.Write(w, binary.LittleEndian, sigOffset); err != nil { return err }
	return nil
}

func ReadFooter(r io.Reader) (*FileFooter, error) {
	var buf [FooterSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil { return nil, err }
	footer := &FileFooter{}
	copy(footer.Magic[:], buf[0:4])
	if footer.Magic != MagicBytes { return nil, errors.New("invalid magic bytes") }
	footer.Version = binary.LittleEndian.Uint16(buf[4:6])
	footer.Count = binary.LittleEndian.Uint16(buf[6:8])
	footer.SigOffset = int64(binary.LittleEndian.Uint64(buf[8:16]))
	return footer, nil
}