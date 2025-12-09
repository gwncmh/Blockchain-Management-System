package ed25519

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// VerificationResult chứa kết quả xác thực cho từng chữ ký trong file
type VerificationResult struct {
	Index     int    // Số thứ tự chữ ký (0, 1, 2...)
	PublicKey []byte // Khóa công khai người ký
	Info      []byte // Thông tin người ký (Metadata/JSON)
	Valid     bool   // Kết quả xác thực (True/False)
}

const MaxFileSize = 10 * 1024 * 1024 * 1024 // 10GB

// ProgressFunc định nghĩa callback báo cáo tiến độ (0.0 -> 100.0)
type ProgressFunc func(percentage float64)

// --- Helper: Reader có báo cáo tiến độ ---
type progressReader struct {
	r          io.Reader
	total      int64
	current    int64
	onProgress ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.current += int64(n)
	if pr.onProgress != nil && pr.total > 0 {
		percent := float64(pr.current) / float64(pr.total) * 100
		pr.onProgress(percent)
	}
	return n, err
}

// SignFile thực hiện ký file an toàn (Atomic Save)
// Cơ chế: Copy nội dung sang file .tmp -> Ký vào .tmp -> Rename đè lên file gốc
func SignFile(filename string, privKey, pubKey, info []byte, onProgress ProgressFunc) error {
	// 1. Mở file gốc (Read Only)
	srcFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	stat, err := srcFile.Stat()
	if err != nil { return err }
	fileSize := stat.Size()

	if fileSize > MaxFileSize {
		return errors.New("file too large")
	}

	// 2. Phân tích cấu trúc file gốc (Tìm Footer cũ nếu có)
	var dataLimit int64      // Giới hạn dữ liệu gốc (không bao gồm chữ ký)
	var copyLimit int64      // Giới hạn dữ liệu cần copy sang file mới (Giữ lại các chữ ký cũ, bỏ footer cũ)
	var currentCount uint16

	hasFooter := false
	if fileSize >= int64(FooterSize) {
		if _, err := srcFile.Seek(-int64(FooterSize), io.SeekEnd); err == nil {
			if footer, err := ReadFooter(srcFile); err == nil {
				// Nếu file đã được ký trước đó
				hasFooter = true
				dataLimit = footer.SigOffset
				currentCount = footer.Count
				
				// Copy toàn bộ file TRỪ Footer cũ (16 bytes cuối)
				// Để ta nối chữ ký mới vào sau chữ ký cũ
				copyLimit = fileSize - int64(FooterSize)
			}
		}
	}

	if !hasFooter {
		// File chưa ký lần nào
		dataLimit = fileSize
		copyLimit = fileSize
		currentCount = 0
	}

	// 3. Tạo file tạm (Atomic Temp File)
	// Lưu file tạm cùng thư mục để đảm bảo lệnh Rename hoạt động nhanh (không phải copy sang ổ đĩa khác)
	ext := filepath.Ext(filename)
	tmpName := filename[0:len(filename)-len(ext)] + ".tmp" + ext
	
	dstFile, err := os.Create(tmpName)
	if err != nil {
		return err
	}
	
	// An toàn: Nếu hàm này lỗi giữa chừng, xóa file tạm đi
	defer func() {
		dstFile.Close()
		if err != nil {
			os.Remove(tmpName) 
		}
	}()

	// 4. QUY TRÌNH COPY & HASH (STREAMING)
	// Ta cần làm 2 việc:
	// a. Copy từ Source -> Dest (Đến copyLimit)
	// b. Tính Hash của Source (Chỉ đến dataLimit)
	
	// Reset con trỏ file gốc về đầu
	if _, err := srcFile.Seek(0, io.SeekStart); err != nil { return err }

	hasher := sha512.New()

	// Setup Progress Reader
	// Tổng bytes cần xử lý = copyLimit
	proxyReader := &progressReader{
		r:          srcFile,
		total:      copyLimit,
		onProgress: onProgress,
	}

	// Logic phức tạp: DataLimit <= CopyLimit
	// - Đoạn 1 (0 -> DataLimit): Copy vào file tạm VÀ đưa vào Hasher
	// - Đoạn 2 (DataLimit -> CopyLimit): Chỉ Copy vào file tạm (đây là các chữ ký cũ, không được hash lại)
	
	// Writer đa luồng cho Đoạn 1: Ghi vào file tạm + Hasher
	multiWriter := io.MultiWriter(dstFile, hasher)

	// Thực hiện Đoạn 1:
	if _, err := io.CopyN(multiWriter, proxyReader, dataLimit); err != nil {
		return err
	}

	// Thực hiện Đoạn 2 (Nếu có chữ ký cũ):
	remaining := copyLimit - dataLimit
	if remaining > 0 {
		// Chỉ copy, không hash
		if _, err := io.CopyN(dstFile, proxyReader, remaining); err != nil {
			return err
		}
	}

	// Hash xong
	fileHash := hasher.Sum(nil)
	if onProgress != nil { onProgress(90.0) } // 90% xong phần nặng nhất

	// 5. Tính toán chữ ký mới
	msg := append(fileHash, info...)
	signature := Sign(privKey, pubKey, msg)
	if signature == nil {
		return errors.New("signing failed")
	}

	// 6. Ghi chữ ký mới vào file tạm (Append)
	if err := WriteSignatureEntry(dstFile, pubKey, signature, info); err != nil {
		return err
	}

	// 7. Ghi Footer mới vào file tạm
	// Footer mới sẽ trỏ về dataLimit cũ (Vùng dữ liệu gốc không đổi)
	// Số lượng chữ ký tăng lên 1
	if err := WriteFooter(dstFile, currentCount+1, dataLimit); err != nil {
		return err
	}

	// 8. Đóng file để chuẩn bị đổi tên
	// Cần Sync để đảm bảo dữ liệu đã xuống đĩa cứng
	if err := dstFile.Sync(); err != nil { return err }
	dstFile.Close() 
	srcFile.Close()

	// 9. ATOMIC SWAP (Quan trọng nhất)
	// Xóa file tạm nếu rename thất bại (đã xử lý ở defer), 
	// nhưng nếu thành công thì không sao.
	
	// Trên Windows, đôi khi cần xóa file đích trước
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		// Nếu không xóa được file gốc (do đang mở?), báo lỗi
		// File tạm vẫn còn đó, dữ liệu không mất.
		return fmt.Errorf("cannot overwrite original file: %v", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("atomic swap failed: %v", err)
	}

	if onProgress != nil { onProgress(100.0) }
	return nil
}

