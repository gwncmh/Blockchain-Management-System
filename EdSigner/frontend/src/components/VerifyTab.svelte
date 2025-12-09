<script>
    import { SelectFile, VerifyFileAction, RemoveSignatureAction, CheckFileOnBlockchain } from '../../wailsjs/go/main/App';
    import { EventsOn } from '../../wailsjs/runtime/runtime';

    // --- STATE ---
    let filePath = "";      // File gốc hoặc file đã ký nhúng (Bắt buộc)
    let sigFilePath = "";   // File chữ ký .sig (Tùy chọn)

    let progress = 0;
    let isVerifying = false;
    let results = [];
    let errorMsg = "";
    // [NEW] State cho Blockchain verification
    let blockchainInfo = [];
    let isCheckingChain = false;

    // Lắng nghe tiến độ từ Backend
    EventsOn("verify-progress", (p) => {
        progress = p;
    });

    // --- CÁC HÀM CHỌN FILE ---
    async function handleBrowseFile() {
        let p = await SelectFile();
        if (p) filePath = p;
    }

    async function handleBrowseSig() {
        let p = await SelectFile();
        if (p) sigFilePath = p;
    }

    // --- HÀM XỬ LÝ CHÍNH ---
    async function handleVerify() {
        // 1. Validate đầu vào
        if (!filePath) {
            errorMsg = "⚠️ Vui lòng chọn File cần kiểm tra (Input 1)!";
            return;
        }

        // 2. Reset trạng thái
        isVerifying = true;
        progress = 0;
        results = [];
        errorMsg = "";

        try {
            // 3. Gọi Backend (Universal Verify)
            // Backend sẽ tự quyết định logic dựa trên việc sigFilePath có dữ liệu hay không
            const rawResults = await VerifyFileAction(filePath, sigFilePath);

            // 4. Xử lý kết quả trả về (Decode Metadata)
            results = rawResults.map(res => {
                const rawInfo = decodeBase64ToUTF8(res.Info);
                const parsed = parseMetadata(rawInfo);
                return { ...res, displayMeta: parsed };
            });

        } catch (err) {
            errorMsg = "Lỗi: " + err;
        } finally {
            isVerifying = false;
        }
    }

    // --- HELPER FUNCTIONS (Decode & Parse) ---
    function decodeBase64ToUTF8(base64Str) {
        try {
            const binaryString = atob(base64Str);
            const bytes = new Uint8Array(binaryString.length);
            for (let i = 0; i < binaryString.length; i++) {
                bytes[i] = binaryString.charCodeAt(i);
            }
            return new TextDecoder("utf-8").decode(bytes);
        } catch (e) {
            return base64Str;
        }
    }

    function parseMetadata(utf8Str) {
        try {
            const json = JSON.parse(utf8Str);
            if ((json.ver || json.v) && json.type) {
                return { isSchema: true, ...json };
            }
            return { isSchema: false, isJson: true, raw: utf8Str, note: "Non-standard JSON" };
        } catch (e) {
            return { isSchema: false, isJson: false, raw: utf8Str, note: utf8Str };
        }
    }

    async function handleExtract() {
        if (!confirm("Bạn muốn tách bỏ chữ ký để lấy lại file gốc?")) return;
        try {
            await RemoveSignatureAction(filePath);
            alert("✅ Đã trích xuất thành công! File mới có đuôi .original");
        } catch (e) {
            alert("Lỗi: " + e);
        }
    }

    // [NEW] Hàm kiểm tra Blockchain
    async function handleCheckBlockchain() {
        if (!filePath) {
            errorMsg = "⚠️ Vui lòng chọn file trước!";
            return;
        }

        isCheckingChain = true;
        blockchainInfo = [];

        try {
            const chainResults = await CheckFileOnBlockchain(filePath);
            blockchainInfo = chainResults;

            if (chainResults.length === 0 || !chainResults[0].exists) {
                errorMsg = "ℹ️ File này chưa được đăng ký trên Blockchain.";
            }
        } catch (err) {
            errorMsg = "Lỗi Blockchain: " + err;
        } finally {
            isCheckingChain = false;
        }
    }
</script>

