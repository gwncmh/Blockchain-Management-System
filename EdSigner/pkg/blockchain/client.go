package blockchain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TxHash là chuỗi kết quả trả về (ví dụ: "0x123...")
type TxHash string

type EthereumClient struct {
	client          *ethclient.Client
	contractAddress common.Address
	parsedABI       abi.ABI
	chainID         *big.Int
}

// ProofRecord chứa thông tin bằng chứng từ Blockchain
type ProofRecord struct {
	FileHash  [32]byte       // Hash của file
	Signer    common.Address // Địa chỉ người ký
	Timestamp uint64         // Thời gian ký (Unix timestamp)
	TxHash    string         // Transaction Hash
	BlockNum  uint64         // Block Number
}

// NewEthereumClient khởi tạo kết nối
func NewEthereumClient(rpcURL string, contractAddrHex string) (*EthereumClient, error) {
	// 1. Kết nối RPC
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối node blockchain: %v", err)
	}

	// 2. Lấy ChainID (để chống Replay Attack)
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("lỗi lấy chainID: %v", err)
	}

	// 3. Parse ABI
	parsedABI, err := abi.JSON(strings.NewReader(DocumentRegistryABI))
	if err != nil {
		return nil, fmt.Errorf("lỗi parse ABI: %v", err)
	}

	return &EthereumClient{
		client:          client,
		contractAddress: common.HexToAddress(contractAddrHex),
		parsedABI:       parsedABI,
		chainID:         chainID,
	}, nil
}

// QueryProof kiểm tra xem file hash có tồn tại trên blockchain không
func (e *EthereumClient) QueryProof(fileHash [32]byte) (bool, error) {
	// Gọi hàm verifyProof(bytes32) view function
	data, err := e.parsedABI.Pack("verifyProof", fileHash)
	if err != nil {
		return false, err
	}

	// CallContract (read-only, không tốn gas)
	result, err := e.client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &e.contractAddress,
		Data: data,
	}, nil)
	
	if err != nil {
		return false, err
	}

	// Parse kết quả (bool)
	var exists bool
	if err := e.parsedABI.UnpackIntoInterface(&exists, "verifyProof", result); err != nil {
		return false, err
	}

	return exists, nil
}

// GetProofHistory lấy lịch sử đăng ký của một file hash từ Event logs
func (e *EthereumClient) GetProofHistory(fileHash [32]byte) ([]ProofRecord, error) {
	// 1. Tạo filter để tìm Event DocumentRegistered có fileHash này
	// Event signature: DocumentRegistered(bytes32 indexed fileHash, address indexed signer, uint256 timestamp)
	
	query := ethereum.FilterQuery{
		Addresses: []common.Address{e.contractAddress},
		Topics: [][]common.Hash{
			{crypto.Keccak256Hash([]byte("DocumentRegistered(bytes32,address,uint256)"))}, // Event signature
			{common.BytesToHash(fileHash[:])}, // Filter theo fileHash (indexed param đầu tiên)
		},
		FromBlock: big.NewInt(0), // Tìm từ block đầu tiên
		ToBlock:   nil,           // Đến block mới nhất
	}

	// 2. Query logs
	logs, err := e.client.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("lỗi query logs: %v", err)
	}

	if len(logs) == 0 {
		return nil, nil // Không tìm thấy
	}

	// 3. Parse từng log thành ProofRecord
	records := make([]ProofRecord, 0, len(logs))
	
	for _, vLog := range logs {
		// Topics[0] = Event signature (đã biết)
		// Topics[1] = fileHash (indexed)
		// Topics[2] = signer (indexed)
		// Data = timestamp (non-indexed)
		
		if len(vLog.Topics) < 3 {
			continue // Log bị lỗi, bỏ qua
		}

		signer := common.HexToAddress(vLog.Topics[2].Hex())
		
		// Unpack timestamp từ Data
		var timestamp uint64
		if len(vLog.Data) >= 32 {
			timestamp = new(big.Int).SetBytes(vLog.Data[:32]).Uint64()
		}

		records = append(records, ProofRecord{
			FileHash:  fileHash,
			Signer:    signer,
			Timestamp: timestamp,
			TxHash:    vLog.TxHash.Hex(),
			BlockNum:  vLog.BlockNumber,
		})
	}

	return records, nil
}

// StoreProof thực hiện quy trình ký và gửi giao dịch
// Input: privKeyBytes (Lấy từ keystore app), fileHash (đã băm 2 lần)
func (e *EthereumClient) StoreProof(privKeyBytes []byte, fileHash [32]byte) (string, error) {
	// 1. Tái tạo Private Key từ bytes
	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return "", errors.New("private key không hợp lệ")
	}

	// 2. Lấy địa chỉ ví từ Private Key
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("lỗi public key")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 3. Lấy Nonce (Số thứ tự giao dịch)
	nonce, err := e.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("lỗi lấy nonce: %v", err)
	}

	// 4. Ước lượng Gas Price
	gasPrice, err := e.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("lỗi lấy gas price: %v", err)
	}

	// 5. Đóng gói dữ liệu gọi hàm (Call Data)
	// Gọi hàm "storeProof" với tham số fileHash
	data, err := e.parsedABI.Pack("storeProof", fileHash)
	if err != nil {
		return "", fmt.Errorf("lỗi đóng gói dữ liệu: %v", err)
	}

	// 6. Tạo Transaction (Raw Tx)
	// Gas Limit: Set cứng 100,000 (đủ cho việc ghi mapping), hoặc có thể EstimateGas
	tx := types.NewTransaction(nonce, e.contractAddress, big.NewInt(0), 100000, gasPrice, data)

	// 7. Ký Transaction (Offline Signing)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.chainID), privKey)
	if err != nil {
		return "", fmt.Errorf("lỗi ký transaction: %v", err)
	}

	// 8. Gửi Transaction đã ký lên mạng
	err = e.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("gửi transaction thất bại: %v", err)
	}

	// Trả về Transaction Hash để tra cứu
	return signedTx.Hash().Hex(), nil
}

// Helper: Kiểm tra kết nối
func (e *EthereumClient) IsConnected() bool {
	_, err := e.client.BlockNumber(context.Background())
	return err == nil
}