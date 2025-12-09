// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Tích hợp thư viện AccessControl của OpenZeppelin để quản lý phân quyền chuyên nghiệp
import "@openzeppelin/contracts/access/AccessControl.sol";

/**
 * @title DocumentRegistry
 * @dev Hệ thống lưu trữ bằng chứng số (Proof of Existence) với phân quyền RBAC.
 */
contract DocumentRegistry is AccessControl {
    
    // --- 1. ĐỊNH NGHĨA VAI TRÒ (ROLES) ---
    
    // STAFF_ROLE: Vai trò nhân viên, được phép ghi dữ liệu
    // (Lưu ý: DEFAULT_ADMIN_ROLE là vai trò Admin mặc định có sẵn trong AccessControl)
    bytes32 public constant STAFF_ROLE = keccak256("STAFF_ROLE");

    // --- 2. LƯU TRỮ TRẠNG THÁI (STATE) ---
    
    // Mapping dùng để kiểm tra nhanh: Hash này đã tồn tại chưa?
    // Key: File Hash (bytes32) -> Value: Đã tồn tại (true/false)
    mapping(bytes32 => bool) private _proofs;

    // --- 3. SỰ KIỆN (EVENTS) ---
    
    // Bắn sự kiện ra Log để tiết kiệm Gas (thay vì lưu toàn bộ info vào State)
    // Các ứng dụng (Frontend) sẽ đọc lịch sử sự kiện này để hiển thị chi tiết.
    event DocumentRegistered(
        bytes32 indexed fileHash, // Indexed để tìm kiếm nhanh
        address indexed signer,   // Ai là người ký?
        uint256 timestamp         // Thời gian ký
    );

    // --- 4. KHỞI TẠO (CONSTRUCTOR) ---
    
    constructor() {
        // Cấp quyền Admin tối cao cho người deploy contract (Bạn/Công ty)
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
    }

    // --- 5. CHỨC NĂNG CHÍNH ---

    /**
     * @dev storeProof: Ghi mã băm tài liệu lên Blockchain.
     * Chỉ người có quyền STAFF_ROLE mới gọi được hàm này.
     */
    function storeProof(bytes32 fileHash) public onlyRole(STAFF_ROLE) {
        // 1. Kiểm tra tài liệu đã tồn tại chưa để tránh spam/tốn gas vô ích
        require(!_proofs[fileHash], "DocumentRegistry: File already registered");

        // 2. Đánh dấu đã tồn tại vào State (Bộ nhớ on-chain)
        _proofs[fileHash] = true;

        // 3. Bắn pháo hiệu (Event) chứa thông tin chi tiết
        emit DocumentRegistered(fileHash, msg.sender, block.timestamp);
    }

    /**
     * @dev verifyProof: Hàm kiểm tra công khai (ai cũng gọi được)
     * Trả về true nếu file đã được đăng ký.
     */
    function verifyProof(bytes32 fileHash) public view returns (bool) {
        return _proofs[fileHash];
    }
    
    // --- CÁC HÀM QUẢN TRỊ NHÂN SỰ (Có sẵn trong AccessControl) ---
    // Admin sẽ dùng các hàm sau (không cần viết lại, đã kế thừa từ AccessControl):
    // - grantRole(STAFF_ROLE, address_nhan_vien): Thêm nhân viên
    // - revokeRole(STAFF_ROLE, address_nhan_vien): Sa thải/Xóa quyền nhân viên
}