// [NEW] SignFileDetached tạo file chữ ký rời (.sig)
// Không ghi đè file gốc. File .sig sẽ nằm cạnh file gốc.
func SignFileDetached(filename string, privKey, pubKey, info []byte, onProgress ProgressFunc) error {
	// 1. Mở file gốc để tính Hash
	srcFile, err := os.Open(filename)
	if err != nil { return err }
	defer srcFile.Close()

	stat, err := srcFile.Stat()
	if err != nil { return err }
	
	// Setup Progress
	proxyReader := &progressReader{
		r:          srcFile,
		total:      stat.Size(),
		onProgress: onProgress,
	}

	// 2. Tính Hash file gốc
	hasher := sha512.New()
	if _, err := io.Copy(hasher, proxyReader); err != nil {
		return err
	}
	fileHash := hasher.Sum(nil)
	
	if onProgress != nil { onProgress(90.0) }

	// 3. Tạo chữ ký
	msg := append(fileHash, info...)
	signature := Sign(privKey, pubKey, msg)
	if signature == nil {
		return errors.New("signing failed")
	}

	// 4. Ghi file .sig
	// Tên file .sig = Tên file gốc + ".sig"
	sigFilename := filename + ".sig"
	sigFile, err := os.Create(sigFilename)
	if err != nil { return err }
	defer sigFile.Close()

	// Ghi Entry chữ ký
	// Trong file rời, Offset bắt đầu từ 0
	if err := WriteSignatureEntry(sigFile, pubKey, signature, info); err != nil {
		return err
	}

	// Ghi Footer
	// Count = 1, SigOffset = 0 (Vì file này chỉ chứa chữ ký, không có nội dung gốc)
	if err := WriteFooter(sigFile, 1, 0); err != nil {
		return err
	}

	if onProgress != nil { onProgress(100.0) }
	return nil
}

