# 🔐 Hệ thống Xác thực và Quản lý Tài liệu Số với Blockchain

> Bài tập lớn môn Công nghệ Blockchain

## 📖 Giới thiệu

Hệ thống xác thực tài liệu số sử dụng công nghệ Blockchain và chữ ký số, đảm bảo tính toàn vẹn và không thể thay đổi của tài liệu điện tử.

### 🎯 Mục tiêu

1. **Chứng minh tính xác thực** - Xác nhận tài liệu là bản gốc
2. **Chống giả mạo** - Phát hiện mọi thay đổi trong tài liệu
3. **Xác minh người ký** - Xác định danh tính người ký và thời điểm ký
4. **Lưu trữ bằng chứng** - Tạo hồ sơ không thể thay đổi trên blockchain

## 🏗️ Kiến trúc Hệ thống

```
┌─────────────┐      ┌─────────────┐      ┌──────────────┐
│   Frontend  │────▶│   Backend   │────▶│  Blockchain  │
│  React+Vite │      │  Node.js    │      │   Hardhat    │
└─────────────┘      └─────────────┘      └──────────────┘
     ↓                      ↓                      ↓
  Upload File         Calculate Hash       Store Hash
  Verify File         RSA Signature        Verify Hash
  View Documents      Interact with        Immutable
                      Smart Contract       Storage
```

## 🔧 Công nghệ Sử dụng

### Smart Contract Layer
- **Solidity** ^0.8.19 - Ngôn ngữ lập trình smart contract
- **Hardhat** 2.x - Framework phát triển Ethereum
- **OpenZeppelin** - Thư viện smart contract chuẩn

### Backend Layer
- **Node.js** < 21 - Runtime JavaScript
- **Express.js** - Web framework
- **Ethers.js** v6 - Thư viện tương tác blockchain
- **Crypto** - Module chữ ký số RSA native
- **Multer** - Xử lý file upload

### Frontend Layer
- **React** 18 - UI framework
- **Vite** - Build tool nhanh
- **Axios** - HTTP client
- **CSS3** - Styling hiện đại

## 📂 Cấu trúc Dự án

```
blockchain-document-auth/
│
├── hardhat/                    # Smart Contract Layer
│   ├── contracts/
│   │   └── DocumentAuth.sol    # Main smart contract
│   ├── scripts/
│   │   └── deploy.js           # Deployment script
│   ├── test/
│   │   └── DocumentAuth.test.js # Unit tests
│   ├── hardhat.config.js       # Hardhat configuration
│   └── package.json
│
├── backend/                    # Backend API Layer
│   ├── contracts/              # Deployed contract info
│   ├── server.js               # Express API server
│   ├── .env                    # Environment variables
│   └── package.json
│
├── frontend/                   # Frontend Layer
│   ├── src/
│   │   ├── contracts/          # Contract ABI & address
│   │   ├── App.jsx             # Main component
│   │   ├── App.css             # Styles
│   │   └── main.jsx            # Entry point
│   └── package.json
│
└── README.md                   # This file
```

## 🚀 Quick Start

### Prerequisites

```bash
node --version  # Phải < 21
npm --version
```

### Installation

```bash
# 1. Clone repository (hoặc tạo mới)
git clone <repo-url>
cd blockchain-document-auth

# 2. Setup Hardhat
cd hardhat
npm install
cd ..

# 3. Setup Backend
cd backend
npm install
cd ..

# 4. Setup Frontend
cd frontend
npm install
cd ..
```

### Running

**Terminal 1 - Blockchain Node:**
```bash
cd hardhat
npx hardhat node
```

**Terminal 2 - Deploy Contract:**
```bash
cd hardhat
npx hardhat run scripts/deploy.js --network localhost
```

**Terminal 3 - Backend API:**
```bash
cd backend
npm run dev
```

**Terminal 4 - Frontend:**
```bash
cd frontend
npm run dev
```

Mở trình duyệt tại: **http://localhost:5173**

## 📋 Tính năng Chính

### 1. 📤 Đăng ký Tài liệu

- Upload file bất kỳ (PDF, DOCX, TXT, hình ảnh...)
- Tự động tính hash SHA-256
- Tạo chữ ký số RSA 2048-bit
- Ghi hash và metadata lên blockchain
- Trả về transaction hash và block number

### 2. 🔍 Xác minh Tài liệu

- Upload file cần kiểm tra
- So sánh hash với blockchain
- Xác minh chữ ký số
- Hiển thị thông tin người ký và thời gian
- Phát hiện mọi thay đổi trong file

### 3. 📋 Quản lý Danh sách

