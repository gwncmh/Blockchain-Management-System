# 🔐 EdSigner

**EdSigner** là ứng dụng **desktop ký số tài liệu điện tử**, sử dụng thuật toán **Ed25519** kết hợp với **Blockchain Ethereum** nhằm đảm bảo **tính toàn vẹn, xác thực danh tính và bằng chứng tồn tại** cho tài liệu số.

Dự án được xây dựng như một **giải pháp thay thế và bổ trợ** cho chữ ký số truyền thống dựa trên **CA (Certificate Authority)**, tập trung vào **hiệu năng cao, khả năng kiểm chứng độc lập và kiến trúc phi tập trung**.

---

## 🎯 Mục tiêu dự án

- Ký số tài liệu bằng **Ed25519**
- Xác minh chữ ký **không cần kết nối Internet**
- Phát hiện mọi chỉnh sửa sau khi ký
- Cung cấp **bằng chứng tồn tại bất biến** thông qua blockchain
- Không phụ thuộc vào **tổ chức chứng thực (CA)**
- Giao diện desktop thân thiện (Windows / macOS / Linux)

---

## 🧠 Tính năng chính

### 🔑 Quản lý khóa (KeyStore v2)
- Hệ thống **2 cặp khóa**:
  - **Ed25519**: ký và xác minh tài liệu
  - **Ethereum**: tương tác blockchain
- Bảo mật cao:
  - **Argon2id**: dẫn xuất khóa
  - **XChaCha20-Poly1305**: mã hóa xác thực
- Keystore dạng JSON được mã hóa hoàn toàn

---

### ✍️ Ký số tài liệu (3 chế độ)
- **Embedded**: chữ ký nhúng trực tiếp vào file
- **Detached**: chữ ký lưu riêng dưới dạng `.sig`
- **Hybrid**: kết hợp cả hai

**Cấu trúc chữ ký**
```

[Magic Header: "ED25519SIG"]
[Version]
[Public Key]
[Signature]
[Metadata (JSON)]
[Nội dung file gốc]

```

---

### 🔍 Xác minh chữ ký
- Xác minh chữ ký nhúng hoặc rời
- Phát hiện mọi thay đổi nội dung file
- Trích xuất thông tin:
  - Người ký
  - Thời điểm ký
  - Ghi chú bổ sung

---

### ⛓️ Bằng chứng số trên Blockchain
- Băm tài liệu bằng:
  - SHA-512 → Keccak-256 (tương thích EVM)
- Lưu **hash tài liệu** lên blockchain Ethereum
- Truy xuất:
  - Transaction hash
  - Block number
  - Địa chỉ người đăng ký
- **Không lưu nội dung file lên blockchain**

---

### 🔐 Cơ chế bảo mật
- Tự động khóa sau 5 phút không hoạt động
- Xóa khóa bí mật khỏi bộ nhớ khi logout
- Chặn ký các file thực thi (`.exe`, `.dll`, `.sh`, …)
- Không lưu khóa riêng ở dạng plaintext

---

## 🏗️ Tổng quan kiến trúc

```

Người dùng
↓
Frontend (Svelte)
↓
Backend (Go + Wails)
↓
Mô-đun Ed25519 ──┐
├─ Blockchain Ethereum
Mô-đun Blockchain ┘

```

---

## 📁 Cấu trúc thư mục

```

EdSigner/
├── app.go
├── main.go
├── go.mod / go.sum
│
├── pkg/
│   ├── ed25519/          # Ký & xác minh Ed25519
│   └── blockchain/       # Tương tác Ethereum
│
├── contracts/
│   └── DocumentRegistry.sol
│
├── build/
│   ├── contracts/
│   ├── windows/
│   └── darwin/
│
├── frontend/             # Giao diện Svelte
└── wails.json

````

---

## ⚙️ Cài đặt & thiết lập

### 1️⃣ Yêu cầu môi trường
- Go **1.24+**
- Node.js **18+**
- Wails CLI
- Ganache (blockchain local)
- Truffle

---

### 2️⃣ Cài đặt công cụ

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
npm install -g ganache truffle
````

---

### 3️⃣ Triển khai Smart Contract

Khởi động Ganache:

```bash
ganache-cli --port 7545
```

Compile & deploy:

```bash
truffle compile
truffle migrate --reset
```

Cập nhật địa chỉ contract trong `app.go`:

```go
const (
    RPC_URL          = "http://127.0.0.1:7545"
    CONTRACT_ADDRESS = "0xYOUR_CONTRACT_ADDRESS"
)
```

---

### 4️⃣ Chạy ứng dụng

#### Chế độ phát triển

```bash
wails dev
```

#### Build production

```bash
wails build -platform windows/amd64
wails build -platform darwin/universal
wails build -platform linux/amd64
```

---

## 🚀 Hướng dẫn sử dụng

1. **Tạo khóa** → sinh keystore được mã hóa
2. **Mở khóa** → nhập mật khẩu
3. **Ký file** → chọn chế độ ký và metadata
4. **Xác minh** → kiểm tra tính toàn vẹn
5. **Đăng ký blockchain** → tạo bằng chứng số

---

## 📊 Hiệu năng

| Thao tác       | Thời gian          |
| -------------- | ------------------ |
| Sinh khóa      | ~2 giây            |
| Ký file 1MB    | ~10 ms             |
| Xác minh       | ~5 ms              |
| Ghi blockchain | ~15 giây (Ganache) |

---

## 🆚 So sánh với chữ ký số CA truyền thống

| Tiêu chí         | CA truyền thống (ECDSA) | EdSigner   |
| ---------------- | ----------------------- | ---------- |
| Thuật toán       | ECDSA                   | Ed25519    |
| Hiệu năng        | Chậm hơn                | Nhanh hơn  |
| Phụ thuộc CA     | Có                      | Không      |
| Xác minh offline | Hạn chế                 | Đầy đủ     |
| Dấu thời gian    | CA Timestamp            | Blockchain |
| Chi phí          | Phí duy trì CA          | Không      |

> **Lưu ý:** EdSigner không nhằm thay thế chữ ký số CA trong các thủ tục pháp lý nhà nước, mà đóng vai trò **bổ trợ** cho các hệ thống phi tập trung và nghiên cứu.

---

## 🔮 Hướng phát triển

* Ký đa chữ ký (Multi-signature)
* Lưu trữ IPFS
* Cơ chế thu hồi chữ ký
* Tích hợp HSM
* Timestamp server

---

## 📜 Giấy phép

Dự án phục vụ **mục đích học tập và nghiên cứu**.

---

## 🙌 Ghi nhận

* Thuật toán Ed25519
* Ethereum & Solidity
* Wails Framework

