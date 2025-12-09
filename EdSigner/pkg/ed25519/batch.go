package ed25519

import "errors"

// VerifyBatch xác thực hàng loạt (Production Safe Mode)
// Thay vì dùng Bos-Coster (phức tạp/rủi ro), ta dùng vòng lặp Verify chuẩn.
// Hiệu năng Go Verify chuẩn rất cao (~20k verify/giây), đủ cho Desktop App.
func VerifyBatch(pubKeys, messages, signatures [][]byte) (bool, error) {
	n := len(pubKeys)
	if n != len(messages) || n != len(signatures) {
		return false, errors.New("input lengths mismatch")
	}

	for i := 0; i < n; i++ {
		// Gọi hàm Verify từ api.go (đã trỏ về crypto/ed25519)
		if !Verify(pubKeys[i], messages[i], signatures[i]) {
			return false, nil // Có ít nhất 1 chữ ký sai
		}
	}

	return true, nil
}