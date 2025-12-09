<script>
    import { SelectFile, SignFileAction, RegisterFileOnChain } from '../../wailsjs/go/main/App';
    import { EventsOn } from '../../wailsjs/runtime/runtime';

    // --- STATE QUẢN LÝ FILE & TIẾN TRÌNH ---
    let filePath = "";
    let progress = 0;
    let isSigning = false;
    let statusMsg = "";
    let isError = false;
    let signMode = 0; // 0: Embedded, 1: Detached, 2: Hybrid
    let isRegistering = false;
    let txHash = "";

    // --- STATE QUẢN LÝ METADATA ---
    
    // 1. Loại tài liệu (Mặc định chọn Generic hoặc Custom tùy bạn)
    let docType = "custom_doc"; 

    // 2. Mảng chứa các trường tùy chỉnh (Dynamic Fields)
    // Mặc định tạo sẵn 1 dòng để người dùng đỡ phải bấm thêm
    let customFields = [
        { key: "Chức vụ", value: "" }
    ];

    // 3. Kho dữ liệu nhập liệu (Data Store)
    let metaData = {
        // Generic
        note: "",
        
        // Custom Doc (Ký văn bản) [NEW]
        signerName: "", // Họ tên cố định

        // Official Letter (Công văn)
        letterRefNo: "",    
        letterIssuer: "",   
        letterReceiver: "", 
        letterSubject: "",

        // Diploma (Bằng cấp)
        studentName: "",
        studentId: "",
        degree: "BSc",
        certNo: "",

        // Contract (Hợp đồng)
        contractNo: "",
        partner: "",
        value: 0,
        currency: "VND"
    };

    // --- LOGIC XỬ LÝ DYNAMIC FIELDS ---
    function addCustomField() {
        customFields = [...customFields, { key: "", value: "" }];
    }

    function removeCustomField(index) {
        customFields = customFields.filter((_, i) => i !== index);
    }

    // --- WAILS EVENTS ---
    EventsOn("sign-progress", (p) => {
        progress = p;
    });

    async function handleBrowse() {
        let p = await SelectFile();
        if (p) filePath = p;
    }

    // --- HÀM CHUẨN BỊ JSON METADATA ---
    function prepareMetadata() {
        let payload = {
            v: "1.0",
            ts: new Date().toISOString(),
            type: docType,
            data: {}, 
            note: "" 
        };

        if (docType === "custom_doc") {
            // --- E. KÝ VĂN BẢN (CUSTOM) ---
            let dynamicData = {
                full_name: metaData.signerName // Trường cố định
            };

            // Gom các trường động vào object data
            customFields.forEach(field => {
                // Chỉ lấy các trường có tên (key) không rỗng
                if (field.key.trim() !== "") {
                    dynamicData[field.key] = field.value;
                }
            });

            payload.data = dynamicData;

            // Tạo Note tóm tắt thông minh
            // VD: "Ký văn bản: Nguyễn Văn A (Chức vụ: Giám Đốc)"
            let extraInfo = customFields.length > 0 && customFields[0].key && customFields[0].value
                ? ` (${customFields[0].key}: ${customFields[0].value})` 
                : "";
            payload.note = `Ký văn bản: ${metaData.signerName}${extraInfo}`;

        } else if (docType === "diploma") {
            // --- B. BẰNG CẤP ---
            payload.data = {
                name: metaData.studentName,
                id: metaData.studentId,
                degree: metaData.degree, 
                no: metaData.certNo
            };
            payload.note = `Bằng/Degree: ${metaData.degree} - ${metaData.studentName}`;

        } else if (docType === "contract") {
            // --- C. HỢP ĐỒNG ---
            payload.data = {
                no: metaData.contractNo,
                partner: metaData.partner,
                val: metaData.value,
                curr: metaData.currency
            };
            payload.note = `HĐ/Contract: ${metaData.contractNo} - ${metaData.partner}`;

        } else if (docType === "official_letter") {
            // --- A. CÔNG VĂN ---
            payload.data = {
                ref_no: metaData.letterRefNo,
                issuer: metaData.letterIssuer,
                recv: metaData.letterReceiver,
                subj: metaData.letterSubject
            };
            payload.note = `CV/Letter: ${metaData.letterRefNo} - ${metaData.letterSubject}`;

        } else {
            // --- D. VĂN BẢN THƯỜNG ---
            payload.note = metaData.note;
        }

        return JSON.stringify(payload);
    }

    async function handleSign() {
        if (!filePath) {
            statusMsg = "Vui lòng chọn file cần ký!";
            isError = true; return;
        }

        // Validate cơ bản cho Custom Doc
        if (docType === 'custom_doc' && !metaData.signerName.trim()) {
            statusMsg = "Lỗi: Vui lòng nhập Họ và tên người ký!";
            isError = true; return;
        }

        isSigning = true;
        statusMsg = "Đang xử lý (Hashing & Signing)...";
        progress = 0;
        isError = false;

        try {
            const finalInfo = prepareMetadata();
            await SignFileAction(filePath, finalInfo, signMode);

            if (signMode == 1) {
                statusMsg = "✅ Đã tạo file chữ ký rời (.sig) thành công!";
            } else if (signMode == 2) {
                statusMsg = "✅ Ký số lai (Hybrid) thành công!";
            } else if (signMode == 0) {
                statusMsg = "✅ Ký số nhúng (Embedded) thành công!";
            }
            progress = 100;
        } catch (err) {
            statusMsg = "Lỗi: " + err;
            isError = true;
        } finally {
            isSigning = false;
        }
    }

    async function handleBlockchainRegister() {
        if (!filePath) return;
        
        if (!confirm("Bạn có chắc chắn muốn ghi dấu ấn file này lên Blockchain không?\n(Hành động này không thể hoàn tác)")) return;

        isRegistering = true;
        statusMsg = "Đang kết nối Blockchain...";
        
        try {
            // Gọi xuống Backend
            let hash = await RegisterFileOnChain(filePath);
            txHash = hash;
            statusMsg = "🚀 Gửi thành công! Transaction Hash: " + hash;
            alert("Thành công! File đã được chứng thực trên Blockchain.");
        } catch (e) {
            statusMsg = "Lỗi Blockchain: " + e;
            alert("Lỗi: " + e);
        } finally {
            isRegistering = false;
        }
    }
