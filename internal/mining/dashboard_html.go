package mining

// DashboardHTML menyajikan halaman utama kontrol pertambangan personal pemain.
const DashboardHTML = `<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Abdul Bot — Mining Core</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=Share+Tech+Mono&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-base: #060913;
            --bg-surface: #0c1222;
            --bg-panel: #141d34;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --accent-purple: #a855f7;
            --accent-purple-glow: rgba(168, 85, 247, 0.4);
            --accent-emerald: #10b981;
            --accent-emerald-glow: rgba(16, 185, 129, 0.4);
            --accent-rose: #f43f5e;
            --border-color: #1e293b;
            --font-main: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            --font-mono: 'Share Tech Mono', monospace;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: var(--font-main);
            background-color: var(--bg-base);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            overflow-x: hidden;
            background-image: 
                radial-gradient(circle at 10% 20%, rgba(168, 85, 247, 0.05) 0%, transparent 40%),
                radial-gradient(circle at 90% 80%, rgba(16, 185, 129, 0.05) 0%, transparent 40%);
        }

        /* HEADER */
        header {
            background-color: rgba(12, 18, 34, 0.8);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--border-color);
            padding: 1.25rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: sticky;
            top: 0;
            z-index: 100;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            font-size: 1.35rem;
            font-weight: 800;
            letter-spacing: -0.03em;
        }

        .logo-icon {
            font-size: 2rem;
            background: linear-gradient(135deg, var(--accent-purple), var(--accent-emerald));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            display: inline-block;
            animation: pulse-glow 3s infinite ease-in-out;
        }

        @keyframes pulse-glow {
            0%, 100% { filter: drop-shadow(0 0 2px var(--accent-purple-glow)); }
            50% { filter: drop-shadow(0 0 10px var(--accent-purple-glow)); }
        }

        .user-nav {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .pub-link {
            color: var(--text-secondary);
            text-decoration: none;
            font-weight: 600;
            font-size: 0.9rem;
            padding: 0.5rem 1rem;
            border-radius: 8px;
            transition: all 0.2s;
            border: 1px solid transparent;
        }

        .pub-link:hover {
            color: var(--text-primary);
            background-color: rgba(255,255,255,0.03);
            border-color: var(--border-color);
        }

        .btn-logout {
            background: none;
            border: 1px solid var(--border-color);
            color: var(--text-secondary);
            font-family: var(--font-main);
            font-weight: 600;
            font-size: 0.85rem;
            padding: 0.5rem 1rem;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s;
        }

        .btn-logout:hover {
            color: var(--accent-rose);
            background-color: rgba(244, 63, 94, 0.05);
            border-color: rgba(244, 63, 94, 0.2);
        }

        /* MAIN WRAPPER */
        .container {
            flex: 1;
            width: 100%;
            max-width: 1100px;
            margin: 0 auto;
            padding: 2.5rem 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 2.5rem;
        }

        /* USER PROFILE HERO */
        .profile-hero {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 24px;
            padding: 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: relative;
            overflow: hidden;
            box-shadow: 0 10px 30px rgba(0,0,0,0.3);
        }

        .profile-hero::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 3px;
            background: linear-gradient(90deg, var(--accent-purple), var(--accent-emerald));
        }

        .user-info {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .user-name {
            font-size: 1.75rem;
            font-weight: 800;
            letter-spacing: -0.02em;
        }

        .user-rank {
            font-size: 0.875rem;
            color: var(--text-secondary);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .rank-badge {
            background-color: rgba(168, 85, 247, 0.15);
            color: #d8b4fe;
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-weight: 700;
            font-size: 0.75rem;
            border: 1px solid rgba(168, 85, 247, 0.2);
        }

        .user-balance-card {
            background-color: rgba(255,255,255,0.02);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1rem 1.75rem;
            text-align: right;
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
        }

        .bal-label {
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            color: var(--text-secondary);
        }

        .bal-value {
            font-family: var(--font-mono);
            font-size: 2.25rem;
            font-weight: 700;
            color: var(--accent-emerald);
            text-shadow: 0 0 10px rgba(16, 185, 129, 0.2);
        }

        /* RIG SECTION */
        .section-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 0.5rem;
        }

        .section-title {
            font-size: 1.25rem;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .section-desc {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-top: -1.75rem;
        }

        .rig-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: 1.5rem;
        }

        /* RIG CARD */
        .rig-card {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            padding: 1.75rem;
            display: flex;
            flex-direction: column;
            gap: 1.5rem;
            position: relative;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            overflow: hidden;
            box-shadow: 0 4px 20px rgba(0,0,0,0.15);
        }

        .rig-card:hover {
            transform: translateY(-4px);
            border-color: rgba(168, 85, 247, 0.3);
            box-shadow: 0 12px 30px rgba(168, 85, 247, 0.1);
        }

        .rig-card.broken:hover {
            border-color: rgba(244, 63, 94, 0.3);
            box-shadow: 0 12px 30px rgba(244, 63, 94, 0.1);
        }

        .rig-badge {
            position: absolute;
            top: 0;
            right: 0;
            background: linear-gradient(135deg, var(--accent-purple), #7c3aed);
            color: white;
            font-size: 0.65rem;
            font-weight: 800;
            padding: 0.35rem 1rem;
            border-radius: 0 0 0 12px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .rig-card.broken .rig-badge {
            background: var(--accent-rose);
        }

        .rig-info {
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
        }

        .rig-name {
            font-size: 1.25rem;
            font-weight: 700;
            letter-spacing: -0.01em;
        }

        .rig-eff {
            font-size: 0.75rem;
            color: var(--text-secondary);
            font-weight: 600;
        }

        /* LIVE ROLLING COUNTER */
        .live-counter-box {
            background-color: rgba(0,0,0,0.2);
            border: 1px solid rgba(255,255,255,0.02);
            border-radius: 12px;
            padding: 0.85rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .counter-label {
            font-size: 0.7rem;
            text-transform: uppercase;
            font-weight: 700;
            letter-spacing: 0.05em;
            color: var(--text-secondary);
        }

        .counter-value {
            font-family: var(--font-mono);
            font-size: 1.6rem;
            font-weight: 700;
            color: var(--accent-purple);
            text-shadow: 0 0 10px rgba(168, 85, 247, 0.3);
        }

        .counter-value.ticking-up {
            animation: text-pulse 1s infinite alternate;
        }

        @keyframes text-pulse {
            from { text-shadow: 0 0 5px rgba(168, 85, 247, 0.3); }
            to { text-shadow: 0 0 15px rgba(168, 85, 247, 0.6); }
        }

        /* STATS PROGRESS BARS */
        .rig-stats {
            display: flex;
            flex-direction: column;
            gap: 0.85rem;
        }

        .stat-row {
            display: flex;
            flex-direction: column;
            gap: 0.35rem;
        }

        .stat-label-row {
            display: flex;
            justify-content: space-between;
            font-size: 0.75rem;
            font-weight: 600;
            color: var(--text-secondary);
        }

        .stat-val-hl {
            color: var(--text-primary);
        }

        .progress-bar-container {
            width: 100%;
            height: 6px;
            background-color: var(--border-color);
            border-radius: 999px;
            overflow: hidden;
        }

        .progress-bar {
            height: 100%;
            border-radius: 999px;
            transition: width 0.5s ease-out;
        }

        .progress-bar.fuel {
            background: linear-gradient(90deg, var(--accent-purple), #a855f7);
            box-shadow: 0 0 5px rgba(168, 85, 247, 0.3);
        }

        .progress-bar.durability {
            background: linear-gradient(90deg, var(--accent-emerald), #059669);
        }

        .rig-card.broken .progress-bar.durability {
            background: var(--accent-rose);
            width: 0% !important;
        }

        /* BLANK RIG SLOT */
        .rig-slot-empty {
            background-color: rgba(12, 18, 34, 0.3);
            border: 2px dashed var(--border-color);
            border-radius: 20px;
            padding: 2.5rem 1.75rem;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            gap: 1rem;
            min-height: 250px;
            cursor: pointer;
            transition: all 0.2s;
        }

        .rig-slot-empty:hover {
            border-color: rgba(168, 85, 247, 0.5);
            background-color: rgba(168, 85, 247, 0.02);
        }

        .empty-icon {
            font-size: 2.5rem;
            color: var(--text-secondary);
            animation: bounce-slow 4s infinite ease-in-out;
        }

        @keyframes bounce-slow {
            0%, 100% { transform: translateY(0); }
            50% { transform: translateY(-8px); }
        }

        .empty-text {
            font-size: 1rem;
            font-weight: 700;
            color: var(--text-secondary);
        }

        .empty-subtext {
            font-size: 0.75rem;
            color: rgba(148, 163, 184, 0.6);
            text-align: center;
            max-width: 200px;
        }

        /* CENTRAL CLAIM ACTION */
        .action-container {
            display: flex;
            justify-content: center;
            margin-top: 1rem;
        }

        .btn-claim-main {
            background: linear-gradient(135deg, var(--accent-emerald), #059669);
            border: none;
            border-radius: 16px;
            color: white;
            font-family: var(--font-main);
            font-size: 1.1rem;
            font-weight: 800;
            letter-spacing: 0.05em;
            padding: 1.1rem 3rem;
            cursor: pointer;
            box-shadow: 0 10px 25px rgba(16, 185, 129, 0.2);
            transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            display: flex;
            align-items: center;
            gap: 0.75rem;
            text-transform: uppercase;
        }

        .btn-claim-main:hover {
            transform: translateY(-2px);
            box-shadow: 0 15px 30px rgba(16, 185, 129, 0.4);
            filter: brightness(1.1);
        }

        .btn-claim-main:active {
            transform: translateY(1px);
        }

        .btn-claim-main:disabled {
            background: var(--border-color);
            color: var(--text-secondary);
            cursor: not-allowed;
            transform: none;
            box-shadow: none;
        }

        /* MODAL BUY SHOP */
        .modal-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(4, 6, 15, 0.8);
            backdrop-filter: blur(8px);
            z-index: 200;
            display: flex;
            justify-content: center;
            align-items: center;
            opacity: 0;
            pointer-events: none;
            transition: opacity 0.3s;
        }

        .modal-overlay.show {
            opacity: 1;
            pointer-events: auto;
        }

        .modal {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 24px;
            width: 90%;
            max-width: 550px;
            padding: 2.25rem;
            display: flex;
            flex-direction: column;
            gap: 1.5rem;
            transform: scale(0.92);
            transition: transform 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.15);
            box-shadow: 0 25px 60px rgba(0,0,0,0.5);
        }

        .modal-overlay.show .modal {
            transform: scale(1);
        }

        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .modal-title {
            font-size: 1.35rem;
            font-weight: 800;
        }

        .modal-close {
            background: none;
            border: none;
            font-size: 1.75rem;
            color: var(--text-secondary);
            cursor: pointer;
            transition: color 0.2s;
        }

        .modal-close:hover {
            color: var(--text-primary);
        }

        .shop-list {
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .shop-item {
            background-color: rgba(255,255,255,0.01);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1.15rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 1rem;
            transition: all 0.2s;
        }

        .shop-item:hover {
            background-color: rgba(255,255,255,0.03);
            border-color: rgba(168, 85, 247, 0.2);
        }

        .item-info {
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
            flex: 1;
        }

        .item-title-row {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .item-name {
            font-weight: 700;
            font-size: 1.05rem;
        }

        .item-desc {
            font-size: 0.75rem;
            color: var(--text-secondary);
            line-height: 1.35;
        }

        .item-meta {
            display: flex;
            gap: 0.75rem;
            font-size: 0.7rem;
            font-weight: 700;
            color: #c084fc;
            margin-top: 0.25rem;
            text-transform: uppercase;
        }

        .item-action {
            display: flex;
            flex-direction: column;
            align-items: flex-end;
            gap: 0.5rem;
        }

        .item-price {
            font-family: var(--font-mono);
            font-weight: 700;
            color: var(--accent-emerald);
            font-size: 1.1rem;
        }

        .btn-buy {
            background-color: var(--accent-purple);
            color: white;
            border: none;
            border-radius: 10px;
            font-family: var(--font-main);
            font-weight: 700;
            font-size: 0.8rem;
            padding: 0.5rem 1rem;
            cursor: pointer;
            transition: all 0.2s;
        }

        .btn-buy:hover:not(:disabled) {
            background-color: #9333ea;
            box-shadow: 0 4px 10px rgba(168, 85, 247, 0.2);
        }

        .btn-buy:disabled {
            background-color: var(--border-color);
            color: rgba(255,255,255,0.15);
            cursor: not-allowed;
        }

        /* TOAST NOTIFICATION */
        .toast {
            position: fixed;
            bottom: 2rem;
            right: 2rem;
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-left: 4px solid var(--accent-purple);
            box-shadow: 0 10px 25px rgba(0,0,0,0.5);
            border-radius: 12px;
            padding: 1rem 1.5rem;
            display: flex;
            align-items: center;
            gap: 0.75rem;
            transform: translateY(150%);
            transition: transform 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            z-index: 300;
        }

        .toast.show {
            transform: translateY(0);
        }

        .toast.success {
            border-left-color: var(--accent-emerald);
        }

        .toast.error {
            border-left-color: var(--accent-rose);
        }

        /* GUIDE SECTION */
        .guide-card {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 24px;
            padding: 2.25rem;
            box-shadow: 0 10px 30px rgba(0,0,0,0.3);
            margin-top: 1rem;
            position: relative;
            overflow: hidden;
        }

        .guide-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 4px;
            height: 100%;
            background: linear-gradient(to bottom, var(--accent-purple), var(--accent-emerald));
        }

        .guide-title {
            font-size: 1.25rem;
            font-weight: 800;
            color: var(--text-primary);
            display: flex;
            align-items: center;
            gap: 0.5rem;
            margin-bottom: 1.25rem;
            letter-spacing: -0.01em;
        }

        .guide-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 1.5rem;
        }

        .guide-item {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .guide-item-title {
            font-size: 0.95rem;
            font-weight: 700;
            color: #cbd5e1;
            display: flex;
            align-items: center;
            gap: 0.35rem;
        }

        .guide-item-desc {
            font-size: 0.8rem;
            color: var(--text-secondary);
            line-height: 1.55;
        }

        /* MOBILE RESPONSIVE */
        @media (max-width: 640px) {
            header {
                padding: 1rem 1.25rem;
            }

            .logo-text {
                display: none;
            }

            .container {
                padding: 1.5rem 1rem;
                gap: 1.75rem;
            }

            .profile-hero {
                flex-direction: column;
                align-items: stretch;
                text-align: center;
                gap: 1.5rem;
            }

            .user-balance-card {
                text-align: center;
            }

            .btn-claim-main {
                width: 100%;
                justify-content: center;
            }

            .modal {
                padding: 1.5rem;
            }

            .shop-item {
                flex-direction: column;
                align-items: stretch;
                gap: 0.85rem;
            }

            .item-action {
                flex-direction: row;
                justify-content: space-between;
                align-items: center;
            }
        }
    </style>
</head>
<body>

    <header>
        <div class="logo">
            <span class="logo-icon">⚡</span>
            <span class="logo-text">ABDUL <span style="color:var(--accent-purple)">MINING</span></span>
        </div>
        <div class="user-nav">
            <a href="/mining/public" class="pub-link" target="_blank">🌐 Dashboard Publik</a>
            <button class="btn-logout" onclick="logout()">LOGOUT</button>
        </div>
    </header>

    <div class="container">
        <!-- PLAYER HERO CARD -->
        <div class="profile-hero">
            <div class="user-info">
                <span class="user-name" id="profileName">Loading...</span>
                <div class="user-rank">
                    <span class="rank-badge" id="profileRank">Rank Loading</span>
                    <span id="profileJID" style="opacity: 0.5; font-size: 0.8rem">JID Loading</span>
                </div>
            </div>
            <div class="user-balance-card">
                <span class="bal-label">Saldo Saat Ini</span>
                <span class="bal-value" id="profileBalance">0</span>
            </div>
        </div>

        <!-- RIG SUB-SECTION -->
        <div>
            <div class="section-header">
                <h2 class="section-title">⚙️ Kompartemen Rig Penambangan</h2>
            </div>
            <span class="section-desc">Maksimal 3 alat tambang aktif. Kelola durabilitas dan refuel secara teratur.</span>
            
            <div class="rig-grid" id="rigContainer" style="margin-top: 1.5rem;">
                <!-- Rig cards injected dynamically -->
            </div>
        </div>

        <!-- CLAIM BUTTON -->
        <div class="action-container">
            <button class="btn-claim-main" id="btnClaimMain" onclick="claimAndRefuel()" disabled>
                <span>🔋 CLAIM & REFUEL ALL RIGS</span>
            </button>
        </div>

        <!-- MINING GUIDE CARD -->
        <div class="guide-card">
            <h3 class="guide-title">📖 PANDUAN PORTAL PENAMBANGAN (MINING CORE)</h3>
            <div class="guide-grid">
                <div class="guide-item">
                    <h4 class="guide-item-title">⚡ Apa itu Virtual Mining Rig?</h4>
                    <p class="guide-item-desc">
                        Sistem pertambangan chip pasif (idle simulator). Anda menyewa unit hardware virtual (rig) menggunakan chip yang Anda miliki. Rig tersebut akan memproduksi chip secara otomatis seiring berjalannya waktu.
                    </p>
                </div>
                <div class="guide-item">
                    <h4 class="guide-item-title">🔋 Sistem Bahan Bakar (Batas AFK 24 Jam)</h4>
                    <p class="guide-item-desc">
                        Tangki bahan bakar (baterai) rig hanya bertahan maksimal <strong>24 jam</strong>. Setelah 24 jam tanpa interaksi, rig akan mati sementara dan berhenti menambang. Klik <strong>CLAIM & REFUEL</strong> secara teratur untuk mentransfer pending koin ke dompet utama sekaligus mengisi ulang bahan bakar rig.
                    </p>
                </div>
                <div class="guide-item">
                    <h4 class="guide-item-title">🛠️ Keausan Unit (Durabilitas / Lifespan)</h4>
                    <p class="guide-item-desc">
                        Setiap rig memiliki batas masa pakai produksi (<em>Max Durability</em>). Begitu rig selesai memproduksi total chip target tersebut, status rig akan berubah menjadi <strong>Broken</strong> (Rusak) secara permanen. Anda perlu menyewa rig baru di slot kosong untuk melanjutkan penambangan.
                    </p>
                </div>
                <div class="guide-item">
                    <h4 class="guide-item-title">⚖️ Penurunan Efisiensi Multi-Rig</h4>
                    <p class="guide-item-desc">
                        Untuk menjaga kestabilan perekonomian bot dari hiperinflasi, efisiensi rig ke-2 dan ke-3 akan menurun: Rig pertama berjalan pada efisiensi <strong>100%</strong>, Rig kedua sebesar <strong>75%</strong>, dan Rig ketiga sebesar <strong>50%</strong>.
                    </p>
                </div>
                <div class="guide-item">
                    <h4 class="guide-item-title">💻 Server & Laptop Safe (Lazy Evaluation)</h4>
                    <p class="guide-item-desc">
                        Kalkulasi penambangan dihitung berdasarkan selisih waktu nyata (timestamp) secara presisi di sisi server. Jika laptop server bot dimatikan pada malam hari, Anda <strong>TIDAK AKAN</strong> kehilangan waktu tambang! Waktu akan langsung terakumulasi tepat saat server aktif kembali.
                    </p>
                </div>
                <div class="guide-item">
                    <h4 class="guide-item-title">🔒 Keamanan Terjamin (Server-Side)</h4>
                    <p class="guide-item-desc">
                        Seluruh sistem diproses di sisi server secara aman dengan database transaksi. Eksploitasi angka visual di browser menggunakan <em>Inspect Element</em> tidak akan memengaruhi saldo nyata Anda di database.
                    </p>
                </div>
            </div>
        </div>
    </div>

    <!-- BUY MODAL SHOP -->
    <div class="modal-overlay" id="buyModal">
        <div class="modal">
            <div class="modal-header">
                <h3 class="modal-title">⚙️ Sewa Hardware Baru</h3>
                <button class="modal-close" onclick="closeBuyModal()">&times;</button>
            </div>
            <p style="font-size:0.8rem; color:var(--text-secondary); margin-top:-0.5rem">
                Penambahan rig ke-2 dan ke-3 memiliki efisiensi menurun masing-masing ke 75% dan 50% untuk mencegah hiperinflasi.
            </p>
            <div class="shop-list" id="shopList">
                <!-- Shop items injected dynamically -->
            </div>
        </div>
    </div>

    <!-- TOAST NOTIFICATION -->
    <div class="toast" id="toast">
        <span id="toastIcon">💡</span>
        <span id="toastMessage">Pesan pemberitahuan.</span>
    </div>

    <script>
        let userData = null;
        let rigsData = [];
        let animationFrameId = null;

        // Inisialisasi awal
        window.addEventListener('DOMContentLoaded', () => {
            fetchStatus();
        });

        // Loop Counter Animasi (Rolling JS Counter)
        function startLiveTicker() {
            if (animationFrameId) {
                cancelAnimationFrame(animationFrameId);
            }

            function tick() {
                let anyActive = false;
                rigsData.forEach((r, idx) => {
                    if (r.is_broken) {
                        const el = document.getElementById('live-counter-' + r.id);
                        if (el) el.innerText = "0";
                        return;
                    }

                    // Hitung durasi realtime sejak LastFuelTime
                    const lastFuel = new Date(r.last_fuel_time);
                    const now = new Date();
                    let elapsedMs = now - lastFuel;
                    
                    // Batasi maksimal 24 jam (86.400.000 ms)
                    const maxMs = 24 * 60 * 60 * 1000;
                    if (elapsedMs > maxMs) {
                        elapsedMs = maxMs;
                    }

                    // Kecepatan per milidetik
                    const speedPerMs = r.effective_speed / (60 * 60 * 1000);
                    let earned = elapsedMs * speedPerMs;

                    // Batasi dengan sisa durabilitas rig
                    const remainingLifespan = r.max_durability - r.chips_mined;
                    if (earned > remainingLifespan) {
                        earned = remainingLifespan;
                    }

                    if (earned < 0) earned = 0;

                    const el = document.getElementById('live-counter-' + r.id);
                    if (el) {
                        // Tampilkan pecahan agar terasa bergerak ("live rolling ticker")
                        el.innerText = "+" + earned.toFixed(3);
                        el.classList.add('ticking-up');
                    }

                    // Update Fuel Indicator Realtime
                    const fuelPct = ((maxMs - elapsedMs) / maxMs) * 100;
                    const fuelPctEl = document.getElementById('fuel-val-' + r.id);
                    const fuelBar = document.getElementById('fuel-bar-' + r.id);
                    if (fuelPctEl) fuelPctEl.innerText = Math.max(0, fuelPct).toFixed(1) + "%";
                    if (fuelBar) fuelBar.style.width = Math.max(0, fuelPct) + "%";

                    anyActive = true;
                });

                animationFrameId = requestAnimationFrame(tick);
            }

            tick();
        }

        // Fetch data dashboard
        function fetchStatus() {
            fetch('/mining/api/status')
                .then(res => {
                    if (res.status === 401) {
                        window.location.href = "/mining/public";
                        return;
                    }
                    return res.json();
                })
                .then(data => {
                    if (!data) return;

                    userData = data.user;
                    rigsData = data.rigs || [];

                    // Render profile
                    document.getElementById('profileName').innerText = userData.Name || "Pemain";
                    document.getElementById('profileRank').innerHTML = data.rank_emoji + " " + data.rank_name;
                    document.getElementById('profileJID').innerText = userData.JID.split('@')[0];
                    document.getElementById('profileBalance').innerText = formatNumber(userData.Balance);

                    // Render Rigs
                    renderRigs();

                    // Start counter rolling
                    startLiveTicker();
                })
                .catch(err => {
                    console.error("Fetch status error:", err);
                    showToast("Gagal memuat status pertambangan.", "error");
                });
        }

        // Render data rig ke Grid UI
        function renderRigs() {
            const container = document.getElementById('rigContainer');
            container.innerHTML = '';

            let hasActiveRigs = false;

            // Render rig yang dimiliki
            rigsData.forEach(r => {
                hasActiveRigs = true;
                const card = document.createElement('div');
                card.className = 'rig-card ' + (r.is_broken ? 'broken' : '');

                const durabilityPct = ((r.max_durability - r.chips_mined) / r.max_durability) * 100;

                let tierTitle = r.tier === 'quantum' ? '🎛️ QUANTUM' : r.tier === 'advanced' ? '🚀 CUDA' : '⚙️ BASIC';

                card.innerHTML = 
                    '<div class="rig-badge">' + tierTitle + ' ' + (r.is_broken ? 'BROKEN' : 'ACTIVE') + '</div>' +
                    '<div class="rig-info">' +
                        '<span class="rig-name">' + (r.tier === 'quantum' ? 'Quantum Core' : r.tier === 'advanced' ? 'CUDA GPU Rig' : 'Basic Drill') + '</span>' +
                        '<span class="rig-eff">Efisiensi Unit: ' + (r.efficiency * 100).toFixed(0) + '% (' + (r.effective_speed).toFixed(1) + '/jam)</span>' +
                    '</div>' +
                    '<div class="live-counter-box">' +
                        '<span class="counter-label">Pending</span>' +
                        '<span class="counter-value" id="live-counter-' + r.id + '">+0.000</span>' +
                    '</div>' +
                    '<div class="rig-stats">' +
                        '<div class="stat-row">' +
                            '<div class="stat-label-row">' +
                                '<span>🔋 Sisa Daya Bahan Bakar</span>' +
                                '<span class="stat-val-hl" id="fuel-val-' + r.id + '">' + r.fuel_percent.toFixed(1) + '%</span>' +
                            '</div>' +
                            '<div class="progress-bar-container">' +
                                '<div class="progress-bar fuel" id="fuel-bar-' + r.id + '" style="width: ' + r.fuel_percent + '%"></div>' +
                            '</div>' +
                        '</div>' +
                        '<div class="stat-row">' +
                            '<div class="stat-label-row">' +
                                '<span>🛠️ Sisa Durabilitas (Lifespan)</span>' +
                                '<span class="stat-val-hl">' + formatNumber(r.max_durability - r.chips_mined) + ' / ' + formatNumber(r.max_durability) + '</span>' +
                            '</div>' +
                            '<div class="progress-bar-container">' +
                                '<div class="progress-bar durability" style="width: ' + durabilityPct + '%"></div>' +
                            '</div>' +
                        '</div>' +
                    '</div>';
                container.appendChild(card);
            });

            // Hitung slot kosong
            const emptySlots = 3 - rigsData.length;
            for (let i = 0; i < emptySlots; i++) {
                const emptyCard = document.createElement('div');
                emptyCard.className = 'rig-slot-empty';
                emptyCard.onclick = openBuyModal;
                emptyCard.innerHTML = 
                    '<span class="empty-icon">⛏️</span>' +
                    '<span class="empty-text">Pasang Hardware Baru</span>' +
                    '<span class="empty-subtext">Ketuk di sini untuk membeli alat tambang di slot ini.</span>';
                container.appendChild(emptyCard);
            }

            // Aktifkan / Nonaktifkan tombol claim
            document.getElementById('btnClaimMain').disabled = rigsData.length === 0;
        }

        // Buka modal belanja rig
        function openBuyModal() {
            document.getElementById('buyModal').classList.add('show');
            renderShopItems();
        }

        function closeBuyModal() {
            document.getElementById('buyModal').classList.remove('show');
        }

        // Render item dagangan di modal shop
        function renderShopItems() {
            const list = document.getElementById('shopList');
            list.innerHTML = '';

            const activeRigCount = rigsData.filter(rg => !rg.is_broken).length;

            const tiers = [
                { key: 'basic', name: 'Basic Drill', cost: 1000, speed: 10, durability: 3000, rank: 'Peasant', minBal: 0, desc: 'Alat bor mekanik dasar. Murah dan stabil.' },
                { key: 'advanced', name: 'CUDA Rig', cost: 5000, speed: 35, durability: 12000, rank: 'Levy', minBal: 5000, desc: 'Rig berkekuatan tinggi menggunakan akselerasi CUDA.' },
                { key: 'quantum', name: 'Quantum Core', cost: 20000, speed: 100, durability: 50000, rank: 'Mercenary', minBal: 20000, desc: 'Tambang tercanggih berbasis komputasi kuantum.' }
            ];

            tiers.forEach(t => {
                const item = document.createElement('div');
                item.className = 'shop-item';

                // hitung efisiensi unit berikutnya
                let nextEff = 1.0;
                if (activeRigCount === 1) nextEff = 0.75;
                else if (activeRigCount >= 2) nextEff = 0.50;

                const nextSpeed = t.speed * nextEff;

                const isLocked = userData.Balance < t.minBal;
                const balanceTooLow = userData.Balance < t.cost;
                const isFull = rigsData.length >= 3;

                let disabledBuy = isLocked || balanceTooLow || isFull;

                item.innerHTML = 
                    '<div class="item-info">' +
                        '<div class="item-title-row">' +
                            '<span class="item-name">' + t.name + '</span>' +
                            (isLocked ? '<span style="color:var(--accent-rose); font-size:0.65rem; font-weight:700">[BUTUH RANK ' + t.rank.toUpperCase() + ']</span>' : '') +
                        '</div>' +
                        '<span class="item-desc">' + t.desc + '</span>' +
                        '<div class="item-meta">' +
                            '<span>⚡ +' + nextSpeed.toFixed(1) + ' chips/jam</span>' +
                            '<span>⏳ Durability: ' + formatNumber(t.durability) + '</span>' +
                        '</div>' +
                    '</div>' +
                    '<div class="item-action">' +
                        '<span class="item-price">' + formatNumber(t.cost) + ' chip</span>' +
                        '<button class="btn-buy" onclick="buyRig(\'' + t.key + '\')" ' + (disabledBuy ? 'disabled' : '') + '>' +
                            (isFull ? 'SLOT PENUH' : balanceTooLow ? 'CHIP KURANG' : 'SEWA NOW') +
                        '</button>' +
                    '</div>';
                list.appendChild(item);
            });
        }

        // Beli rig baru
        function buyRig(tier) {
            fetch('/mining/api/buy', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ tier })
            })
            .then(res => res.json())
            .then(data => {
                if (data.success) {
                    showToast("Sewa rig berhasil dipasang!", "success");
                    closeBuyModal();
                    fetchStatus();
                } else {
                    showToast(data.error || "Gagal menyewa rig.", "error");
                }
            })
            .catch(err => {
                console.error("Buy rig error:", err);
                showToast("Terjadi kesalahan jaringan.", "error");
            });
        }

        // Cairkan koin & isi bahan bakar (Claim & Refuel)
        function claimAndRefuel() {
            const btn = document.getElementById('btnClaimMain');
            btn.disabled = true;
            btn.innerHTML = '<span>⚡ PROCESSING CLAIMS...</span>';

            fetch('/mining/api/claim', { method: 'POST' })
            .then(res => res.json())
            .then(data => {
                btn.innerHTML = '<span>🔋 CLAIM & REFUEL ALL RIGS</span>';
                if (data.success) {
                    if (data.claimed_chips > 0) {
                        showToast('Klaim sukses! *+' + formatNumber(data.claimed_chips) + '* chip telah dikreditkan ke saldo utama Anda!', "success");
                    } else {
                        showToast("Bahan bakar rig berhasil diisi penuh kembali!", "success");
                    }
                    fetchStatus();
                } else {
                    showToast(data.error || "Gagal mencairkan tambang.", "error");
                    btn.disabled = false;
                }
            })
            .catch(err => {
                console.error("Claim error:", err);
                showToast("Terjadi kesalahan jaringan.", "error");
                btn.innerHTML = '<span>🔋 CLAIM & REFUEL ALL RIGS</span>';
                btn.disabled = false;
            });
        }

        // Logout
        function logout() {
            fetch('/mining/api/logout', { method: 'POST' })
            .then(() => {
                window.location.href = "/mining/public";
            });
        }

        // Toast alert helper
        function showToast(message, type = "success") {
            const t = document.getElementById('toast');
            const icon = document.getElementById('toastIcon');
            const msg = document.getElementById('toastMessage');

            t.className = "toast " + type;
            icon.innerText = type === 'success' ? '✅' : '❌';
            msg.innerText = message;

            t.classList.add('show');
            setTimeout(() => {
                t.classList.remove('show');
            }, 4000);
        }

        // Format angka ribuan dengan titik
        function formatNumber(n) {
            return n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".");
        }
    </script>
</body>
</html>
`