// VerifyFile xác thực toàn bộ chữ ký trong file (có báo cáo tiến độ hash)
// Giữ nguyên logic cũ vì Verify chỉ đọc (Read-Only), không cần Atomic
func VerifyFile(filename string, onProgress ProgressFunc) ([]VerificationResult, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 1. Đọc Footer
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() < int64(FooterSize) {
		return nil, errors.New("file too small or not signed")
	}

	if _, err := f.Seek(-int64(FooterSize), io.SeekEnd); err != nil {
		return nil, err
	}
	footer, err := ReadFooter(f)
	if err != nil {
		return nil, errors.New("invalid file format or not signed")
	}

	if footer.Version != FormatVersion {
		return nil, fmt.Errorf("format version mismatch: file is v%d, tool expects v%d", footer.Version, FormatVersion)
	}

	// 2. Hash dữ liệu gốc (Có Progress)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	
	hasher := sha512.New()
	reader := io.LimitReader(f, footer.SigOffset)
	proxyReader := &progressReader{
		r:          reader,
		total:      footer.SigOffset,
		onProgress: onProgress,
	}

	if _, err := io.Copy(hasher, proxyReader); err != nil {
		return nil, err
	}
	fileHash := hasher.Sum(nil)
	
	if onProgress != nil { onProgress(100.0) }

	// 3. Duyệt và verify từng chữ ký
	if _, err := f.Seek(footer.SigOffset, io.SeekStart); err != nil {
		return nil, err
	}

	results := make([]VerificationResult, 0, footer.Count)

	for i := 0; i < int(footer.Count); i++ {
		entry, err := ReadSignatureEntry(f)
		if err != nil {
			return results, errors.New("error reading signature entry")
		}

		var isValid bool

		switch entry.AlgoID {
		case AlgoEd25519:
			msg := append(fileHash, entry.Info...)
			isValid = Verify(entry.PublicKey, msg, entry.Signature)
		
		case AlgoDilithium3:
			isValid = false
			
		default:
			isValid = false
		}

		results = append(results, VerificationResult{
			Index:     i,
			PublicKey: entry.PublicKey,
			Info:      entry.Info,
			Valid:     isValid,
		})
	}

	return results, nil
}