</script>

<div class="tab-container">
    <h2>✒️ Ký Số (Sign Document)</h2>
    
    <div class="card">
        <div class="form-group">
            <label>1. Chọn File (Select File)</label>
            <div class="file-input">
                <input type="text" readonly bind:value={filePath} placeholder="Chưa chọn file..." />
                <button on:click={handleBrowse}>📂 Chọn...</button>
            </div>
        </div>

        <div class="form-group">
            <label>2. Chế độ Ký (Signing Mode)</label>
            <div class="radio-group">
                <label class="radio-item {signMode === 0 ? 'selected' : ''}">
                    <input type="radio" bind:group={signMode} value={0} />
                    <span class="radio-label">
                        <strong>🖇️ Nhúng (Embedded)</strong>
                        <span class="desc">Gọn nhẹ, 1 file duy nhất.</span>
                    </span>
                </label>

                <label class="radio-item {signMode === 1 ? 'selected' : ''}">
                    <input type="radio" bind:group={signMode} value={1} />
                    <span class="radio-label">
                        <strong>📑 Rời (Detached)</strong>
                        <span class="desc">Giữ nguyên file gốc.</span>
                    </span>
                </label>

                <label class="radio-item {signMode === 2 ? 'selected' : ''}">
                    <input type="radio" bind:group={signMode} value={2} />
                    <span class="radio-label">
                        <strong>🛡️ Lai (Hybrid)</strong>
                        <span class="desc">Bảo mật kép (Nhúng + Rời).</span>
                    </span>
                </label>
            </div>
        </div>

        <div class="form-group">
            <label>2. Thông tin Ký (Metadata)</label>
            
            <div class="form-container-inner">
                <div class="row-mb">
                    <label class="sub-label">Loại văn bản (Document Type)</label>
                    <select bind:value={docType} class="type-select">
                        <option value="custom_doc">✍️ Ký văn bản (Tùy chỉnh / Custom)</option>
                        <option value="official_letter">🏛️ Công văn / Thông báo (Official Letter)</option>
                        <option value="diploma">🎓 Bằng cấp / Chứng chỉ (Diploma)</option>
                        <option value="contract">🤝 Hợp đồng kinh tế (Contract)</option>
                        <option value="generic">📝 Văn bản thường (Generic)</option>
                    </select>
                </div>

                <div class="dynamic-area">
                    
                    {#if docType === 'custom_doc'}
                        <div class="row-mb">
                            <label class="sub-label">Họ và tên người ký (Full Name) <span class="required">*</span></label>
                            <input type="text" placeholder="VD: Nguyễn Văn A" bind:value={metaData.signerName} />
                        </div>

                        <label class="sub-label" style="margin-top: 15px;">Thông tin thêm (Additional Fields)</label>
                        
                        {#each customFields as field, i}
                            <div class="row dynamic-row">
                                <div class="col-40">
                                    <input type="text" placeholder="Tên trường (VD: Chức vụ)" bind:value={field.key} />
                                </div>
                                <div class="col-50">
                                    <input type="text" placeholder="Giá trị (VD: Giám đốc)" bind:value={field.value} />
                                </div>
                                <div class="col-10">
                                    <button class="btn-remove" on:click={() => removeCustomField(i)} title="Xóa dòng này">🗑️</button>
                                </div>
                            </div>
                        {/each}

                        <button class="btn-add-field" on:click={addCustomField}>+ Thêm trường khác</button>

                    {:else if docType === 'official_letter'}
                        <div class="row">
                            <div class="col">
                                <label class="sub-label">Số / Ký hiệu (Ref No)</label>
                                <input type="text" placeholder="VD: 123/TB-UBND" bind:value={metaData.letterRefNo} />
                            </div>
                            <div class="col">
                                <label class="sub-label">Cơ quan ban hành (Issuer)</label>
                                <input type="text" placeholder="VD: Bộ Tài Chính" bind:value={metaData.letterIssuer} />
                            </div>
                        </div>
                        <div class="row-mb">
                            <label class="sub-label">Nơi nhận (Receiver)</label>
                            <input type="text" placeholder="VD: Công ty ABC..." bind:value={metaData.letterReceiver} />
                        </div>
                        <div class="row-mb">
                            <label class="sub-label">Trích yếu / Chủ đề (Subject)</label>
                            <textarea rows="2" placeholder="VD: Về việc hướng dẫn thực hiện..." bind:value={metaData.letterSubject}></textarea>
                        </div>

                    {:else if docType === 'diploma'}
                        <div class="row">
                            <div class="col">
                                <label class="sub-label">Họ tên người nhận (Name)</label>
                                <input type="text" placeholder="NGUYEN VAN A" bind:value={metaData.studentName} />
                            </div>
                            <div class="col">
                                <label class="sub-label">Mã số / ID (Student ID)</label>
                                <input type="text" placeholder="B2015..." bind:value={metaData.studentId} />
                            </div>
                        </div>
                        <div class="row">
                            <div class="col">
                                <label class="sub-label">Học vị (Degree)</label>
                                <select bind:value={metaData.degree}>
                                    <option value="BSc">Cử nhân (Bachelor)</option>
                                    <option value="Eng">Kỹ sư (Engineer)</option>
                                    <option value="MSc">Thạc sỹ (Master)</option>
                                    <option value="PhD">Tiến sỹ (Doctor)</option>
                                    <option value="Cert">Chứng chỉ (Certificate)</option>
                                </select>
                            </div>
                            <div class="col">
                                <label class="sub-label">Số hiệu bằng (Cert No)</label>
                                <input type="text" placeholder="QA-2025-..." bind:value={metaData.certNo} />
                            </div>
                        </div>

                    {:else if docType === 'contract'}
                        <div class="row">
                            <div class="col">
                                <label class="sub-label">Số Hợp Đồng (Contract No)</label>
                                <input type="text" placeholder="HĐ-2025/..." bind:value={metaData.contractNo} />
                            </div>
                            <div class="col">
                                <label class="sub-label">Đối tác (Partner)</label>
                                <input type="text" placeholder="Công ty B..." bind:value={metaData.partner} />
                            </div>
                        </div>
                        <div class="row">
                            <div class="col-70">
                                <label class="sub-label">Giá trị (Value)</label>
                                <input type="number" bind:value={metaData.value} />
                            </div>
                            <div class="col-30">
                                <label class="sub-label">Tiền tệ</label>
                                <select bind:value={metaData.currency}>
                                    <option value="VND">VND</option>
                                    <option value="USD">USD</option>
                                </select>
                            </div>
                        </div>

                    {:else}
                        <div class="row-mb">
                            <label class="sub-label">Ghi chú (Note)</label>
                            <textarea rows="4" placeholder="Nhập ghi chú tùy ý..." bind:value={metaData.note}></textarea>
                        </div>
                    {/if}
                </div>
            </div>
        </div>

        <button class="btn-action" on:click={handleSign} disabled={isSigning}>
            {isSigning ? `Đang xử lý... ${progress.toFixed(1)}%` : "🖋️ KÝ NGAY (SIGN NOW)"}
        </button>

        <div class="blockchain-area" style="margin-top: 20px; border-top: 1px solid #444; padding-top: 20px;">
            <h3>🌐 Blockchain Proof (Bằng chứng số)</h3>
            <p style="color: #aaa; font-size: 0.9rem; margin-bottom: 10px;">
                Lưu dấu vân tay (Hash) của tài liệu này lên mạng Private Blockchain để chứng minh sự tồn tại và tính toàn vẹn.
            </p>
            
            {#if txHash}
                <div class="tx-success">
                    <p>✅ Đã tồn tại trên Chain!</p>
                    <code style="display:block; word-break:break-all; font-size: 0.8rem; margin-top:5px;">TX: {txHash}</code>
                </div>
            {:else}
                <button class="btn-chain" on:click={handleBlockchainRegister} disabled={isSigning || isRegistering}>
                    {isRegistering ? "⏳ Đang khai thác..." : "🔗 ĐƯA LÊN BLOCKCHAIN"}
                </button>
            {/if}
        </div>

        {#if isSigning || progress > 0}
            <div class="progress-bar-container">
                <div class="progress-bar" style="width: {progress}%"></div>
            </div>
        {/if}
    </div>

    {#if statusMsg}
        <p class="status {isError ? 'error' : 'success'}">{statusMsg}</p>
    {/if}
</div>

<style>
    .tab-container { padding: 20px; max-width: 700px; margin: 0 auto; }
    .card { background: #2a2a2a; padding: 20px; border-radius: 8px; border: 1px solid #444; }
    
    .form-group { margin-bottom: 20px; text-align: left; }
    label { display: block; margin-bottom: 8px; color: #00d2ff; font-weight: bold; font-size: 1rem; }
    
    .required { color: #ff4d4d; margin-left: 3px; }

    /* File Input */
    .file-input { display: flex; gap: 5px; }
    .file-input input { flex: 1; padding: 10px; background: #111; border: 1px solid #555; color: white; border-radius: 4px; }
    .file-input button { padding: 10px; cursor: pointer; background: #444; color: white; border: none; border-radius: 4px; }
    .file-input button:hover { background: #555; }

    /* [NEW] Style cho Radio Group */
    .radio-group { display: flex; gap: 15px; }
    .radio-item {
        flex: 1; display: flex; align-items: flex-start; gap: 10px;
        background: #222; padding: 15px; border: 1px solid #444; border-radius: 6px; cursor: pointer; transition: 0.2s;
    }
    .radio-item:hover { background: #2c2c2c; border-color: #666; }
    .radio-item.selected { background: rgba(0, 210, 255, 0.1); border-color: #00d2ff; }
    .radio-item input { width: auto; margin-top: 5px; cursor: pointer; }
    .radio-label { display: flex; flex-direction: column; }
    .radio-label strong { color: #fff; font-size: 0.95rem; }
    .radio-label .desc { font-size: 0.8rem; color: #888; margin-top: 3px; line-height: 1.3; }

    /* Inner Form Styling */
    .form-container-inner { background: #222; padding: 15px; border: 1px solid #333; border-radius: 6px; }
    .sub-label { color: #aaa; font-size: 0.85rem; margin-bottom: 4px; font-weight: normal; }
    .row { display: flex; gap: 15px; margin-bottom: 10px; }
    .row-mb { margin-bottom: 10px; }
    .col { flex: 1; }
    .col-70 { flex: 0.7; }
    .col-30 { flex: 0.3; }

    /* Dynamic Form Styling */
    .dynamic-row { align-items: center; margin-bottom: 8px; }
    .col-40 { flex: 0.4; }
    .col-50 { flex: 0.5; }
    .col-10 { flex: 0.1; display: flex; justify-content: flex-end; }
    
    .btn-remove { background: #222; color: #ff4d4d; border: 1px solid #444; padding: 8px 12px; border-radius: 4px; cursor: pointer; transition: 0.2s; }
    .btn-remove:hover { background: #333; border-color: #ff4d4d; }

    .btn-add-field { background: none; border: 1px dashed #666; color: #00d2ff; width: 100%; padding: 10px; border-radius: 4px; cursor: pointer; margin-top: 5px; font-size: 0.9rem; transition: 0.2s; }
    .btn-add-field:hover { border-color: #00d2ff; background: rgba(0, 210, 255, 0.05); }

    /* Inputs */
    input, select, textarea { 
        width: 100%; padding: 10px; background: #111; 
        border: 1px solid #444; color: #fff; border-radius: 4px; box-sizing: border-box; font-family: inherit;
    }
    input:focus, select:focus, textarea:focus { border-color: #00d2ff; outline: none; }
    .type-select { font-weight: bold; color: #fff; background: #1a1a1a; border: 1px solid #00d2ff; }

    .dynamic-area { margin-top: 15px; padding-top: 15px; border-top: 1px dashed #444; }

    /* Action Button */
    .btn-action { width: 100%; padding: 15px; font-size: 1.1rem; font-weight: bold; background: #d400ff; color: white; border: none; border-radius: 4px; cursor: pointer; margin-top: 10px; transition: 0.2s; }
    .btn-action:hover { background: #b000d4; }
    .btn-action:disabled { background: #555; cursor: not-allowed; }

    /* Blockchain Area */
    .btn-chain {
        width: 100%; padding: 12px; background: #2c3e50; color: #00d2ff;
        border: 1px solid #00d2ff; font-weight: bold; cursor: pointer;
        transition: 0.2s; border-radius: 4px;
    }
    .btn-chain:hover { background: #00d2ff; color: #000; }
    .tx-success {
        background: rgba(0, 255, 157, 0.1); border: 1px solid #00ff9d;
        padding: 10px; border-radius: 4px; color: #00ff9d; text-align: center;
    }

    /* Progress Bar */
    .progress-bar-container { width: 100%; height: 8px; background: #111; border-radius: 4px; margin-top: 15px; overflow: hidden; }
    .progress-bar { height: 100%; background: #00ff9d; transition: width 0.2s; }

    /* Status */
    .status { margin-top: 15px; font-weight: bold; text-align: center; }
    .success { color: #00ff9d; }
    .error { color: #ff4d4d; }
</style>