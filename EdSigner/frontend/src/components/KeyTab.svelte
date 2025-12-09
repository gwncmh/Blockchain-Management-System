<script>
    import { GenerateAndSaveKey, UnlockKey, SelectSaveFile, SelectFile, GetEthAddress } from '../../wailsjs/go/main/App';
    // [FIX 1] Sửa đường dẫn import đúng (thêm ../../)
    import { EventsOn } from '../../wailsjs/runtime/runtime';

    let password = "";
    let confirmPassword = ""; 
    let meta = "";
    let statusMsg = "";
    let isError = false;
    let currentPubKey = "";
    let ethAddress = ""; 
    
    let showPassword = false;
    function toggleShowPassword() {
        showPassword = !showPassword;
    }

    async function handleGenerate() {
        if (!password || password.length < 8) {
            statusMsg = "Mật khẩu phải ít nhất 8 ký tự!";
            isError = true; return;
        }

        if (password !== confirmPassword) {
            statusMsg = "❌ Mật khẩu xác nhận không khớp!";
            isError = true; return;
        }

        try {
            let savePath = await SelectSaveFile("my-secret-key.json");
            if (!savePath) return;

            statusMsg = "Đang tạo khóa crypto...";
            currentPubKey = await GenerateAndSaveKey(password, meta, savePath);
            ethAddress = await GetEthAddress();
            statusMsg = "✅ Đã tạo & Lưu khóa thành công!"; isError = false;
            password = ""; confirmPassword = "";
        } catch (err) {
            statusMsg = "Lỗi: " + err;
            isError = true;
        }
    }

    async function handleUnlock() {
        if (!password) {
            statusMsg = "Vui lòng nhập mật khẩu!";
            isError = true; return;
        }
        try {
            let filePath = await SelectFile();
            if (!filePath) return;

            statusMsg = "Đang kiểm tra và giải mã...";
            
            // Backend tự động xử lý Migrate nếu cần
            let result = await UnlockKey(filePath, password);
            currentPubKey = result;
            ethAddress = await GetEthAddress();

            statusMsg = "✅ Mở khóa thành công! Sẵn sàng ký."; isError = false;
        } catch (err) {
            statusMsg = "Lỗi: " + err;
            isError = true;
        }
    }

    // [NEW] Lắng nghe sự kiện Migration thành công từ Backend
    EventsOn("migration-success", () => {
        alert("🎉 Nâng cấp Bảo mật (Security Upgrade)\n\nHệ thống phát hiện khóa của bạn đang ở định dạng cũ (v1) và đã tự động nâng cấp lên chuẩn v2.\n\nQUAN TRỌNG: Từ nay, địa chỉ ví Blockchain của bạn sẽ được cố định.");
    });
</script>