- Xem tất cả tài liệu đã đăng ký
- Lọc theo người ký
- Xem chi tiết từng tài liệu
- Theo dõi lịch sử đăng ký

## 🔐 Bảo mật

### Hash Function
- **SHA-256**: Cryptographic hash function chuẩn
- One-way function: Không thể reverse
- Deterministic: Cùng input → cùng output
- Avalanche effect: Thay đổi 1 bit → hash hoàn toàn khác

### Digital Signature
- **RSA 2048-bit**: Asymmetric encryption
- Private key: Ký tài liệu (chỉ người sở hữu có)
- Public key: Xác minh chữ ký (công khai)
- Non-repudiation: Không thể chối bỏ

### Blockchain
- **Immutability**: Dữ liệu không thể thay đổi
- **Transparency**: Mọi người đều xem được
- **Decentralization**: Không có single point of failure
- **Timestamp**: Bằng chứng thời gian không thể giả mạo

## 🧪 Testing

### Chạy Unit Tests

```bash
cd hardhat
npx hardhat test
```

### Test Cases

- ✅ Contract deployment
- ✅ Document registration
- ✅ Duplicate prevention
- ✅ Document verification
- ✅ Signer verification
- ✅ Timestamp accuracy
- ✅ Multiple documents handling
- ✅ Gas optimization

### Test Coverage

```bash
npx hardhat coverage
```

## 📊 Flow Hoạt động

### Đăng ký Tài liệu

```
User Upload File
       ↓
Frontend: File → Buffer
       ↓
Backend: Buffer → SHA-256 Hash
       ↓
Backend: Hash → RSA Signature
       ↓
Smart Contract: Store (Hash + Signature + Metadata)
       ↓
Blockchain: Confirm Transaction
       ↓
Return: Transaction Hash + Block Number
```

### Xác minh Tài liệu

```
User Upload File
       ↓
Frontend: File → Buffer
       ↓
Backend: Buffer → SHA-256 Hash
       ↓
Smart Contract: Query Hash
       ↓
Blockchain: Return Document Info (if exists)
       ↓
Backend: Verify RSA Signature
       ↓
Return: Verified ✅ or Invalid ❌
```

## 🎓 Giá trị Học thuật

### Kiến thức được áp dụng

1. **Blockchain Fundamentals**
   - Smart contracts
   - Transactions
   - Gas optimization
   - Events & logs

2. **Cryptography**
   - Hash functions (SHA-256)
   - Digital signatures (RSA)
   - Public/Private key infrastructure

3. **Web Development**
   - RESTful API design
   - React component architecture
   - Asynchronous programming
   - Error handling

4. **Security**
   - Input validation
   - Authentication
   - Data integrity
   - Non-repudiation

## 📈 Mở rộng Tương lai

### Phase 2 (Nâng cao)
- [ ] Kết nối MetaMask wallet
- [ ] Multi-signature support
- [ ] IPFS integration (lưu file thật)
- [ ] Role-based access control
- [ ] Document expiration

### Phase 3 (Production)
- [ ] Deploy lên testnet/mainnet
- [ ] Database integration (PostgreSQL)
- [ ] User authentication system
- [ ] Email notifications
- [ ] Mobile app (React Native)

## 🐛 Troubleshooting

### Lỗi thường gặp

**1. "Cannot find module"**
```bash
npm install
```

**2. "Contract not found"**
```bash
cd hardhat
npx hardhat run scripts/deploy.js --network localhost
```

**3. "Invalid private key"**
- Kiểm tra file `.env`
- Đảm bảo Hardhat node đang chạy
- Sử dụng private key từ Hardhat test accounts

**4. "Port already in use"**
```bash
# Kill process on port
lsof -ti:3001 | xargs kill -9  # Backend
lsof -ti:5173 | xargs kill -9  # Frontend
```

## 📚 Tài liệu Tham khảo

- [Solidity Documentation](https://docs.soliditylang.org/)
- [Hardhat Documentation](https://hardhat.org/docs)
- [Ethers.js Documentation](https://docs.ethers.org/)
- [React Documentation](https://react.dev/)
- [Express.js Guide](https://expressjs.com/)

## 👥 Đóng góp

Dự án này là bài tập học thuật. Mọi đóng góp, ý kiến đều được hoan nghênh!

## 📄 License

MIT License - Tự do sử dụng cho mục đích học tập

## 🙏 Lời cảm ơn

- Giảng viên môn Công nghệ Blockchain
- Cộng đồng Ethereum developers
- OpenZeppelin team
- Hardhat contributors

---

**📧 Contact:** [Your Email]  
**🔗 GitHub:** [Your GitHub]  
**📅 Date:** 2024

**Happy Coding! 🚀**
