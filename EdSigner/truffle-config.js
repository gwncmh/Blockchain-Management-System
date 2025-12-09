module.exports = {
  networks: {
    development: {
      host: "127.0.0.1",
      port: 7545,
      network_id: "*",
    },
  },

  // --- SỬA ĐOẠN NÀY ---
  compilers: {
    solc: {
      version: "0.8.20",    // Giữ nguyên phiên bản
      settings: {
        optimizer: {
          enabled: true,
          runs: 200
        },
        evmVersion: "paris" // <--- QUAN TRỌNG NHẤT: Ép dùng chuẩn Paris (không có PUSH0)
      }
    }
  }
  // --------------------
};