// [NEW] VerifyFileDetached xác thực file gốc dựa trên file chữ ký rời
func VerifyFileDetached(originalPath, sigPath string, onProgress ProgressFunc) ([]VerificationResult, error) {
	// 1. Đọc file .sig để lấy thông tin chữ ký
	sigFile, err := os.Open(sigPath)
	if err != nil { return nil, fmt.Errorf("cannot open sig file: %v", err) }
	defer sigFile.Close()

	// Đọc Footer của file .sig để validate format
	stat, _ := sigFile.Stat()
	if stat.Size() < int64(FooterSize) {
		return nil, errors.New("invalid .sig file")
	}
	if _, err := sigFile.Seek(-int64(FooterSize), io.SeekEnd); err != nil { return nil, err }
	footer, err := ReadFooter(sigFile)
	if err != nil { return nil, errors.New("invalid .sig footer") }

	// 2. Tính Hash của file GỐC (Original)
	// Lưu ý: Hash file gốc, không phải file .sig
	orgFile, err := os.Open(originalPath)
	if err != nil { return nil, fmt.Errorf("cannot open original file: %v", err) }
	defer orgFile.Close()

	orgStat, _ := orgFile.Stat()
	hasher := sha512.New()
	proxyReader := &progressReader{
		r:          orgFile,
		total:      orgStat.Size(),
		onProgress: onProgress,
	}

	if _, err := io.Copy(hasher, proxyReader); err != nil { return nil, err }
	fileHash := hasher.Sum(nil)

	if onProgress != nil { onProgress(90.0) }

	// 3. Đọc Entry trong file .sig và verify
	// Trong file rời, Signature nằm ngay đầu file (Offset 0)
	if _, err := sigFile.Seek(0, io.SeekStart); err != nil { return nil, err }

	results := make([]VerificationResult, 0)
	
	// Mặc định Detached Signature hiện tại chỉ support 1 signature/file (version 1)
	// Nhưng loop này hỗ trợ mở rộng sau này
	for i := 0; i < int(footer.Count); i++ {
		entry, err := ReadSignatureEntry(sigFile)
		if err != nil { break } // Hết data hoặc lỗi

		var isValid bool
		if entry.AlgoID == AlgoEd25519 {
			msg := append(fileHash, entry.Info...)
			isValid = Verify(entry.PublicKey, msg, entry.Signature)
		}

		results = append(results, VerificationResult{
			Index:     i,
			PublicKey: entry.PublicKey,
			Info:      entry.Info,
			Valid:     isValid,
		})
	}

	if onProgress != nil { onProgress(100.0) }
	return results, nil
}

// RestoreOriginalFile trích xuất phần dữ liệu gốc ra một file mới
// Input: signedPath (file đã ký), outPath (nơi lưu file gốc)
func RestoreOriginalFile(signedPath, outPath string) error {
	// 1. Mở file đã ký
	f, err := os.Open(signedPath)
	if err != nil { return err }
	defer f.Close()

	// 2. Đọc Footer để tìm điểm cắt (SigOffset)
	stat, err := f.Stat()
	if err != nil { return err }
	if stat.Size() < int64(FooterSize) {
		return errors.New("file too small, maybe not signed")
	}

	if _, err := f.Seek(-int64(FooterSize), io.SeekEnd); err != nil { return err }
	footer, err := ReadFooter(f)
	if err != nil {
		return errors.New("cannot read footer (file not signed?)")
	}

	// 3. Tạo file đầu ra
	outFile, err := os.Create(outPath)
	if err != nil { return err }
	defer outFile.Close()

	// 4. Copy đúng đoạn từ 0 -> SigOffset sang file mới
	if _, err := f.Seek(0, io.SeekStart); err != nil { return err }
	
	// Chỉ copy đúng lượng data gốc
	copied, err := io.CopyN(outFile, f, footer.SigOffset)
	if err != nil { return err }

	if copied != footer.SigOffset {
		return errors.New("integrity check failed during extraction")
	}

	return nil
}

// [NEW] HashFile chỉ tính SHA-512 của file (phục vụ Blockchain Integration)
func HashFile(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil { return nil, err }
	defer f.Close()

	// Nếu là file đã ký (có Footer), ta chỉ hash phần nội dung gốc
	// Nếu là file thường, hash toàn bộ
	stat, _ := f.Stat()
	limit := stat.Size()

	if stat.Size() >= int64(FooterSize) {
		if _, err := f.Seek(-int64(FooterSize), io.SeekEnd); err == nil {
			if footer, err := ReadFooter(f); err == nil {
				// Đây là file đã ký nhúng -> Chỉ hash dữ liệu gốc
				limit = footer.SigOffset
			}
		}
	}

	// Quay về đầu file để hash
	if _, err := f.Seek(0, io.SeekStart); err != nil { return nil, err }

	hasher := sha512.New()
	if _, err := io.Copy(hasher, io.LimitReader(f, limit)); err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}