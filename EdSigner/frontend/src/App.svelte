<script>
  import KeyTab from './components/KeyTab.svelte';
  import SignTab from './components/SignTab.svelte';
  import VerifyTab from './components/VerifyTab.svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  // State để chuyển tab
  let activeTab = 'keys'; // keys | sign | verify

  function setTab(tab) {
    activeTab = tab;
  }

  // [NEW] Lắng nghe sự kiện từ Backend
  EventsOn("auto-lock-triggered", () => {
      alert("⚠️ Đã quá 5 phút không hoạt động. Khóa đã tự động đóng để bảo mật.");
      activeTab = 'keys'; // Quay về tab Key
      // Có thể reload lại trang nếu muốn reset sạch sẽ state của các tab con
      // window.location.reload(); 
  });
</script>

<main>
  <nav class="sidebar">
    <div class="logo">
      <h1>EdSigner</h1>
      <span class="version">v2.0 High-Perf</span>
    </div>

    <button class:active={activeTab === 'keys'} on:click={() => setTab('keys')}>
      🔐 Keys
    </button>
    <button class:active={activeTab === 'sign'} on:click={() => setTab('sign')}>
      ✒️ Sign
    </button>
    <button class:active={activeTab === 'verify'} on:click={() => setTab('verify')}>
      🔍 Verify
    </button>
  </nav>

  <div class="content">
    {#if activeTab === 'keys'}
      <KeyTab />
    {:else if activeTab === 'sign'}
      <SignTab />
    {:else if activeTab === 'verify'}
      <VerifyTab />
    {/if}
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    background-color: #1a1a1a;
    color: #e0e0e0;
    overflow: hidden; /* App desktop không scroll body */
  }

  main {
    display: flex;
    height: 100vh;
    width: 100vw;
  }

  /* Sidebar Style */
  .sidebar {
    width: 250px;
    background-color: #111;
    display: flex;
    flex-direction: column;
    padding: 20px 0;
    border-right: 1px solid #333;
  }

  .logo {
    text-align: center;
    margin-bottom: 40px;
  }
  .logo h1 { color: #00d2ff; margin: 0; text-transform: uppercase; letter-spacing: 2px; }
  .version { font-size: 0.8rem; color: #555; }

  .sidebar button {
    background: transparent;
    border: none;
    color: #888;
    padding: 15px 20px;
    text-align: left;
    font-size: 1.1rem;
    cursor: pointer;
    transition: 0.2s;
    border-left: 4px solid transparent;
  }

  .sidebar button:hover {
    background: #1f1f1f;
    color: white;
  }

  .sidebar button.active {
    background: #252525;
    color: #00d2ff;
    border-left-color: #00d2ff;
    font-weight: bold;
  }

  /* Content Style */
  .content {
    flex: 1;
    background-color: #1e1e1e;
    overflow-y: auto; /* Scroll nội dung nếu dài */
    position: relative;
  }
</style>