// PublicDashboardHTML menyajikan panel pemantauan publik (read-only) untuk melihat rig milik semua pemain.
const PublicDashboardHTML = `<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Abdul Bot — Mining Leaderboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=Share+Tech+Mono&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-base: #060913;
            --bg-surface: #0c1222;
            --bg-panel: #141d34;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --accent-purple: #a855f7;
            --accent-purple-glow: rgba(168, 85, 247, 0.4);
            --accent-emerald: #10b981;
            --border-color: #1e293b;
            --font-main: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            --font-mono: 'Share Tech Mono', monospace;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: var(--font-main);
            background-color: var(--bg-base);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            background-image: 
                radial-gradient(circle at 50% 10%, rgba(168, 85, 247, 0.05) 0%, transparent 40%);
        }

        header {
            background-color: rgba(12, 18, 34, 0.8);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--border-color);
            padding: 1.25rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: sticky;
            top: 0;
            z-index: 100;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            font-size: 1.35rem;
            font-weight: 800;
            letter-spacing: -0.03em;
        }

        .logo-icon {
            font-size: 2rem;
            background: linear-gradient(135deg, var(--accent-purple), var(--accent-emerald));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .container {
            flex: 1;
            width: 100%;
            max-width: 900px;
            margin: 0 auto;
            padding: 2.5rem 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 2rem;
        }

        .hero-desc {
            text-align: center;
            margin-bottom: 1rem;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .hero-desc h1 {
            font-size: 2rem;
            font-weight: 800;
            letter-spacing: -0.02em;
        }

        .hero-desc p {
            color: var(--text-secondary);
            font-size: 0.95rem;
            max-width: 500px;
            margin: 0 auto;
        }

        /* PUBLIC LEADERBOARD CARD */
        .player-row {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            padding: 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 1.25rem;
            box-shadow: 0 4px 15px rgba(0,0,0,0.15);
            transition: border-color 0.2s;
        }

        .player-row:hover {
            border-color: rgba(168, 85, 247, 0.2);
        }

        .player-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid rgba(255,255,255,0.03);
            padding-bottom: 0.85rem;
        }

        .player-identity {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .player-name {
            font-size: 1.15rem;
            font-weight: 700;
            letter-spacing: -0.01em;
        }

        .player-rank {
            background-color: rgba(255,255,255,0.04);
            padding: 0.2rem 0.6rem;
            border-radius: 9999px;
            font-size: 0.7rem;
            font-weight: 700;
            color: var(--text-secondary);
        }

        .player-total-rigs {
            display: flex;
            gap: 0.5rem;
            font-size: 0.8rem;
            color: var(--text-secondary);
        }

        /* PUBLIC MINI RIG LIST */
        .mini-rigs-list {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
            gap: 1rem;
        }

        .mini-rig-badge {
            background-color: rgba(255,255,255,0.01);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 0.85rem 1rem;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
            position: relative;
        }

        .mini-rig-tier {
            font-size: 0.75rem;
            font-weight: 800;
            color: #c084fc;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .mini-rig-tier.basic { color: #cbd5e1; }
        .mini-rig-tier.advanced { color: #a855f7; }
        .mini-rig-tier.quantum { color: var(--accent-emerald); }

        .mini-rig-stats {
            display: flex;
            flex-direction: column;
            font-size: 0.7rem;
            color: var(--text-secondary);
            gap: 0.25rem;
        }

        .stat-line {
            display: flex;
            justify-content: space-between;
        }

        .stat-value {
            color: var(--text-primary);
            font-weight: 600;
        }

        .stat-value.broken {
            color: var(--accent-rose);
        }

        .no-data {
            text-align: center;
            padding: 4rem 2rem;
            color: var(--text-secondary);
            font-size: 1rem;
            border: 1px dashed var(--border-color);
            border-radius: 20px;
        }

        /* MOBILE RESPONSIVE */
        @media (max-width: 640px) {
            header {
                padding: 1rem 1.25rem;
            }

            .container {
                padding: 1.5rem 1rem;
            }

            .player-header {
                flex-direction: column;
                align-items: flex-start;
                gap: 0.5rem;
            }
        }
    </style>
</head>
<body>

    <header>
        <div class="logo">
            <span class="logo-icon">⚡</span>
            <span class="logo-text">ABDUL <span style="color:var(--accent-purple)">MINING</span></span>
        </div>
        <span style="font-size: 0.75rem; font-weight:700; color:var(--text-secondary); background: rgba(168,85,247,0.1); border: 1px solid rgba(168,85,247,0.2); padding: 0.35rem 0.75rem; border-radius: 8px;">
            🌐 PUBLIC MONITOR
        </span>
    </header>

    <div class="container">
        <div class="hero-desc">
            <h1>🌐 Pemantauan Rig Pertambangan</h1>
            <p>Daftar rig pertambangan aktif milik para pemain di grup. Untuk mulai menambang, kirim pesan <b>@bot tambang</b> di WhatsApp Anda!</p>
        </div>

        <div id="leaderboardList" style="display: flex; flex-direction: column; gap: 1.5rem; margin-top: 1rem;">
            <!-- Player rows injected dynamically -->
            <div class="no-data">Sedang memuat data pertambangan...</div>
        </div>
    </div>

    <script>
        window.addEventListener('DOMContentLoaded', () => {
            fetchPublicLeaderboard();
        });

        function fetchPublicLeaderboard() {
            fetch('/mining/api/public-list')
                .then(res => res.json())
                .then(data => {
                    renderLeaderboard(data || []);
                })
                .catch(err => {
                    console.error("Fetch public error:", err);
                    document.getElementById('leaderboardList').innerHTML = '<div class="no-data" style="color:var(--accent-rose)">Gagal memuat data dari server. Silakan coba beberapa saat lagi.</div>';
                });
        }

        function renderLeaderboard(players) {
            const container = document.getElementById('leaderboardList');
            container.innerHTML = '';

            if (players.length === 0) {
                container.innerHTML = '<div class="no-data">Belum ada pemain yang memasang hardware pertambangan. Jadilah yang pertama dengan mengetik <b>@bot tambang</b>!</div>';
                return;
            }

            players.forEach((p, index) => {
                const row = document.createElement('div');
                row.className = 'player-row';

                // hitung total rig aktif dan rusak
                const totalActive = p.rigs.filter(r => !r.is_broken).length;
                const totalBroken = p.rigs.filter(r => r.is_broken).length;

                let rigHTML = '';
                p.rigs.forEach(r => {
                    const durabilityPct = ((r.max_durability - r.chips_mined) / r.max_durability) * 100;
                    const isBroken = r.is_broken;

                    rigHTML += 
                        '<div class="mini-rig-badge">' +
                            '<span class="mini-rig-tier ' + r.tier + '">' + (r.tier === 'quantum' ? 'Quantum Drill' : r.tier === 'advanced' ? 'CUDA Rig' : 'Basic Drill') + '</span>' +
                            '<div class="mini-rig-stats">' +
                                '<div class="stat-line">' +
                                    '<span>Status:</span>' +
                                    '<span class="stat-value ' + (isBroken ? 'broken' : '') + '">' + (isBroken ? '⚠️ RUSAK' : '🟢 RUNNING') + '</span>' +
                                '</div>' +
                                '<div class="stat-line">' +
                                    '<span>Efisiensi:</span>' +
                                    '<span class="stat-value">' + (r.efficiency * 100).toFixed(0) + '%</span>' +
                                '</div>' +
                                '<div class="stat-line">' +
                                    '<span>Hasil Tambang:</span>' +
                                    '<span class="stat-value">' + formatNumber(r.chips_mined) + ' / ' + formatNumber(r.max_durability) + '</span>' +
                                '</div>' +
                            '</div>' +
                        '</div>';
                });

                row.innerHTML = 
                    '<div class="player-header">' +
                        '<div class="player-identity">' +
                            '<span style="font-weight:800; opacity:0.3; font-size:1.1rem">#' + (index+1) + '</span>' +
                            '<span class="player-name">' + escapeHTML(p.name || "Unknown") + '</span>' +
                            '<span class="player-rank">' + (p.rank_emoji || "") + ' ' + (p.rank_name || "Unknown") + '</span>' +
                        '</div>' +
                        '<div class="player-total-rigs">' +
                            '<span>Saldo: <b style="color:var(--accent-emerald)">' + formatNumber(p.balance || 0) + ' chip</b></span>' +
                            '<span style="opacity:0.3">|</span>' +
                            '<span>⚙️ ' + totalActive + ' Rig Aktif ' + (totalBroken > 0 ? ('(' + totalBroken + ' Rusak)') : '') + '</span>' +
                        '</div>' +
                    '</div>' +
                    '<div class="mini-rigs-list">' +
                        rigHTML +
                    '</div>';
                container.appendChild(row);
            });
        }

        function escapeHTML(str) {
            return str.replace(/[&<>'"]/g, 
                tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
            );
        }

        function formatNumber(n) {
            return n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".");
        }
    </script>
</body>
</html>
`