<div class="tab-container">
    <h2>🔐 Quản lý Khóa (Key Management)</h2>

    {#if currentPubKey}
        <div class="card success-card">
            <h3>🔓 KÉT ĐÃ MỞ (UNLOCKED)</h3>
            
            <p><strong>Ed25519 Public Key (Ký File):</strong></p>
            <code class="pubkey">{currentPubKey}</code>
            
            <div style="margin-top: 15px; padding-top: 15px; border-top: 1px dashed #00ff9d;">
                <p><strong>Ethereum Wallet (Ký Blockchain):</strong></p>
                <div style="display: flex; gap: 10px; align-items: center;">
                    <code class="pubkey" style="color: #00d2ff; flex: 1;">{ethAddress}</code>
                    <button class="btn-copy" on:click={() => navigator.clipboard.writeText(ethAddress)}>Copy</button>
                </div>
                <p class="hint">Copy địa chỉ này gửi cho Admin để được cấp quyền ghi sổ cái.</p>
            </div>

            <button class="btn-reset" on:click={() => { currentPubKey = ""; ethAddress = ""; }}>🔒 Đóng Khóa</button>
        </div>
    {:else}
        <div class="card">
            <div class="form-group">
                <label for="pass-input">Mật khẩu (Password)</label>
                <div class="password-wrapper">
                    {#if showPassword}
                        <input id="pass-input" type="text" bind:value={password} placeholder="Nhập mật khẩu..." />
                    {:else}
                        <input id="pass-input" type="password" bind:value={password} placeholder="Nhập mật khẩu..." />
                    {/if}
                    
                    <button class="eye-btn" on:click={toggleShowPassword} tabindex="-1">
                        {showPassword ? "🙈" : "👁️"}
                    </button>
                </div>
            </div>

            <div class="form-group">
                <label for="confirm-input">Xác nhận mật khẩu (Chỉ dùng khi Tạo Mới)</label>
                <div class="password-wrapper">
                    {#if showPassword}
                        <input id="confirm-input" type="text" bind:value={confirmPassword} placeholder="Nhập lại mật khẩu..." />
                    {:else}
                        <input id="confirm-input" type="password" bind:value={confirmPassword} placeholder="Nhập lại mật khẩu..." />
                    {/if}
                </div>
            </div>
            
            <div class="form-group">
                <label for="meta-input">Ghi chú (Metadata - Chỉ dùng khi Tạo Mới)</label>
                <input id="meta-input" type="text" bind:value={meta} placeholder="VD: Khóa chính công ty..." />
            </div>

            <div class="btn-group">
                <button class="btn btn-primary" on:click={handleGenerate}>➕ Tạo Khóa Mới</button>
                <button class="btn btn-secondary" on:click={handleUnlock}>📂 Mở Khóa Có Sẵn</button>
            </div>
        </div>
    {/if}

    {#if statusMsg}
        <p class="status {isError ? 'error' : 'success'}">{statusMsg}</p>
    {/if}
</div>

<style>
    .tab-container { padding: 20px; max-width: 600px; margin: 0 auto; }
    .card { background: #2a2a2a; padding: 20px; border-radius: 8px; border: 1px solid #444; }
    .success-card { border-color: #00ff9d; background: #1a332a; }
    .form-group { margin-bottom: 15px; text-align: left; }
    label { display: block; margin-bottom: 5px; color: #ccc; font-size: 0.9rem; }
    
    .password-wrapper { position: relative; display: flex; align-items: center; }
    .password-wrapper input { width: 100%; padding-right: 40px; } 
    
    .eye-btn {
        position: absolute; right: 5px; background: none; border: none;
        cursor: pointer; font-size: 1.2rem; padding: 5px; color: #888;
        transition: color 0.2s; z-index: 10;
    }
    .eye-btn:hover { color: #fff; }

    input { width: 100%; padding: 10px; background: #111; border: 1px solid #555; color: white; border-radius: 4px; box-sizing: border-box; }
    
    .btn-group { display: flex; gap: 10px; margin-top: 20px; }
    .btn { flex: 1; padding: 12px; border: none; cursor: pointer; border-radius: 4px; font-weight: bold; transition: 0.2s; }
    .btn-primary { background: #0078d4; color: white; }
    .btn-primary:hover { background: #005a9e; }
    .btn-secondary { background: #444; color: white; }
    .btn-secondary:hover { background: #666; }
    
    .btn-reset { margin-top: 15px; padding: 8px 15px; background: #444; color: #ccc; border: none; border-radius: 4px; cursor: pointer; }
    .btn-reset:hover { background: #555; color: white; }

    .btn-copy { padding: 5px 10px; background: #333; border: 1px solid #555; color: white; border-radius: 4px; cursor: pointer; }
    .btn-copy:hover { background: #444; border-color: #00d2ff; }

    .pubkey { display: block; word-break: break-all; background: #000; padding: 10px; color: #00ff9d; border-radius: 4px; font-family: monospace; }
    .status { margin-top: 15px; font-weight: bold; }
    .success { color: #00ff9d; }
    .error { color: #ff4d4d; }
    .hint { font-size: 0.8rem; color: #aaa; margin-top: 10px; }
</style>