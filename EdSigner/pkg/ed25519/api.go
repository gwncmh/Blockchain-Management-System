package ed25519

import (
	"crypto/ed25519"
)

// --- WRAPPER CHO STANDARD LIBRARY ---

// GenerateKey tạo cặp khóa từ seed (32 bytes).
// Output: publicKey (32 bytes), privateKey (32 bytes - thực chất là seed gốc)
// Lưu ý: App của bạn đang lưu Seed (32 bytes) làm PrivateKey, nên ta trả về seed.
func GenerateKey(seed []byte) ([]byte, []byte) {
	if len(seed) != 32 {
		panic("seed must be exactly 32 bytes")
	}

	// Dùng thư viện chuẩn để tạo key
	privKeyStd := ed25519.NewKeyFromSeed(seed)
	
	// privKeyStd trong Go là 64 bytes (32 bytes seed + 32 bytes pubkey)
	// Ta tách lấy Public Key (32 bytes sau cùng)
	pubKey := make([]byte, 32)
	copy(pubKey, privKeyStd[32:])

	return pubKey, seed
}

// Sign tạo chữ ký bằng thư viện chuẩn.
// Input: privateKey (ở đây là Seed 32 bytes), pubKey (không dùng nhưng giữ để khớp interface cũ), message.
func Sign(seed, pubKey, message []byte) []byte {
	if len(seed) != 32 { return nil }

	// 1. Tái tạo Private Key chuẩn (64 bytes) từ Seed
	privKeyStd := ed25519.NewKeyFromSeed(seed)

	// 2. Ký (Ed25519 Pure)
	signature := ed25519.Sign(privKeyStd, message)

	return signature
}

// Verify xác thực chữ ký bằng thư viện chuẩn.
func Verify(pubKey, message, signature []byte) bool {
	if len(pubKey) != 32 || len(signature) != 64 {
		return false
	}
	return ed25519.Verify(pubKey, message, signature)
}

// VerifyBatchWrapper giữ lại để tương thích, nhưng gọi Verify đơn lẻ.
// Thư viện chuẩn không hỗ trợ Batch Verify native (vì nó phức tạp và ít an toàn hơn),
// nhưng với Desktop App thì verify loop vẫn cực nhanh.
func VerifyBatchWrapper(pubKeys, messages, signatures [][]byte) (bool, error) {
	if len(pubKeys) != len(messages) || len(pubKeys) != len(signatures) {
		return false, nil
	}

	for i := range pubKeys {
		if !Verify(pubKeys[i], messages[i], signatures[i]) {
			return false, nil
		}
	}
	return true, nil
}