<div class="tab-container">
    <h2>🔍 Kiểm Tra & Soi Chiếu (Universal Verify)</h2>
    
    <div class="card search-box">
        <div class="file-group">
            <label>1. File Cần Kiểm Tra (Original / Embedded File) <span class="req">*</span></label>
            <div class="file-input">
                <input type="text" readonly bind:value={filePath} placeholder="Chọn file gốc (PDF, Docx...) hoặc file đã ký nhúng..." />
                <button on:click={handleBrowseFile}>📂 Chọn...</button>
            </div>
        </div>

        <div class="file-group">
            <label>2. File Chữ Ký Rời (.sig) <span class="opt">(Tùy chọn)</span></label>
            <div class="file-input">
                <input type="text" readonly bind:value={sigFilePath} class:has-value={sigFilePath} placeholder="Chỉ chọn nếu bạn có file .sig riêng biệt..." />
                
                {#if sigFilePath}
                    <button class="btn-clear" on:click={() => sigFilePath = ""} title="Xóa file này">❌</button>
                {:else}
                    <button on:click={handleBrowseSig}>📂 Chọn...</button>
                {/if}
            </div>
        </div>
        
        <button class="btn-verify" on:click={handleVerify} disabled={isVerifying}>
             {isVerifying ? `Đang quét... ${progress.toFixed(1)}%` : "🔍 KIỂM TRA NGAY"}
        </button>
    </div>

    <!-- [NEW] Blockchain Verification Section -->
    <div class="card blockchain-verify-box">
        <h3>🌐 Kiểm Tra Trên Blockchain</h3>
        <p class="hint">Xác minh file này có được đăng ký trên sổ cái phân tán không</p>
        
        <button class="btn-chain-check" on:click={handleCheckBlockchain} disabled={isCheckingChain || !filePath}>
            {isCheckingChain ? "⏳ Đang truy vấn..." : "🔗 KIỂM TRA BLOCKCHAIN"}
        </button>

        {#if blockchainInfo.length > 0 && blockchainInfo[0].exists}
            <div class="chain-results">
                <div class="chain-success-banner">
                    ✅ File đã được chứng thực trên Blockchain!
                </div>

                {#each blockchainInfo as record}
                    <div class="chain-record">
                        <div class="record-row">
                            <span class="label">👤 Người ký:</span>
                            <code class="value addr">{record.signer}</code>
                        </div>
                        <div class="record-row">
                            <span class="label">🕐 Thời gian:</span>
                            <span class="value">{new Date(record.timestamp * 1000).toLocaleString('vi-VN')}</span>
                        </div>
                        <div class="record-row">
                            <span class="label">📦 Block:</span>
                            <span class="value">#{record.block_num}</span>
                        </div>
                        <div class="record-row">
                            <span class="label">🔗 TX Hash:</span>
                            <code class="value tx">{record.tx_hash}</code>
                            <button class="btn-copy-mini" on:click={() => navigator.clipboard.writeText(record.tx_hash)}>📋</button>
                        </div>
                    </div>
                {/each}
            </div>
        {:else if blockchainInfo.length > 0 && !blockchainInfo[0].exists}
            <div class="chain-not-found">
                ℹ️ File này chưa được đăng ký trên Blockchain
            </div>
        {/if}
    </div>

    {#if !results.length && !errorMsg}
        <div class="hint-box">
            <p>💡 <strong>Hướng dẫn:</strong></p>
            <ul>
                <li>Để kiểm tra <strong>Ký Nhúng</strong>: Chỉ cần chọn file ở mục (1).</li>
                <li>Để kiểm tra <strong>Ký Rời</strong> (hoặc Hybrid): Chọn file gốc ở mục (1) và file .sig ở mục (2).</li>
            </ul>
        </div>
    {/if}

    {#if errorMsg}
        <p class="error">{errorMsg}</p>
    {/if}

    {#if results.length > 0}
        <div class="results-area">
            <div class="result-header-row">
                <h3>Kết quả ({results.length} chữ ký được tìm thấy):</h3>
                {#if !sigFilePath}
                    <button class="btn-extract" on:click={handleExtract}>🔓 Trích xuất File Gốc</button>
                {/if}
            </div>
            
            {#each results as res}
                <div class="res-item {res.Valid ? 'valid-border' : 'invalid-border'}">
                    <div class="res-header">
                        <span class="idx">#{res.Index + 1}</span>
                        {#if res.Valid}
                             <span class="badge valid">✅ CHỮ KÝ HỢP LỆ (VALID)</span>
                        {:else}
                            <span class="badge invalid">❌ KHÔNG HỢP LỆ (INVALID)</span>
                        {/if}
                    </div>
                    
                    <div class="res-body">
                        {#if res.displayMeta.isSchema}
                            {@const meta = res.displayMeta}
                            
                            <div class="meta-tag-row">
                                 <span class="tag-type type-{meta.type}">
                                    {meta.type === 'custom_doc' ? 'VĂN BẢN KÝ' : meta.type.toUpperCase()}
                                </span>
                                 <span class="timestamp">🕒 {new Date(meta.ts).toLocaleString()}</span>
                            </div>

                            <div class="meta-content">
                                {#if meta.type === 'custom_doc'}
                                    <p class="big-text">{meta.data.full_name}</p>
                                    <div class="divider"></div>
                                    {#each Object.entries(meta.data) as [key, val]}
                                         {#if key !== 'full_name'}
                                            <div class="kv-row"><span class="k">{key}:</span> <span class="v">{val}</span></div>
                                        {/if}
                                     {/each}
                                {:else}
                                     <p class="generic-note">"{meta.note || JSON.stringify(meta.data)}"</p>
                                {/if}
                                <p class="meta-uuid">ID: {meta.id}</p>
                            </div>

                        {:else}
                            <div class="raw-content">
                                <span class="tag-type type-unknown">RAW DATA</span>
                                <p>"{res.displayMeta.raw}"</p>
                            </div>
                         {/if}
                    </div>

                    <div class="res-footer">
                        <span class="label">Public Key:</span>
                         <code class="pubkey">{res.PublicKey}</code>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .tab-container { padding: 20px; max-width: 750px; margin: 0 auto; }
    
    /* Search Box Styles */
    .search-box { background: #2a2a2a; padding: 20px; border-radius: 8px; border: 1px solid #444; margin-bottom: 20px; display: flex; flex-direction: column; gap: 15px; }
    
    .file-group label { display: block; margin-bottom: 5px; color: #ccc; font-size: 0.95rem; font-weight: bold; }
    .req { color: #ff4d4d; }
    .opt { color: #888; font-weight: normal; font-size: 0.85rem; margin-left: 5px; }

    .file-input { display: flex; gap: 5px; }
    .file-input input { flex: 1; padding: 12px; background: #111; border: 1px solid #555; color: white; border-radius: 4px; transition: border-color 0.2s; }
    .file-input input.has-value { border-color: #00d2ff; background: #1a252a; }
    .file-input button { padding: 0 15px; background: #444; color: white; border: none; cursor: pointer; border-radius: 4px; transition: 0.2s; }
    .file-input button:hover { background: #555; }
    
    .btn-clear { background: #4a2a2a !important; color: #ff8888 !important; }
    .btn-clear:hover { background: #663333 !important; }

    .btn-verify { padding: 15px; background: #ffaa00; color: black; font-weight: bold; border: none; border-radius: 4px; cursor: pointer; font-size: 1.1rem; margin-top: 5px; transition: 0.2s; }
    .btn-verify:hover { background: #e69900; }
    .btn-verify:disabled { background: #555; cursor: not-allowed; }

    /* Hint Box */
    .hint-box { background: #222; padding: 15px; border-radius: 6px; border: 1px dashed #444; color: #aaa; font-size: 0.9rem; }
    .hint-box ul { margin: 5px 0 0 20px; padding: 0; }
    .hint-box li { margin-bottom: 4px; }

    /* Results Styles */
    .results-area { margin-top: 20px; }
    .result-header-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
    .btn-extract { background: #333; color: #ccc; border: 1px solid #555; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 0.85rem; }
    .btn-extract:hover { background: #444; color: white; border-color: #777; }

    .res-item { background: #1e1e1e; margin-bottom: 15px; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px rgba(0,0,0,0.3); }
    .valid-border { border-left: 5px solid #00ff9d; }
    .invalid-border { border-left: 5px solid #ff4d4d; border: 1px solid #ff4d4d; }

    .res-header { background: #252525; padding: 10px 15px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #333; }
    .badge { padding: 4px 8px; border-radius: 4px; font-weight: bold; font-size: 0.85rem; }
    .valid { background: rgba(0, 255, 157, 0.1); color: #00ff9d; }
    .invalid { background: rgba(255, 77, 77, 0.1); color: #ff4d4d; }

    .res-body { padding: 15px; }
    
    /* Meta Content Styling */
    .meta-tag-row { display: flex; justify-content: space-between; margin-bottom: 10px; }
    .tag-type { padding: 3px 8px; border-radius: 3px; font-size: 0.75rem; font-weight: bold; text-transform: uppercase; background: #444; color: #ccc; }
    .type-custom_doc { background: #ff007b; color: white; }
    .type-contract { background: #00d2ff; color: black; }
    
    .timestamp { font-size: 0.8rem; color: #666; }
    
    .big-text { font-size: 1.2rem; font-weight: bold; color: white; margin-bottom: 5px; }
    .divider { border-top: 1px dashed #333; margin: 8px 0; }
    .kv-row { margin-bottom: 3px; font-size: 0.95rem; display: flex; }
    .kv-row .k { color: #888; width: 100px; }
    .kv-row .v { color: #00d2ff; font-weight: 500; }
    
    .meta-uuid { font-size: 0.75rem; color: #444; text-align: right; margin-top: 10px; font-family: monospace; }
    .generic-note { font-style: italic; color: #bbb; }

    .res-footer { background: #151515; padding: 8px 15px; border-top: 1px solid #333; font-size: 0.8rem; display: flex; align-items: center; gap: 10px; }
    .pubkey { color: #666; font-family: monospace; word-break: break-all; }
    
    .error { color: #ff4d4d; font-weight: bold; text-align: center; margin-top: 10px; }

    /* Blockchain Verify Styles */
    .blockchain-verify-box {
        background: #1a1a2e;
        border: 1px solid #00d2ff;
        margin-bottom: 20px;
    }

    .blockchain-verify-box h3 {
        color: #00d2ff;
        margin-bottom: 10px;
    }

    .blockchain-verify-box .hint {
        color: #888;
        font-size: 0.85rem;
        margin-bottom: 15px;
    }

    .btn-chain-check {
        width: 100%;
        padding: 12px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 6px;
        font-weight: bold;
        cursor: pointer;
        transition: 0.3s;
    }

    .btn-chain-check:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
    }

    .btn-chain-check:disabled {
        background: #444;
        cursor: not-allowed;
    }

    .chain-results {
        margin-top: 20px;
    }

    .chain-success-banner {
        background: rgba(0, 255, 157, 0.1);
        border: 2px solid #00ff9d;
        padding: 15px;
        border-radius: 8px;
        color: #00ff9d;
        font-weight: bold;
        text-align: center;
        margin-bottom: 15px;
        font-size: 1.1rem;
    }

    .chain-record {
        background: #252525;
        padding: 15px;
        border-radius: 6px;
        border-left: 4px solid #00d2ff;
        margin-bottom: 10px;
    }

    .record-row {
        display: flex;
        align-items: center;
        margin-bottom: 8px;
        gap: 10px;
    }

    .record-row:last-child {
        margin-bottom: 0;
    }

    .record-row .label {
        color: #888;
        min-width: 120px;
        font-size: 0.9rem;
    }

    .record-row .value {
        color: #fff;
        flex: 1;
    }

    .record-row .value.addr,
    .record-row .value.tx {
        font-family: monospace;
        font-size: 0.85rem;
        color: #00d2ff;
        word-break: break-all;
    }

    .btn-copy-mini {
        background: #333;
        border: 1px solid #555;
        padding: 4px 8px;
        border-radius: 4px;
        cursor: pointer;
        font-size: 0.85rem;
        transition: 0.2s;
    }

    .btn-copy-mini:hover {
        background: #444;
        border-color: #00d2ff;
    }

    .chain-not-found {
        margin-top: 15px;
        padding: 15px;
        background: rgba(255, 165, 0, 0.1);
        border: 1px dashed #ffa500;
        border-radius: 6px;
        color: #ffa500;
        text-align: center;
    }
</style>