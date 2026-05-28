package trading

// DashboardHTML menyajikan halaman utama kontrol trading simulator personal pemain.
const DashboardHTML = `<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Trading Simulator — Dashboard</title>
    <!-- Fonts -->
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-main: #060814;
            --bg-card: #0d1127;
            --bg-card-hover: #121835;
            --border-color: #1e295d;
            --border-glow: rgba(139, 92, 246, 0.2);
            --text-primary: #f3f4f6;
            --text-secondary: #9ca3af;
            --text-muted: #6b7280;
            
            --color-bullish: #10b981;
            --color-bullish-glow: rgba(16, 185, 129, 0.15);
            --color-bearish: #ef4444;
            --color-bearish-glow: rgba(239, 68, 68, 0.15);
            --color-primary: #8b5cf6;
            --color-primary-hover: #a78bfa;
            --color-primary-glow: rgba(139, 92, 246, 0.3);
            
            --color-warning: #f59e0b;
            --color-info: #3b82f6;
            
            --font-main: 'Plus Jakarta Sans', sans-serif;
            --font-mono: 'JetBrains Mono', monospace;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: var(--font-main);
            background-color: var(--bg-main);
            color: var(--text-primary);
            min-height: 100vh;
            overflow-x: hidden;
            display: flex;
            flex-direction: column;
        }

        /* Custom Scrollbar */
        ::-webkit-scrollbar {
            width: 8px;
            height: 8px;
        }
        ::-webkit-scrollbar-track {
            background: var(--bg-main);
        }
        ::-webkit-scrollbar-thumb {
            background: var(--border-color);
            border-radius: 4px;
        }
        ::-webkit-scrollbar-thumb:hover {
            background: var(--color-primary);
        }

        /* Ambient Glow Backgrounds */
        .ambient-glow-1 {
            position: absolute;
            top: -10%;
            left: 20%;
            width: 50vw;
            height: 50vw;
            background: radial-gradient(circle, rgba(139, 92, 246, 0.08) 0%, rgba(0,0,0,0) 70%);
            z-index: -1;
            pointer-events: none;
        }
        .ambient-glow-2 {
            position: absolute;
            bottom: -10%;
            right: 10%;
            width: 40vw;
            height: 40vw;
            background: radial-gradient(circle, rgba(16, 185, 129, 0.05) 0%, rgba(0,0,0,0) 70%);
            z-index: -1;
            pointer-events: none;
        }

        /* Header CSS */
        header {
            background: rgba(13, 17, 39, 0.7);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--border-color);
            padding: 1rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: sticky;
            top: 0;
            z-index: 50;
        }

        .logo-section {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .logo-icon {
            font-size: 1.75rem;
            filter: drop-shadow(0 0 8px var(--color-primary));
            animation: pulse-glow 2s infinite ease-in-out;
        }

        .logo-title {
            font-weight: 800;
            font-size: 1.25rem;
            letter-spacing: 0.5px;
            background: linear-gradient(135deg, #fff 0%, #a78bfa 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .logo-badge {
            font-size: 0.65rem;
            font-weight: 700;
            background: var(--color-primary-glow);
            color: var(--color-primary-hover);
            border: 1px solid var(--color-primary);
            padding: 0.15rem 0.4rem;
            border-radius: 4px;
            text-transform: uppercase;
        }

        .user-nav {
            display: flex;
            align-items: center;
            gap: 1.5rem;
        }

        .wallet-info {
            display: flex;
            align-items: center;
            background: rgba(6, 8, 20, 0.5);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 0.35rem 0.5rem 0.35rem 1rem;
            gap: 1rem;
        }

        .wallet-item {
            display: flex;
            flex-direction: column;
        }

        .wallet-label {
            font-size: 0.65rem;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .wallet-val {
            font-family: var(--font-mono);
            font-weight: 700;
            font-size: 0.9rem;
        }

        .wallet-val.trading {
            color: var(--color-primary-hover);
        }

        .wallet-actions {
            display: flex;
            gap: 0.35rem;
        }

        .btn-wallet {
            padding: 0.35rem 0.75rem;
            font-size: 0.75rem;
            font-weight: 700;
            border-radius: 8px;
            cursor: pointer;
            border: 1px solid transparent;
            transition: all 0.2s ease;
        }

        .btn-deposit {
            background: var(--color-primary);
            color: #fff;
        }

        .btn-deposit:hover {
            background: var(--color-primary-hover);
            box-shadow: 0 0 12px var(--color-primary-glow);
        }

        .btn-withdraw {
            background: transparent;
            border-color: var(--border-color);
            color: var(--text-primary);
        }

        .btn-withdraw:hover {
            border-color: var(--text-secondary);
            background: rgba(255,255,255,0.03);
        }

        .profile-chip {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            background: rgba(139, 92, 246, 0.08);
            border: 1px solid var(--border-color);
            padding: 0.4rem 1rem;
            border-radius: 12px;
        }

        .profile-avatar {
            font-size: 1.25rem;
        }

        .profile-details {
            display: flex;
            flex-direction: column;
        }

        .profile-name {
            font-size: 0.85rem;
            font-weight: 700;
        }

        .profile-rank {
            font-size: 0.7rem;
        }

        .btn-logout {
            background: transparent;
            border: none;
            color: var(--text-secondary);
            cursor: pointer;
            font-size: 1.2rem;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 0.5rem;
            border-radius: 8px;
            transition: all 0.2s ease;
        }

        .btn-logout:hover {
            color: var(--color-bearish);
            background: rgba(239, 68, 68, 0.1);
        }

        /* Main Content Grid */
        main {
            flex: 1;
            padding: 1.5rem 2rem;
            display: grid;
            grid-template-columns: 1fr 340px;
            gap: 1.5rem;
            max-width: 1600px;
            margin: 0 auto;
            width: 100%;
        }

        .left-column {
            display: flex;
            flex-direction: column;
            gap: 1.5rem;
            min-width: 0; /* Prevents flex children from stretching */
        }

        /* Glassmorphism Card Style */
        .glass-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1.25rem;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
            position: relative;
            overflow: hidden;
            transition: border-color 0.3s ease;
        }

        .glass-card:hover {
            border-color: rgba(139, 92, 246, 0.4);
        }

        /* Chart Section Style */
        .chart-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1rem;
        }

        .chart-title-group {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .chart-title {
            font-size: 1rem;
            font-weight: 700;
        }

        .session-badge {
            font-size: 0.7rem;
            font-weight: 700;
            padding: 0.2rem 0.5rem;
            border-radius: 6px;
            background: #1f2937;
            color: var(--text-secondary);
        }

        .session-badge.active {
            background: var(--color-primary-glow);
            color: var(--color-primary-hover);
            border: 1px solid var(--color-primary);
        }

        .chart-ticker-info {
            display: flex;
            gap: 1.5rem;
        }

        .ticker-stat {
            display: flex;
            flex-direction: column;
        }

        .ticker-label {
            font-size: 0.65rem;
            color: var(--text-muted);
            text-transform: uppercase;
        }

        .ticker-val {
            font-family: var(--font-mono);
            font-size: 0.85rem;
            font-weight: 600;
        }

        .ticker-val.bullish { color: var(--color-bullish); }
        .ticker-val.bearish { color: var(--color-bearish); }

        .chart-container {
            position: relative;
            background: #04060f;
            border: 1px solid rgba(255,255,255,0.03);
            border-radius: 12px;
            height: 480px;
            width: 100%;
            display: flex;
            flex-direction: column;
        }

        #candlestickChart {
            flex: 1;
            width: 100%;
        }

        #indicatorChart {
            height: 120px;
            width: 100%;
            border-top: 1px solid var(--border-color);
        }

        /* Play overlay on chart */
        .chart-overlay {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(6, 8, 20, 0.8);
            backdrop-filter: blur(8px);
            z-index: 10;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            gap: 1.5rem;
            border-radius: 12px;
            text-align: center;
            padding: 2rem;
        }

        .overlay-title {
            font-size: 1.5rem;
            font-weight: 800;
            color: #fff;
        }

        .overlay-desc {
            color: var(--text-secondary);
            font-size: 0.9rem;
            max-width: 420px;
            line-height: 1.6;
        }

        .btn-start-trading {
            background: linear-gradient(135deg, var(--color-primary) 0%, #7c3aed 100%);
            color: #fff;
            font-size: 1rem;
            font-weight: 700;
            padding: 0.8rem 2rem;
            border-radius: 12px;
            cursor: pointer;
            border: none;
            box-shadow: 0 4px 20px var(--color-primary-glow);
            transition: all 0.3s ease;
        }

        .btn-start-trading:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 24px rgba(139, 92, 246, 0.5);
        }

        .btn-start-trading:active {
            transform: translateY(0);
        }

        /* Order Panel Styles */
        .order-panel {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 1.5rem;
        }

        .order-inputs {
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .trading-rules {
            display: grid;
            grid-template-columns: repeat(5, minmax(0, 1fr));
            gap: 0.5rem;
            margin-bottom: 1rem;
        }

        .rule-pill {
            background: rgba(6, 8, 20, 0.55);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 0.55rem 0.65rem;
            min-height: 58px;
        }

        .rule-label {
            display: block;
            color: var(--text-muted);
            font-size: 0.62rem;
            font-weight: 800;
            text-transform: uppercase;
            margin-bottom: 0.2rem;
        }

        .rule-value {
            display: block;
            color: var(--text-primary);
            font-family: var(--font-mono);
            font-size: 0.72rem;
            font-weight: 700;
            line-height: 1.25;
        }

        .trading-rules-note {
            color: var(--text-muted);
            font-size: 0.72rem;
            line-height: 1.4;
            margin: -0.4rem 0 1rem;
        }

        .input-group {
            display: flex;
            flex-direction: column;
            gap: 0.35rem;
        }

        .input-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .input-label {
            font-size: 0.75rem;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
        }

        .input-help {
            font-size: 0.7rem;
            color: var(--text-muted);
            font-family: var(--font-mono);
        }

        .input-wrapper {
            position: relative;
            display: flex;
            align-items: center;
        }

        .input-field {
            width: 100%;
            background: rgba(6, 8, 20, 0.6);
            border: 1px solid var(--border-color);
            border-radius: 10px;
            padding: 0.65rem 1rem;
            font-family: var(--font-mono);
            color: #fff;
            font-size: 0.9rem;
            font-weight: 600;
            outline: none;
            transition: all 0.2s ease;
        }

        .input-field:focus {
            border-color: var(--color-primary);
            box-shadow: 0 0 8px var(--border-glow);
        }

        .input-suffix {
            position: absolute;
            right: 1rem;
            font-size: 0.75rem;
            color: var(--text-secondary);
            font-weight: 700;
        }

        .quick-pct-row {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 0.35rem;
            margin-top: 0.25rem;
        }

        .btn-quick-pct {
            background: rgba(255,255,255,0.02);
            border: 1px solid var(--border-color);
            color: var(--text-secondary);
            font-size: 0.7rem;
            font-weight: 700;
            padding: 0.25rem 0;
            border-radius: 6px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .btn-quick-pct:hover {
            border-color: var(--text-secondary);
            color: #fff;
            background: rgba(255,255,255,0.05);
        }

        /* Range Slider */
        .slider-container {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .slider {
            flex: 1;
            -webkit-appearance: none;
            height: 6px;
            border-radius: 3px;
            background: var(--border-color);
            outline: none;
        }

        .slider::-webkit-slider-thumb {
            -webkit-appearance: none;
            appearance: none;
            width: 16px;
            height: 16px;
            border-radius: 50%;
            background: var(--color-primary);
            cursor: pointer;
            box-shadow: 0 0 10px var(--color-primary-glow);
            transition: all 0.2s ease;
        }

        .slider::-webkit-slider-thumb:hover {
            background: var(--color-primary-hover);
            transform: scale(1.2);
        }

        /* Big action buttons */
        .order-actions {
            display: flex;
            flex-direction: column;
            justify-content: center;
            gap: 1rem;
        }

        .btn-trade-action {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 0.75rem;
            height: 56px;
            border-radius: 12px;
            font-size: 1rem;
            font-weight: 800;
            color: #fff;
            border: none;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 4px 12px rgba(0,0,0,0.2);
        }

        .btn-trade-action.long {
            background: linear-gradient(135deg, var(--color-bullish) 0%, #059669 100%);
        }

        .btn-trade-action.long:hover {
            transform: translateY(-2px);
            box-shadow: 0 0 20px var(--color-bullish-glow);
        }

        .btn-trade-action.short {
            background: linear-gradient(135deg, var(--color-bearish) 0%, #dc2626 100%);
        }

        .btn-trade-action.short:hover {
            transform: translateY(-2px);
            box-shadow: 0 0 20px var(--color-bearish-glow);
        }

        .btn-trade-action:active {
            transform: translateY(0);
        }

        .btn-trade-action:disabled {
            background: #1f2937 !important;
            color: var(--text-muted) !important;
            cursor: not-allowed;
            transform: none !important;
            box-shadow: none !important;
        }

        @media (max-width: 1100px) {
            .trading-rules {
                grid-template-columns: repeat(2, minmax(0, 1fr));
            }

            .order-panel {
                grid-template-columns: 1fr;
            }
        }

        @media (max-width: 640px) {
            .trading-rules {
                grid-template-columns: 1fr;
            }
        }

        /* Toggle Button Group */
        .toggle-options {
            display: flex;
            gap: 0.5rem;
            background: rgba(6, 8, 20, 0.6);
            border: 1px solid var(--border-color);
            border-radius: 10px;
            padding: 0.25rem;
        }

        .btn-toggle-opt {
            flex: 1;
            background: transparent;
            border: none;
            color: var(--text-secondary);
            font-size: 0.75rem;
            font-weight: 600;
            padding: 0.4rem 0.5rem;
            border-radius: 6px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .btn-toggle-opt.active {
            background: var(--color-primary-glow);
            color: var(--color-primary-hover);
        }

        /* Right Column (Side Panels) */
        .right-column {
            display: flex;
            flex-direction: column;
            gap: 1.5rem;
        }

        .panel-title {
            font-size: 0.85rem;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.75px;
            margin-bottom: 0.75rem;
            color: var(--text-primary);
            display: flex;
            align-items: center;
            gap: 0.5rem;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 0.5rem;
        }

        /* Live session status panel */
        .live-stat-row {
            display: flex;
            justify-content: space-between;
            font-size: 0.8rem;
            margin-bottom: 0.5rem;
        }

        .live-stat-label {
            color: var(--text-secondary);
        }

        .live-stat-val {
            font-family: var(--font-mono);
            font-weight: 700;
        }

        .progress-bar-container {
            background: #1f2937;
            height: 6px;
            border-radius: 3px;
            width: 100%;
            margin-top: 0.5rem;
            overflow: hidden;
        }

        .progress-bar-fill {
            background: var(--color-primary);
            height: 100%;
            width: 0%;
            box-shadow: 0 0 8px var(--color-primary-glow);
            transition: width 0.3s ease;
        }

        /* News feed styled */
        .news-feed {
            max-height: 180px;
            overflow-y: auto;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
            padding-right: 0.25rem;
        }

        .news-item {
            background: rgba(6, 8, 20, 0.4);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 0.5rem 0.75rem;
            font-size: 0.75rem;
            line-height: 1.4;
        }

        .news-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 0.25rem;
        }

        .news-time {
            font-family: var(--font-mono);
            color: var(--text-muted);
        }

        .news-badge {
            font-weight: 800;
            font-size: 0.6rem;
            text-transform: uppercase;
        }

        .news-badge.bullish { color: var(--color-bullish); }
        .news-badge.bearish { color: var(--color-bearish); }
        .news-badge.neutral { color: var(--text-muted); }

        .news-text {
            color: var(--text-primary);
        }

        /* Positions list */
        .positions-list {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
            max-height: 320px;
            overflow-y: auto;
            padding-right: 0.25rem;
        }

        .position-card {
            background: rgba(6, 8, 20, 0.5);
            border: 1px solid var(--border-color);
            border-radius: 10px;
            padding: 0.75rem;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .position-card.bullish { border-left: 3px solid var(--color-bullish); }
        .position-card.bearish { border-left: 3px solid var(--color-bearish); }

        .pos-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .pos-title-group {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .pos-badge {
            font-size: 0.65rem;
            font-weight: 800;
            padding: 0.1rem 0.35rem;
            border-radius: 4px;
            color: #fff;
            text-transform: uppercase;
        }

        .pos-badge.bullish { background: var(--color-bullish); }
        .pos-badge.bearish { background: var(--color-bearish); }

        .pos-size {
            font-family: var(--font-mono);
            font-size: 0.75rem;
            color: var(--text-secondary);
        }

        .btn-close-pos {
            background: var(--color-bearish-glow);
            border: 1px solid var(--color-bearish);
            color: var(--color-bearish);
            font-size: 0.7rem;
            font-weight: 700;
            padding: 0.2rem 0.5rem;
            border-radius: 4px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .btn-close-pos:hover {
            background: var(--color-bearish);
            color: #fff;
        }

        .pos-details {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 0.35rem 1rem;
            font-size: 0.7rem;
        }

        .pos-detail-item {
            display: flex;
            justify-content: space-between;
        }

        .pos-detail-label {
            color: var(--text-muted);
        }

        .pos-detail-val {
            font-family: var(--font-mono);
            font-weight: 600;
        }

        .pos-pnl-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-top: 1px dashed var(--border-color);
            padding-top: 0.5rem;
            margin-top: 0.25rem;
        }

        .pos-pnl-label {
            font-size: 0.75rem;
            font-weight: 700;
        }

        .pos-pnl-val {
            font-family: var(--font-mono);
            font-size: 0.95rem;
            font-weight: 800;
        }

        .pos-pnl-val.bullish { color: var(--color-bullish); }
        .pos-pnl-val.bearish { color: var(--color-bearish); }

        /* Empty states */
        .empty-state {
            padding: 2rem 1rem;
            text-align: center;
            color: var(--text-muted);
            font-size: 0.75rem;
            border: 1px dashed var(--border-color);
            border-radius: 8px;
            background: rgba(6, 8, 20, 0.2);
        }

        /* History & Stats bottom area */
        .tabs-header {
            display: flex;
            gap: 1rem;
            border-bottom: 1px solid var(--border-color);
            margin-bottom: 1rem;
        }

        .tab-btn {
            background: transparent;
            border: none;
            color: var(--text-secondary);
            font-family: var(--font-main);
            font-size: 0.85rem;
            font-weight: 700;
            padding: 0.5rem 1rem;
            cursor: pointer;
            position: relative;
            transition: all 0.2s ease;
        }

        .tab-btn:hover {
            color: #fff;
        }

        .tab-btn.active {
            color: var(--color-primary-hover);
        }

        .tab-btn.active::after {
            content: '';
            position: absolute;
            bottom: -1px;
            left: 0;
            right: 0;
            height: 2px;
            background: var(--color-primary);
            box-shadow: 0 -2px 8px var(--color-primary-hover);
        }

        .tab-content {
            display: none;
            overflow-x: auto;
        }

        .tab-content.active {
            display: block;
        }

        /* Premium Tables style */
        .trade-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.8rem;
            text-align: left;
        }

        .trade-table th {
            color: var(--text-secondary);
            font-weight: 600;
            padding: 0.75rem 1rem;
            background: rgba(6, 8, 20, 0.3);
            border-bottom: 1px solid var(--border-color);
        }

        .trade-table td {
            padding: 0.75rem 1rem;
            border-bottom: 1px solid rgba(255, 255, 255, 0.02);
            font-weight: 500;
        }

        .trade-table tr:hover {
            background: rgba(255,255,255,0.01);
        }

        .trade-table td.mono {
            font-family: var(--font-mono);
        }

        .pnl-indicator {
            font-weight: 700;
        }
        .pnl-indicator.bullish { color: var(--color-bullish); }
        .pnl-indicator.bearish { color: var(--color-bearish); }

        /* Leaderboard ranks glow styles */
        .rank-gold {
            background: rgba(245, 158, 11, 0.03) !important;
            color: #f59e0b !important;
        }
        .rank-silver {
            background: rgba(156, 163, 175, 0.03) !important;
            color: #d1d5db !important;
        }
        .rank-bronze {
            background: rgba(180, 83, 9, 0.03) !important;
            color: #c2410c !important;
        }
        .rank-badge {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            width: 22px;
            height: 22px;
            border-radius: 50%;
            font-weight: 700;
            font-size: 0.7rem;
        }
        .rank-badge.gold { background: #f59e0b; color: #090d16; }
        .rank-badge.silver { background: #9ca3af; color: #090d16; }
        .rank-badge.bronze { background: #b45309; color: #090d16; }
        .rank-badge.other { border: 1px solid rgba(255,255,255,0.15); color: var(--text-secondary); }

        /* Stats grid bottom */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 1rem;
        }

        .stat-card {
            background: rgba(6, 8, 20, 0.3);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 0.75rem 1rem;
            text-align: center;
        }

        .stat-label {
            font-size: 0.65rem;
            color: var(--text-secondary);
            text-transform: uppercase;
            margin-bottom: 0.25rem;
        }

        .stat-val {
            font-family: var(--font-mono);
            font-size: 1.15rem;
            font-weight: 700;
        }

        /* Modals & Dialogs styling */
        .modal-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(3, 4, 10, 0.85);
            backdrop-filter: blur(10px);
            z-index: 100;
            display: none; /* Flex when active */
            justify-content: center;
            align-items: center;
            padding: 2rem;
            animation: fade-in 0.25s ease-out;
        }

        .modal-overlay.active {
            display: flex;
        }

        .modal-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            max-width: 520px;
            width: 100%;
            padding: 2rem;
            box-shadow: 0 10px 40px rgba(0,0,0,0.6);
            animation: slide-up 0.3s cubic-bezier(0.16, 1, 0.3, 1);
            position: relative;
        }

        .modal-close {
            position: absolute;
            top: 1.25rem;
            right: 1.25rem;
            background: transparent;
            border: none;
            color: var(--text-secondary);
            font-size: 1.5rem;
            cursor: pointer;
            transition: color 0.2s ease;
        }

        .modal-close:hover {
            color: #fff;
        }

        .modal-title {
            font-size: 1.25rem;
            font-weight: 800;
            margin-bottom: 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .modal-desc {
            font-size: 0.85rem;
            color: var(--text-secondary);
            line-height: 1.6;
            margin-bottom: 1.5rem;
        }

        .modal-form {
            display: flex;
            flex-direction: column;
            gap: 1.25rem;
        }

        .modal-form-row {
            display: flex;
            flex-direction: column;
            gap: 0.35rem;
        }

        .modal-action-btn {
            background: var(--color-primary);
            color: #fff;
            font-weight: 700;
            font-size: 0.95rem;
            padding: 0.8rem;
            border-radius: 10px;
            border: none;
            cursor: pointer;
            box-shadow: 0 4px 15px var(--color-primary-glow);
            transition: all 0.2s ease;
        }

        .modal-action-btn:hover {
            background: var(--color-primary-hover);
            box-shadow: 0 6px 20px rgba(139, 92, 246, 0.4);
        }

        /* Multi-step Tutorial Wizard Styles */
        .tutorial-modal {
            max-width: 720px;
        }

        .tutorial-steps-indicator {
            display: flex;
            justify-content: space-between;
            margin-bottom: 2rem;
            position: relative;
            padding: 0 1rem;
        }

        .tutorial-steps-indicator::before {
            content: '';
            position: absolute;
            top: 50%;
            left: 2rem;
            right: 2rem;
            height: 2px;
            background: var(--border-color);
            z-index: 1;
            transform: translateY(-50%);
        }

        .tutorial-step-dot {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            background: var(--bg-card);
            border: 2px solid var(--border-color);
            color: var(--text-secondary);
            font-weight: 700;
            font-size: 0.85rem;
            display: flex;
            align-items: center;
            justify-content: center;
            position: relative;
            z-index: 2;
            transition: all 0.3s ease;
        }

        .tutorial-step-dot.active {
            background: var(--color-primary);
            border-color: var(--color-primary-hover);
            color: #fff;
            box-shadow: 0 0 12px var(--color-primary);
        }

        .tutorial-step-dot.completed {
            background: var(--color-bullish);
            border-color: var(--color-bullish);
            color: #fff;
            box-shadow: 0 0 12px var(--color-bullish-glow);
        }

        .tutorial-content-pane {
            display: none;
            min-height: 240px;
            animation: fade-in 0.3s ease;
        }

        .tutorial-content-pane.active {
            display: block;
        }

        .tut-heading {
            font-size: 1.15rem;
            font-weight: 700;
            margin-bottom: 0.75rem;
            color: #fff;
        }

        .tut-text {
            font-size: 0.85rem;
            color: var(--text-secondary);
            line-height: 1.7;
            margin-bottom: 1rem;
        }

        .tut-card-display {
            background: #04060f;
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 1.25rem;
            margin: 1.25rem 0;
            display: flex;
            gap: 1.5rem;
            align-items: center;
        }

        .tut-visual {
            font-size: 2.5rem;
            min-width: 60px;
            text-align: center;
            filter: drop-shadow(0 0 10px rgba(255,255,255,0.1));
        }

        .tut-visual-info {
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
        }

        .tut-visual-label {
            font-weight: 700;
            font-size: 0.9rem;
            color: var(--color-primary-hover);
        }

        .tut-visual-desc {
            font-size: 0.75rem;
            color: var(--text-secondary);
        }

        .tutorial-footer {
            display: flex;
            justify-content: space-between;
            margin-top: 2rem;
            border-top: 1px solid var(--border-color);
            padding-top: 1.25rem;
        }

        .btn-tutorial-nav {
            background: transparent;
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            font-weight: 700;
            font-size: 0.85rem;
            padding: 0.6rem 1.2rem;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .btn-tutorial-nav:hover {
            background: rgba(255,255,255,0.03);
            border-color: var(--text-secondary);
        }

        .btn-tutorial-nav.next {
            background: var(--color-primary);
            border-color: transparent;
            color: #fff;
        }

        .btn-tutorial-nav.next:hover {
            background: var(--color-primary-hover);
            box-shadow: 0 0 10px var(--color-primary-glow);
        }

        /* Debug Modal Terminal Style */
        .debug-terminal {
            background: #020308 !important;
            border-color: #00ff66 !important;
            font-family: var(--font-mono);
            max-width: 640px !important;
            box-shadow: 0 0 30px rgba(0, 255, 102, 0.15) !important;
        }

        .debug-terminal .modal-title {
            color: #00ff66 !important;
        }

        .terminal-screen {
            background: rgba(0,0,0,0.8);
            border: 1px solid #113311;
            border-radius: 8px;
            padding: 1rem;
            font-size: 0.8rem;
            color: #00ff66;
            min-height: 200px;
            max-height: 320px;
            overflow-y: auto;
            margin-bottom: 1rem;
            line-height: 1.6;
        }

        .terminal-line {
            display: flex;
            gap: 0.5rem;
        }

        .terminal-prompt {
            color: #008833;
        }

        .terminal-input {
            background: transparent;
            border: none;
            outline: none;
            color: #00ff66;
            font-family: var(--font-mono);
            font-size: 0.8rem;
            flex: 1;
        }

        /* Quick Toast Notifications */
        .toast-container {
            position: fixed;
            bottom: 2rem;
            right: 2rem;
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
            z-index: 1000;
        }

        .toast {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-left: 4px solid var(--color-primary);
            border-radius: 8px;
            padding: 0.75rem 1.25rem;
            font-size: 0.8rem;
            font-weight: 600;
            color: #fff;
            box-shadow: 0 4px 15px rgba(0,0,0,0.3);
            display: flex;
            align-items: center;
            gap: 0.75rem;
            transform: translateX(120%);
            transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
            max-width: 320px;
        }

        .toast.show {
            transform: translateX(0);
        }

        .toast.success { border-left-color: var(--color-bullish); }
        .toast.error { border-left-color: var(--color-bearish); }
        .toast.warning { border-left-color: var(--color-warning); }

        /* Animations */
        @keyframes pulse-glow {
            0% { transform: scale(1); filter: drop-shadow(0 0 4px var(--color-primary)); }
            50% { transform: scale(1.05); filter: drop-shadow(0 0 12px var(--color-primary-hover)); }
            100% { transform: scale(1); filter: drop-shadow(0 0 4px var(--color-primary)); }
        }

        @keyframes fade-in {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        @keyframes slide-up {
            from { transform: translateY(20px); opacity: 0; }
            to { transform: translateY(0); opacity: 1; }
        }
    </style>
</head>
<body>
    <div class="ambient-glow-1"></div>
    <div class="ambient-glow-2"></div>

    <!-- Header -->
    <header>
        <div class="logo-section">
            <span class="logo-icon">⚡</span>
            <span class="logo-title">Abdul Trading</span>
            <span class="logo-badge">Simulator</span>
        </div>

        <div class="user-nav">
            <!-- Wallet Chips Info -->
            <div class="wallet-info">
                <div class="wallet-item">
                    <span class="wallet-label">Utama</span>
                    <span id="headerChipBalance" class="wallet-val">-</span>
                </div>
                <div class="wallet-item" style="border-left: 1px solid var(--border-color); padding-left: 1rem;">
                    <span class="wallet-label">Trading</span>
                    <span id="headerTradingBalance" class="wallet-val trading">-</span>
                </div>
                <div class="wallet-actions" style="margin-left: 0.5rem;">
                    <button class="btn-wallet btn-deposit" onclick="openModal('depositModal')">Deposit</button>
                    <button class="btn-wallet btn-withdraw" onclick="openModal('withdrawModal')">WD</button>
                </div>
            </div>

            <!-- Profile Info -->
            <div class="profile-chip">
                <span id="headerRankEmoji" class="profile-avatar">👤</span>
                <div class="profile-details">
                    <span id="headerName" class="profile-name">-</span>
                    <span id="headerRankName" class="profile-rank">-</span>
                </div>
            </div>

            <button class="btn-logout" onclick="logout()" title="Logout Sesi">
                🚪
            </button>
        </div>
    </header>

    <!-- Main Content -->
    <main>
        <!-- Left Side: Chart & Position Controls -->
        <div class="left-column">
            <!-- Main Candle Chart -->
            <div class="glass-card" style="padding: 1.25rem 0.5rem 0.5rem 0.5rem;">
                <div class="chart-header" style="padding: 0 1rem 0.5rem 1rem;">
                    <div class="chart-title-group">
                        <span class="chart-title">BTC/CHIP PERFORMANCES</span>
                        <span id="sessionStatusBadge" class="session-badge">Observasi</span>
                    </div>

                    <div class="chart-ticker-info">
                        <div class="ticker-stat">
                            <span class="ticker-label">Live Price</span>
                            <span id="tickerLivePrice" class="ticker-val">-</span>
                        </div>
                        <div class="ticker-stat">
                            <span class="ticker-label">Open</span>
                            <span id="tickerOpen" class="ticker-val">-</span>
                        </div>
                        <div class="ticker-stat">
                            <span class="ticker-label">High</span>
                            <span id="tickerHigh" class="ticker-val bullish">-</span>
                        </div>
                        <div class="ticker-stat">
                            <span class="ticker-label">Low</span>
                            <span id="tickerLow" class="ticker-val bearish">-</span>
                        </div>
                    </div>
                </div>

                <div class="chart-container">
                    <canvas id="candlestickChart"></canvas>
                    <canvas id="indicatorChart"></canvas>

                    <!-- Chart Start overlay -->
                    <div id="chartOverlay" class="chart-overlay">
                        <div class="overlay-title">Mulai Sesi Trading Baru</div>
                        <div class="overlay-desc">Masing-masing sesi trading terdiri dari 40 lilin observasi pola historis dan 20 lilin trading real-time (setiap candle ditutup dalam 3 detik). Analisa dan buka posisi di saat yang tepat!</div>
                        <button class="btn-start-trading" onclick="startNewSession()">Mulai Trading Sesi Baru</button>
                    </div>
                </div>
            </div>

            <!-- Trade controls panel -->
            <div class="glass-card">
                <div class="panel-title">🛡️ BUAT TRANSAKSI BARU (OPEN POSITION)</div>
                <div class="trading-rules">
                    <div class="rule-pill">
                        <span class="rule-label">Fee</span>
                        <span class="rule-value">0.15% × margin × leverage</span>
                    </div>
                    <div class="rule-pill">
                        <span class="rule-label">Spread</span>
                        <span class="rule-value">Entry +/- 0.15%</span>
                    </div>
                    <div class="rule-pill">
                        <span class="rule-label">Cooldown</span>
                        <span class="rule-value">6 detik setelah open/close</span>
                    </div>
                    <div class="rule-pill">
                        <span class="rule-label">Posisi Aktif</span>
                        <span class="rule-value">Maksimal 1 posisi</span>
                    </div>
                    <div class="rule-pill">
                        <span class="rule-label">Limit Sesi</span>
                        <span class="rule-value">Maksimal 4 transaksi</span>
                    </div>
                </div>
                <div class="trading-rules-note">PnL Bersih sudah otomatis dipotong fee. Spread membuat LONG masuk sedikit di atas harga market dan SHORT sedikit di bawah harga market.</div>
                <div class="order-panel">
                    <!-- Column 1: Inputs -->
                    <div class="order-inputs">
                        <div class="input-group">
                            <div class="input-header">
                                <span class="input-label">Jumlah Margin (Chips)</span>
                                <span id="lblAvailableTradingMargin" class="input-help">Tersedia: - Chips</span>
                            </div>
                            <div class="input-wrapper">
                                <input type="number" id="txtOrderSize" class="input-field" placeholder="Jumlah modal..." min="1" value="100">
                                <span class="input-suffix">CHIP</span>
                            </div>
                            <div class="quick-pct-row">
                                <button class="btn-quick-pct" onclick="setOrderSizePct(0.1)">10%</button>
                                <button class="btn-quick-pct" onclick="setOrderSizePct(0.25)">25%</button>
                                <button class="btn-quick-pct" onclick="setOrderSizePct(0.5)">50%</button>
                                <button class="btn-quick-pct" onclick="setOrderSizePct(1.0)">100%</button>
                            </div>
                        </div>

                        <div class="input-group">
                            <div class="input-header">
                                <span class="input-label">Leverage</span>
                                <span id="lblLeverageMultiplier" class="input-help" style="font-weight: 700; color: var(--color-primary-hover);">5x</span>
                            </div>
                            <div class="slider-container">
                                <span class="input-help">1x</span>
                                <input type="range" id="sliderLeverage" class="slider" min="1" max="50" value="5" oninput="updateLeverageLabel()">
                                <span id="lblMaxLeverageLimit" class="input-help">50x</span>
                            </div>
                        </div>
                    </div>

                    <!-- Column 2: Advanced Stop settings & long/short actions -->
                    <div class="order-inputs">
                        <!-- Advanced Toggles (Stop Loss / Take Profit / Trailing Stop) -->
                        <div class="input-group">
                            <span class="input-label">Stop Orders (Opsional)</span>
                            <div class="toggle-options" style="margin-bottom: 0.5rem;">
                                <button id="btnToggleSL" class="btn-toggle-opt" onclick="toggleStopField('SL')">Stop Loss</button>
                                <button id="btnToggleTP" class="btn-toggle-opt" onclick="toggleStopField('TP')">Take Profit</button>
                                <button id="btnToggleTS" class="btn-toggle-opt" onclick="toggleStopField('TS')">Trailing Stop</button>
                            </div>
                            
                            <!-- SL input -->
                            <div id="wrapperSL" class="input-wrapper" style="display: none; margin-bottom: 0.35rem;">
                                <input type="number" step="0.01" id="txtOrderSL" class="input-field" placeholder="Batas Stop Loss harga...">
                                <span class="input-suffix" style="font-size: 0.65rem; color: var(--color-bearish);">STOP LOSS</span>
                            </div>
                            <!-- TP input -->
                            <div id="wrapperTP" class="input-wrapper" style="display: none; margin-bottom: 0.35rem;">
                                <input type="number" step="0.01" id="txtOrderTP" class="input-field" placeholder="Batas Take Profit harga...">
                                <span class="input-suffix" style="font-size: 0.65rem; color: var(--color-bullish);">TAKE PROFIT</span>
                            </div>
                            <!-- TS input -->
                            <div id="wrapperTS" class="input-wrapper" style="display: none;">
                                <input type="number" step="0.1" id="txtOrderTS" class="input-field" placeholder="Persentase trailing drawdown...">
                                <span class="input-suffix" style="font-size: 0.65rem; color: var(--color-primary-hover);">TRAILING %</span>
                            </div>
                        </div>

                        <!-- Long/Short Action buttons -->
                        <div class="order-actions">
                            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                                <button id="btnBuyLong" class="btn-trade-action long" onclick="openPosition('long')" disabled>
                                    <span>📈</span> LONG
                                </button>
                                <button id="btnSellShort" class="btn-trade-action short" onclick="openPosition('short')" disabled>
                                    <span>📉</span> SHORT
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Right Side: Session Stats, News & Open Positions -->
        <div class="right-column">
            <!-- Active Session Details -->
            <div class="glass-card">
                <div class="panel-title">📊 DETAIL SESI TRADING</div>
                <div class="live-stat-row">
                    <span class="live-stat-label">Sesi ID</span>
                    <span id="sessionDetailID" class="live-stat-val">-</span>
                </div>
                <div class="live-stat-row">
                    <span class="live-stat-label">Sisa Waktu Sesi</span>
                    <span id="sessionCountdown" class="live-stat-val">-</span>
                </div>
                <div class="live-stat-row">
                    <span class="live-stat-label">Candle Terbuka</span>
                    <span id="sessionCandleProgress" class="live-stat-val">-</span>
                </div>
                <div class="progress-bar-container">
                    <div id="sessionProgressFill" class="progress-bar-fill"></div>
                </div>
            </div>

            <!-- Active News Feed -->
            <div class="glass-card">
                <div class="panel-title">📰 SENTIMEN PASAR & BERITA</div>
                <div id="newsFeed" class="news-feed">
                    <div class="empty-state">Belum ada berita aktif. Mulai sesi trading baru untuk melihat sentimen pasar.</div>
                </div>
            </div>

            <!-- Open Positions List -->
            <div class="glass-card" style="flex: 1; display: flex; flex-direction: column;">
                <div class="panel-title">🔑 POSISI AKTIF (OPEN POSITIONS)</div>
                <div id="activePositionsList" class="positions-list" style="flex: 1;">
                    <div class="empty-state">Tidak ada posisi terbuka. Silakan analisa chart dan buka posisi!</div>
                </div>
            </div>
        </div>
    </main>

    <!-- Bottom History & statistics panel -->
    <div style="max-width: 1600px; width: 100%; margin: 0 auto 3rem auto; padding: 0 2rem;">
        <div class="glass-card">
            <div class="tabs-header">
                <button class="tab-btn active" onclick="switchTab('tabHistoryTrades', this)">RIWAYAT TRANSAKSI</button>
                <button class="tab-btn" onclick="switchTab('tabHistorySessions', this)">RIWAYAT SESI</button>
                <button class="tab-btn" onclick="switchTab('tabStats', this)">STATISTIK TRADER</button>
                <button class="tab-btn" onclick="switchTab('tabLeaderboard', this)">🏆 LEADERBOARD</button>
            </div>

            <!-- Tab 1: Trade History -->
            <div id="tabHistoryTrades" class="tab-content active">
                <table class="trade-table">
                    <thead>
                        <tr>
                            <th>Waktu</th>
                            <th>ID</th>
                            <th>Arah</th>
                            <th>Leverage</th>
                            <th>Margin Size</th>
                            <th>Entry Price</th>
                            <th>Exit Price</th>
                            <th>Hasil Status</th>
                            <th>Net Profit</th>
                        </tr>
                    </thead>
                    <tbody id="tblTradeHistoryBody">
                        <tr>
                            <td colspan="9" style="text-align: center; color: var(--text-muted);">Belum ada riwayat transaksi.</td>
                        </tr>
                    </tbody>
                </table>
            </div>

            <!-- Tab 2: Session History -->
            <div id="tabHistorySessions" class="tab-content">
                <table class="trade-table">
                    <thead>
                        <tr>
                            <th>Mulai</th>
                            <th>Sesi ID</th>
                            <th>Tipe Pola</th>
                            <th>Nama Pola</th>
                            <th>Jumlah Trade</th>
                            <th>Status Sesi</th>
                            <th>Total Profit/Loss</th>
                        </tr>
                    </thead>
                    <tbody id="tblSessionHistoryBody">
                        <tr>
                            <td colspan="7" style="text-align: center; color: var(--text-muted);">Belum ada riwayat sesi.</td>
                        </tr>
                    </tbody>
                </table>
            </div>

            <!-- Tab 3: Trader Stats -->
            <div id="tabStats" class="tab-content">
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-label">Win Rate</div>
                        <div id="statWinRate" class="stat-val">- %</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Total Keuntungan Bersih</div>
                        <div id="statTotalPnL" class="stat-val">- Chips</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Jumlah Transaksi</div>
                        <div id="statTotalTrades" class="stat-val">-</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Rata-rata P&L per Trade</div>
                        <div id="statAvgPnL" class="stat-val">-</div>
                    </div>
                </div>
            </div>

            <!-- Tab 4: Leaderboard -->
            <div id="tabLeaderboard" class="tab-content">
                <table class="trade-table">
                    <thead>
                        <tr>
                            <th style="width: 80px; text-align: center;">Rank</th>
                            <th>Trader (Username)</th>
                            <th style="text-align: right;">Saldo Simulator</th>
                            <th style="text-align: right; color: var(--color-bullish);">Total Profit</th>
                            <th style="text-align: right; color: var(--color-bearish);">Total Loss</th>
                            <th style="text-align: center;">Transaksi</th>
                            <th style="text-align: center;">Sesi Selesai</th>
                        </tr>
                    </thead>
                    <tbody id="tblLeaderboardBody">
                        <tr>
                            <td colspan="7" style="text-align: center; color: var(--text-muted);">Memuat data peringkat global...</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <!-- Modals -->
    <!-- Deposit Modal -->
    <div id="depositModal" class="modal-overlay">
        <div class="modal-card">
            <button class="modal-close" onclick="closeModal('depositModal')">×</button>
            <div class="modal-title">💰 Deposit ke Akun Trading</div>
            <p class="modal-desc">Pindahkan chip dari saldo utama kamu ke akun trading simulator. Konversi saldo adalah 1:1 tanpa potongan biaya.</p>
            
            <div class="modal-form">
                <div class="modal-form-row">
                    <div class="input-header">
                        <span class="input-label">Jumlah Deposit</span>
                        <span id="lblDepositMaxLimit" class="input-help">Maksimal: - Chips</span>
                    </div>
                    <div class="input-wrapper">
                        <input type="number" id="txtDepositAmount" class="input-field" placeholder="Min. 500 Chips">
                        <span class="input-suffix">CHIP</span>
                    </div>
                </div>
                <button class="modal-action-btn" onclick="executeDeposit()">Konfirmasi Deposit</button>
            </div>
        </div>
    </div>

    <!-- Withdraw Modal -->
    <div id="withdrawModal" class="modal-overlay">
        <div class="modal-card">
            <button class="modal-close" onclick="closeModal('withdrawModal')">×</button>
            <div class="modal-title">💸 Withdraw ke Saldo Utama</div>
            <p class="modal-desc">Kembalikan profit atau saldo trading simulator ke dompet chip utama kamu. Konversi saldo adalah 1:1 tanpa potongan biaya.</p>
            
            <div class="modal-form">
                <div class="modal-form-row">
                    <div class="input-header">
                        <span class="input-label">Jumlah Withdraw</span>
                        <span id="lblWithdrawMaxLimit" class="input-help">Tersedia: - Chips</span>
                    </div>
                    <div class="input-wrapper">
                        <input type="number" id="txtWithdrawAmount" class="input-field" placeholder="Min. 100 Chips">
                        <span class="input-suffix">CHIP</span>
                    </div>
                </div>
                <button class="modal-action-btn" onclick="executeWithdraw()">Konfirmasi Withdraw</button>
            </div>
        </div>
    </div>

    <!-- Session recap modal -->
    <div id="recapModal" class="modal-overlay">
        <div class="modal-card" style="text-align: center; max-width: 440px;">
            <div id="recapIcon" style="font-size: 4rem; margin-bottom: 0.5rem; animation: pulse-glow 2s infinite;">🏁</div>
            <div id="recapResultTitle" class="modal-title" style="justify-content: center; font-size: 1.5rem;">Sesi Trading Selesai!</div>
            <p id="recapResultDesc" class="modal-desc">Analisa teknikal selesai. Pola chart telah berhasil terselesaikan.</p>
            
            <div class="glass-card" style="background: rgba(6, 8, 20, 0.4); text-align: left; margin-bottom: 1.5rem;">
                <div class="live-stat-row">
                    <span class="live-stat-label">Nama Pola Terbentuk</span>
                    <span id="recapPatternName" class="live-stat-val" style="color: var(--color-primary-hover); font-size: 0.95rem;">-</span>
                </div>
                <div class="live-stat-row">
                    <span class="live-stat-label">Total P&L Hasil Sesi</span>
                    <span id="recapTotalPnL" class="live-stat-val" style="font-size: 1.15rem;">-</span>
                </div>
                <div class="live-stat-row" style="border-top: 1px dashed var(--border-color); padding-top: 0.5rem; margin-top: 0.5rem;">
                    <span class="live-stat-label">Saldo Trading Baru</span>
                    <span id="recapNewBalance" class="live-stat-val">-</span>
                </div>
            </div>
            
            <button class="modal-action-btn" style="width: 100%;" onclick="closeModal('recapModal')">Selesai & Lanjut</button>
        </div>
    </div>

    <!-- Tutorial Modal Wizard -->
    <div id="tutorialModal" class="modal-overlay">
        <div class="modal-card tutorial-modal">
            <button class="modal-close" onclick="closeModal('tutorialModal')">×</button>
            <div class="modal-title">🎓 Akademi Trading Simulator</div>
            
            <!-- step dots -->
            <div class="tutorial-steps-indicator">
                <div class="tutorial-step-dot active" id="tutDot0">1</div>
                <div class="tutorial-step-dot" id="tutDot1">2</div>
                <div class="tutorial-step-dot" id="tutDot2">3</div>
                <div class="tutorial-step-dot" id="tutDot3">4</div>
                <div class="tutorial-step-dot" id="tutDot4">5</div>
                <div class="tutorial-step-dot" id="tutDot5">6</div>
                <div class="tutorial-step-dot" id="tutDot6">7</div>
            </div>

            <!-- step content pages -->
            <!-- Step 1: Candlestick -->
            <div class="tutorial-content-pane active" id="tutPane0">
                <div class="tut-heading">Langkah 1: Memahami Candlestick (Lilin Grafik)</div>
                <p class="tut-text">Candlestick menunjukkan harga dalam jangka waktu tertentu. Setiap candlestick memiliki tubuh dan sumbu:</p>
                <div class="tut-card-display">
                    <span class="tut-visual">🕯️</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label" style="color: var(--color-bullish);">Hijau (Bullish)</span>
                        <span class="tut-visual-desc">Harga penutupan berada di ATAS harga pembukaan. Menandakan dominasi pembeli.</span>
                        <span class="tut-visual-label" style="color: var(--color-bearish); margin-top: 0.5rem;">Merah (Bearish)</span>
                        <span class="tut-visual-desc">Harga penutupan berada di BAWAH harga pembukaan. Menandakan dominasi penjual.</span>
                    </div>
                </div>
                <p class="tut-text">Sumbu tipis di atas/bawah mewakili harga tertinggi dan terendah yang dicapai selama periode tersebut.</p>
            </div>

            <!-- Step 2: Patterns -->
            <div class="tutorial-content-pane" id="tutPane1">
                <div class="tut-heading">Langkah 2: Mengenal Pola Chart (Chart Patterns)</div>
                <p class="tut-text">Di Trading Simulator, pergerakan grafik BUKAN tebakan acak 50/50, melainkan digenerasi berdasarkan 23 pola grafik teknikal asli:</p>
                <div class="tut-card-display">
                    <span class="tut-visual">📊</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label">Pola Reversal (Pembalikan Arah)</span>
                        <span class="tut-visual-desc">Contoh: Double Bottom (Bullish), Head & Shoulders (Bearish). Grafik akan berbalik arah tajam dari tren sebelumnya.</span>
                        <span class="tut-visual-label" style="margin-top: 0.5rem;">Pola Continuation (Penerusan)</span>
                        <span class="tut-visual-desc">Contoh: Bull Flag, Cup and Handle. Grafik akan istirahat sejenak lalu melanjutkan tren awal.</span>
                    </div>
                </div>
                <p class="tut-text">Masing-masing sesi trading memberi kamu 40 lilin observasi. Gunakan fase observasi ini untuk menganalisa pola apa yang sedang terbentuk!</p>
            </div>

            <!-- Step 3: Indicators -->
            <div class="tutorial-content-pane" id="tutPane2">
                <div class="tut-heading">Langkah 3: Menggunakan Indikator Teknikal</div>
                <p class="tut-text">Simulator ini dilengkapi dengan 3 indikator terpopuler yang otomatis terhitung di grafik:</p>
                <div class="tut-card-display">
                    <span class="tut-visual">📉</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label">Simple Moving Average (MA 20)</span>
                        <span class="tut-visual-desc">Garis ungu di grafik candlestick. Menunjukkan tren rata-rata 20 lilin terakhir.</span>
                        <span class="tut-visual-label" style="margin-top: 0.5rem;">Relative Strength Index (RSI)</span>
                        <span class="tut-visual-desc">Menilai apakah pasar sudah jenuh beli (Overbought > 70, siap koreksi) atau jenuh jual (Oversold < 30, siap memantul naik).</span>
                    </div>
                </div>
            </div>

            <!-- Step 4: Long/Short -->
            <div class="tutorial-content-pane" id="tutPane3">
                <div class="tut-heading">Langkah 4: Membuka Posisi (Long vs Short)</div>
                <p class="tut-text">Kamu bisa menghasilkan chip di segala kondisi pasar (naik maupun turun):</p>
                <div class="tut-card-display">
                    <span class="tut-visual">📈</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label" style="color: var(--color-bullish);">LONG (Beli)</span>
                        <span class="tut-visual-desc">Gunakan ini jika kamu memprediksi grafik akan NAIK. Keuntungan bertambah seiring kenaikan harga.</span>
                    </div>
                </div>
                <div class="tut-card-display">
                    <span class="tut-visual">📉</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label" style="color: var(--color-bearish);">SHORT (Jual)</span>
                        <span class="tut-visual-desc">Gunakan ini jika kamu memprediksi grafik akan TURUN. Keuntungan bertambah seiring penurunan harga.</span>
                    </div>
                </div>
            </div>

            <!-- Step 5: Leverage -->
            <div class="tutorial-content-pane" id="tutPane4">
                <div class="tut-heading">Langkah 5: Memahami Daya Ungkit (Leverage)</div>
                <p class="tut-text">Leverage adalah alat pemacu keuntungan. Kamu meminjam kekuatan pasar untuk melipatgandakan margin trading kamu:</p>
                <div class="tut-card-display">
                    <span class="tut-visual">🚀</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label">Pengali Profit & Loss (P&L)</span>
                        <span class="tut-visual-desc">Dengan leverage 10x, margin 100 chip kamu setara dengan kekuatan 1000 chip. Jika harga naik 5%, keuntungan kamu adalah 5% × 10 = 50% (+50 chip)!</span>
                        <span class="tut-visual-label" style="color: var(--color-bearish); margin-top: 0.5rem;">Resiko Likuidasi</span>
                        <span class="tut-visual-desc">Jika kerugian posisi kamu menyentuh 90% dari modal margin kamu, posisi akan otomatis dilikuidasi (margin hangus) untuk menghindari hutang.</span>
                    </div>
                </div>
                <p class="tut-text">Batas leverage maksimal disesuaikan dengan pangkat chip kamu: dari Peasant (5x) hingga King (50x).</p>
            </div>

            <!-- Step 6: Risk Management -->
            <div class="tutorial-content-pane" id="tutPane5">
                <div class="tut-heading">Langkah 6: Manajemen Resiko (Stop Orders)</div>
                <p class="tut-text">Amankan transaksi kamu secara otomatis dengan menggunakan fitur Stop Orders:</p>
                <div class="tut-card-display">
                    <span class="tut-visual">🛡️</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label">Stop Loss (SL)</span>
                        <span class="tut-visual-desc">Otomatis menutup posisi rugi kamu ketika harga menyentuh batas tertentu untuk membatasi kerugian.</span>
                        <span class="tut-visual-label" style="margin-top: 0.5rem;">Take Profit (TP)</span>
                        <span class="tut-visual-desc">Otomatis mengamankan keuntungan posisi kamu begitu menyentuh target profit yang ditentukan.</span>
                        <span class="tut-visual-label" style="margin-top: 0.5rem;">Trailing Stop (TS)</span>
                        <span class="tut-visual-desc">Batas stop dinamis yang mengikuti profit puncak kamu secara otomatis. Menjaga keuntungan agar tidak kembali hilang saat berbalik arah.</span>
                    </div>
                </div>
            </div>

            <!-- Step 7: Practice -->
            <div class="tutorial-content-pane" id="tutPane6">
                <div class="tut-heading">Langkah 7: Sesi Latihan Simulator</div>
                <p class="tut-text">Selamat! Kamu telah menguasai dasar-dasar trading simulator.</p>
                <div class="tut-card-display">
                    <span class="tut-visual">🎮</span>
                    <div class="tut-visual-info">
                        <span class="tut-visual-label" style="color: var(--color-primary-hover);">Mari Mulai Sesi Latihan!</span>
                        <span class="tut-visual-desc">Klik tombol di bawah untuk membuat satu grafik latihan bebas resiko (saldo tidak berkurang/bertambah). Selesaikan satu trade untuk membuka akses trading menggunakan saldo real kamu!</span>
                    </div>
                </div>
            </div>

            <!-- Tutorial Navigation Footer -->
            <div class="tutorial-footer">
                <button class="btn-tutorial-nav" id="btnTutorialPrev" onclick="navTutorial(-1)">Sebelumnya</button>
                <button class="btn-tutorial-nav next" id="btnTutorialNext" onclick="handleNextClick()">Selanjutnya</button>
            </div>
        </div>
    </div>

    <!-- Debug Modal Screen (Terminal) -->
    <div id="debugModal" class="modal-overlay">
        <div class="modal-card debug-terminal">
            <button class="modal-close" style="color: #00ff66;" onclick="closeModal('debugModal')">×</button>
            <div class="modal-title">👾 ABDUL DEV INTERFACE — DEBUG PANEL</div>
            
            <div class="terminal-screen" id="debugTerminalScreen">
                <div class="terminal-line"><span class="terminal-prompt">></span> system init --success</div>
                <div class="terminal-line"><span class="terminal-prompt">></span> type 'help' for available developer commands.</div>
            </div>

            <div class="input-wrapper" style="border: 1px solid #113311; background: rgba(0,0,0,0.5);">
                <span style="color: #008833; padding-left: 1rem; font-weight: 700; font-family: var(--font-mono); font-size: 0.8rem;">$</span>
                <input type="text" id="txtDebugInput" class="terminal-input" style="padding: 0.65rem 0.5rem;" placeholder="Ketik perintah debug..." onkeydown="handleDebugCommand(event)">
            </div>
        </div>
    </div>

    <!-- Toast container -->
    <div id="toastContainer" class="toast-container"></div>

    <!-- Script Game Engine JS -->
    <script>
        // State variables
        var userData = null;
        var activeSession = null;
        var activePositions = [];
        var tutorialStep = 0;
        var tutorialStepsData = [];
        var isTutorialCompleted = false;
        var debugMode = false;
        var debugPassword = "";
        var playInterval = null;
        
        // Resolution data retrieved in advance but hidden from client
        var hiddenResolutionData = [];
        var hiddenIndicators = null;
        var hiddenPatternName = "";
        var currentTickIndex = 0;
        var currentLivePrice = 0;
        var isOpeningPosition = false;
        var tradeCooldownUntil = 0;
        var TRADING_FEE_RATE = 0.0015;
        var TRADE_COOLDOWN_MS = 6000;

        // Canvas setups
        var mainCanvas = document.getElementById('candlestickChart');
        var mainCtx = mainCanvas.getContext('2d');
        var indCanvas = document.getElementById('indicatorChart');
        var indCtx = indCanvas.getContext('2d');

        // Color helper
        var CSS_COLORS = {
            bullish: '#10b981',
            bearish: '#ef4444',
            primary: '#8b5cf6',
            primaryHover: '#a78bfa',
            border: '#1e295d',
            textPrimary: '#f3f4f6',
            textSecondary: '#9ca3af',
            textMuted: '#6b7280',
            bg: '#04060f'
        };

        function isValidMarketPrice(price) {
            return typeof price === 'number' && Number.isFinite(price) && price > 0;
        }

        function getLastVisiblePrice() {
            if (!activeSession || !activeSession.observation || activeSession.observation.length === 0) {
                return 0;
            }
            var lastCandle = activeSession.observation[activeSession.observation.length - 1];
            return lastCandle && isValidMarketPrice(lastCandle.c) ? lastCandle.c : 0;
        }

        function setTradeButtonsDisabled(disabled) {
            document.getElementById('btnBuyLong').disabled = disabled;
            document.getElementById('btnSellShort').disabled = disabled;
        }

        function hasOpenPosition() {
            return activePositions.some(function(p) { return p.status === 'open'; });
        }

        function updateTradeButtons() {
            var disabled = !activeSession || !isValidMarketPrice(currentLivePrice) || isOpeningPosition || hasOpenPosition() || Date.now() < tradeCooldownUntil;
            setTradeButtonsDisabled(disabled);
        }

        function startTradeCooldown() {
            tradeCooldownUntil = Date.now() + TRADE_COOLDOWN_MS;
            updateTradeButtons();
            setTimeout(updateTradeButtons, TRADE_COOLDOWN_MS + 100);
        }

        function estimateTradingFee(pos) {
            if (Number.isFinite(pos.fee) && pos.fee > 0) {
                return pos.fee;
            }
            return Math.ceil(pos.size * pos.leverage * TRADING_FEE_RATE);
        }

        // Resize Canvas to fit wrapper
        function resizeCanvases() {
            var width = mainCanvas.parentElement.clientWidth;
            mainCanvas.width = width;
            mainCanvas.height = mainCanvas.parentElement.clientHeight - 120; // spare for indicators
            indCanvas.width = width;
            indCanvas.height = 120;
            drawChart();
        }

        window.addEventListener('resize', resizeCanvases);

        // Run on startup
        window.addEventListener('DOMContentLoaded', function() {
            fetchStatus();
            fetchHistory();
            
            // Listen for keypress to trigger debug modal (Ctrl + Shift + D)
            window.addEventListener('keydown', function(e) {
                if (e.ctrlKey && e.shiftKey && e.code === 'KeyD') {
                    openModal('debugModal');
                }
            });
        });

        // =====================================================================
        // API FETCH FUNCTIONS
        // =====================================================================

        async function fetchStatus() {
            try {
                var res = await fetch('/trading/api/status');
                if (!res.ok) throw new Error("Gagal mengambil status.");
                var data = await res.json();
                
                userData = data;
                updateHeaderUI();
                
                if (data.active_session && !activeSession) {
                    recoverActiveSession(data.active_session);
                } else if (!activeSession) {
                    document.getElementById('chartOverlay').style.display = 'flex';
                }

                // Check tutorial
                fetchTutorialProgress();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        async function fetchHistory() {
            try {
                // Trade History
                var resTrades = await fetch('/trading/api/history/trades?limit=20');
                if (resTrades.ok) {
                    var trades = await resTrades.json();
                    renderTradeHistory(trades);
                }

                // Session History
                var resSessions = await fetch('/trading/api/history/sessions?limit=10');
                if (resSessions.ok) {
                    var sessions = await resSessions.json();
                    renderSessionHistory(sessions);
                }
            } catch (err) {
                console.error("Gagal mengambil riwayat:", err);
            }
        }

        async function fetchLeaderboard() {
            try {
                var res = await fetch('/trading/api/leaderboard');
                if (res.ok) {
                    var leaderboard = await res.json();
                    renderLeaderboard(leaderboard);
                }
            } catch (err) {
                console.error("Gagal mengambil leaderboard:", err);
            }
        }

        function renderLeaderboard(data) {
            var tbody = document.getElementById('tblLeaderboardBody');
            tbody.innerHTML = '';

            if (!data || data.length === 0) {
                tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--text-muted);">Belum ada data peringkat. Jadilah yang pertama!</td></tr>';
                return;
            }

            data.forEach(function(row) {
                var tr = document.createElement('tr');
                
                // Add podium styling class
                var rankClass = '';
                var badgeClass = 'other';
                if (row.rank === 1) { rankClass = 'rank-gold'; badgeClass = 'gold'; }
                else if (row.rank === 2) { rankClass = 'rank-silver'; badgeClass = 'silver'; }
                else if (row.rank === 3) { rankClass = 'rank-bronze'; badgeClass = 'bronze'; }

                if (rankClass) tr.className = rankClass;

                var isMe = userData && userData.account && userData.account.jid.startsWith(row.username + '@');
                var meLabel = isMe ? ' <span class="session-badge active" style="font-size: 8px; padding: 2px 4px; margin-left: 5px;">KAMU</span>' : '';

                tr.innerHTML = '\n' +
                    '                    <td style="text-align: center; font-weight: 700;"><span class="rank-badge ' + badgeClass + '">' + row.rank + '</span></td>\n' +
                    '                    <td style="font-family: var(--font-main); font-weight: 600;">' + row.username + meLabel + '</td>\n' +
                    '                    <td class="mono" style="text-align: right; font-weight: 700;">' + row.balance.toLocaleString() + ' Chips</td>\n' +
                    '                    <td class="mono" style="text-align: right; color: var(--color-bullish); font-weight: 600;">+' + row.total_profit.toLocaleString() + '</td>\n' +
                    '                    <td class="mono" style="text-align: right; color: var(--color-bearish); font-weight: 600;">-' + row.total_loss.toLocaleString() + '</td>\n' +
                    '                    <td style="text-align: center;">' + row.total_trades + '</td>\n' +
                    '                    <td style="text-align: center;">' + row.total_sessions + '</td>\n' +
                    '                ';
                tbody.appendChild(tr);
            });
        }

        async function fetchTutorialProgress() {
            try {
                var res = await fetch('/trading/api/tutorial/progress');
                if (!res.ok) return;
                var data = await res.json();
                
                isTutorialCompleted = data.complete;
                tutorialStepsData = data.steps;
                tutorialStep = data.progress.completed_steps;
                if (tutorialStep > 6) {
                    tutorialStep = 6;
                }
                
                if (!isTutorialCompleted) {
                    openModal('tutorialModal');
                    updateTutorialUI();
                }
            } catch (e) {
                console.error("Gagal mengambil tutorial:", e);
            }
        }

        // =====================================================================
        // GAME LIFECYCLE
        // =====================================================================

        async function startNewSession() {
            if (playInterval) clearInterval(playInterval);
            setTradeButtonsDisabled(true);
            currentLivePrice = 0;
            document.getElementById('chartOverlay').style.display = 'none';
            showToast("Membuat chart analisis baru...", "info");

            try {
                var res = await fetch('/trading/api/session/start', { method: 'POST' });
                if (!res.ok) {
                    var errData = await res.json();
                    throw new Error(errData.error || "Gagal memulai sesi.");
                }

                var data = await res.json();
                activeSession = data;
                activePositions = [];
                currentTickIndex = 0;
                hiddenResolutionData = [];
                hiddenIndicators = null;
                
                // Clear news feed
                var newsFeed = document.getElementById('newsFeed');
                newsFeed.innerHTML = '';

                // Add observation news
                if (data.news) {
                    data.news.forEach(function(n) {
                        if (n.time < data.obs_candles) {
                            addNewsItem(n);
                        }
                    });
                }

                // UI setup for observation
                document.getElementById('sessionStatusBadge').innerText = "Observasi (TA)";
                document.getElementById('sessionStatusBadge').className = "session-badge";
                document.getElementById('sessionDetailID').innerText = "#" + data.session_id;
                document.getElementById('sessionCountdown').innerText = "Mulai Trading...";
                document.getElementById('sessionCandleProgress').innerText = "40 / 60";
                document.getElementById('sessionProgressFill').style.width = "66.6%";

                var lastObservation = data.observation && data.observation.length > 0 ? data.observation[data.observation.length - 1] : null;
                if (!lastObservation || !isValidMarketPrice(lastObservation.c)) {
                    throw new Error("Harga chart belum siap. Coba mulai ulang sesi.");
                }
                currentLivePrice = lastObservation.c;
                document.getElementById('tickerLivePrice').innerText = currentLivePrice.toFixed(2);
                document.getElementById('tickerOpen').innerText = lastObservation.o.toFixed(2);
                document.getElementById('tickerHigh').innerText = lastObservation.h.toFixed(2);
                document.getElementById('tickerLow').innerText = lastObservation.l.toFixed(2);
                document.getElementById('tickerLivePrice').style.color = lastObservation.c >= lastObservation.o ? CSS_COLORS.bullish : CSS_COLORS.bearish;

                updateTradeButtons();

                // Load resolution in background immediately
                fetchResolution(data.session_id);

                // Initial render of observation chart
                resizeCanvases();
                
                // Fetch status to update balances in case of auto-close from prior sessions
                fetchStatus();
                fetchHistory();

            } catch (err) {
                showToast(err.message, 'error');
                document.getElementById('chartOverlay').style.display = 'flex';
                setTradeButtonsDisabled(true);
            }
        }

        async function fetchResolution(sessionId, resumeTickIndex) {
            try {
                var res = await fetch('/trading/api/session/resolution?id=' + sessionId);
                if (!res.ok) return;
                var data = await res.json();
                
                hiddenResolutionData = data.resolution;
                hiddenIndicators = data.indicators;
                hiddenPatternName = data.pattern_name;
                
                if (resumeTickIndex !== undefined) {
                    // Push all hidden indicators up to resumeTickIndex into activeSession.indicators
                    if (hiddenIndicators && activeSession.indicators) {
                        for (var i = 0; i < resumeTickIndex; i++) {
                            if (hiddenIndicators.MA20 && hiddenIndicators.MA20[i]) {
                                activeSession.indicators.MA20.push(hiddenIndicators.MA20[i]);
                            }
                            if (hiddenIndicators.RSI && hiddenIndicators.RSI[i]) {
                                activeSession.indicators.RSI.push(hiddenIndicators.RSI[i]);
                            }
                            if (hiddenIndicators.MACD && hiddenIndicators.MACD[i]) {
                                activeSession.indicators.MACD.push(hiddenIndicators.MACD[i]);
                                activeSession.indicators.MACDSig.push(hiddenIndicators.MACDSig[i]);
                            }
                        }
                    }

                    currentTickIndex = resumeTickIndex;
                    // Resume ticking!
                    if (currentTickIndex >= hiddenResolutionData.length) {
                        endSession();
                    } else {
                        // Start ticking immediately
                        tick();
                        playInterval = setInterval(tick, 3000);
                    }
                } else {
                    // Start live tick simulation!
                    setTimeout(startResolutionTicks, 2000);
                }
            } catch (err) {
                console.error("Gagal memuat data resolusi:", err);
            }
        }

        function recoverActiveSession(sessionData) {
            if (playInterval) clearInterval(playInterval);
            document.getElementById('chartOverlay').style.display = 'none';
            showToast("Memulihkan sesi trading aktif...", "info");

            activeSession = {
                session_id: sessionData.session_id,
                observation: sessionData.observation,
                news: sessionData.news,
                indicators: sessionData.indicators,
                difficulty: sessionData.difficulty,
                obs_candles: sessionData.obs_candles || 40,
                res_candles: sessionData.res_candles || 20,
                duration: sessionData.duration || 60
            };

            // Push already revealed resolution candles
            if (sessionData.revealed_resolution) {
                sessionData.revealed_resolution.forEach(function(c) {
                    activeSession.observation.push(c);
                });
            }

            // Set currentLivePrice to the last candle's close price to prevent NaN or undefined display issues
            if (activeSession.observation && activeSession.observation.length > 0) {
                currentLivePrice = getLastVisiblePrice();
                if (isValidMarketPrice(currentLivePrice)) {
                    document.getElementById('tickerLivePrice').innerText = currentLivePrice.toFixed(2);
                }
            }

            // Setup news feed for observation
            var newsFeed = document.getElementById('newsFeed');
            newsFeed.innerHTML = '';
            if (sessionData.news) {
                sessionData.news.forEach(function(n) {
                    if (n.time < sessionData.obs_candles) {
                        addNewsItem(n);
                    }
                    if (n.time >= sessionData.obs_candles && n.time < sessionData.obs_candles + sessionData.tick_index) {
                        addNewsItem(n);
                    }
                });
            }

            // Setup active positions
            activePositions = [];
            if (sessionData.positions) {
                activePositions = sessionData.positions.filter(function(p) { return p.status === 'open'; });
            }
            renderActivePositions();

            // Setup UI labels
            document.getElementById('sessionStatusBadge').innerText = "Trading Live";
            document.getElementById('sessionStatusBadge').className = "session-badge active";
            document.getElementById('sessionDetailID').innerText = "#" + sessionData.session_id;
            document.getElementById('sessionCountdown').innerText = "Melanjutkan Sesi...";
            
            var totalPercent = ((40 + sessionData.tick_index) / 60) * 100;
            document.getElementById('sessionCandleProgress').innerText = (40 + sessionData.tick_index) + " / 60";
            document.getElementById('sessionProgressFill').style.width = totalPercent + "%";

            updateTradeButtons();

            // Fetch and resume resolution
            fetchResolution(sessionData.session_id, sessionData.tick_index);

            // Initial render of observation chart
            resizeCanvases();
            fetchHistory();
        }

        function startResolutionTicks() {
            if (playInterval) clearInterval(playInterval);
            
            showToast("📊 FASE TRADING DIMULAI! Pergerakan grafik 20 candle real-time.", "success");
            document.getElementById('sessionStatusBadge').innerText = "Trading Live";
            document.getElementById('sessionStatusBadge').className = "session-badge active";

            currentTickIndex = 0;
            tick(); // First instant tick
            
            playInterval = setInterval(tick, 3000); // 3 seconds per resolution candle!
        }

        async function tick() {
            if (!activeSession || currentTickIndex >= hiddenResolutionData.length) {
                endSession();
                return;
            }

            var nextCandle = hiddenResolutionData[currentTickIndex];
            if (!nextCandle || !isValidMarketPrice(nextCandle.c)) {
                endSession();
                return;
            }
            currentLivePrice = nextCandle.c;

            // Push next candle
            activeSession.observation.push(nextCandle);
            
            // Push indicators
            if (hiddenIndicators && activeSession.indicators) {
                if (hiddenIndicators.MA20 && hiddenIndicators.MA20[currentTickIndex]) {
                    activeSession.indicators.MA20.push(hiddenIndicators.MA20[currentTickIndex]);
                }
                if (hiddenIndicators.RSI && hiddenIndicators.RSI[currentTickIndex]) {
                    activeSession.indicators.RSI.push(hiddenIndicators.RSI[currentTickIndex]);
                }
                if (hiddenIndicators.MACD && hiddenIndicators.MACD[currentTickIndex]) {
                    activeSession.indicators.MACD.push(hiddenIndicators.MACD[currentTickIndex]);
                    activeSession.indicators.MACDSig.push(hiddenIndicators.MACDSig[currentTickIndex]);
                }
            }

            // Update ticker labels
            document.getElementById('tickerLivePrice').innerText = currentLivePrice.toFixed(2);
            document.getElementById('tickerOpen').innerText = nextCandle.o.toFixed(2);
            document.getElementById('tickerHigh').innerText = nextCandle.h.toFixed(2);
            document.getElementById('tickerLow').innerText = nextCandle.l.toFixed(2);
            
            var liveColor = nextCandle.c >= nextCandle.o ? CSS_COLORS.bullish : CSS_COLORS.bearish;
            document.getElementById('tickerLivePrice').style.color = liveColor;

            // Check if there is active news for this tick index
            var globalIndex = activeSession.obs_candles + currentTickIndex;
            if (activeSession.news) {
                activeSession.news.forEach(function(n) {
                    if (n.time === globalIndex) {
                        addNewsItem(n);
                        showToast('📰 BERITA BARU: ' + n.headline, n.sentiment === 'bullish' ? 'success' : (n.sentiment === 'bearish' ? 'error' : 'warning'));
                    }
                });
            }

            // Update countdown & progress
            var remaining = hiddenResolutionData.length - currentTickIndex;
            document.getElementById('sessionCountdown').innerText = (remaining * 3) + ' Detik';
            document.getElementById('sessionCandleProgress').innerText = (globalIndex + 1) + ' / 60';
            
            var progressPct = ((globalIndex + 1) / 60) * 100;
            document.getElementById('sessionProgressFill').style.width = progressPct + '%';

            // Draw chart
            drawChart();

            // Check Stop Orders on Server via API
            checkStopsOnServer(currentLivePrice);

            // Update Positions PnL locally in memory
            updateLocalPositionsPnL();

            currentTickIndex++;
        }

        async function checkStopsOnServer(price) {
            if (!isValidMarketPrice(price)) return;

            try {
                var res = await fetch('/trading/api/session/check-stops', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ price: price })
                });
                
                if (res.ok) {
                    var data = await res.json();
                    if (data.triggered && data.triggered.length > 0) {
                        data.triggered.forEach(function(order) {
                            activePositions = activePositions.filter(function(p) { return p.id !== order.position_id; });
                            var label = "Stopped Out";
                            if (order.reason === 'liquidated') label = "🚨 LIKUIDASI!";
                            else if (order.reason === 'take_profit') label = "🎯 TAKE PROFIT!";
                            else if (order.reason === 'trailing_stopped') label = "📈 TRAILING STOPPED";
                            
                            showToast(label + ' Posisi #' + order.position_id + ' ditutup pada ' + order.exit_price.toFixed(2) + ' (PnL: ' + (order.pnl > 0 ? '+' : '') + order.pnl + ' Chips)', order.pnl > 0 ? 'success' : 'error');
                        });
                        startTradeCooldown();
                        renderActivePositions();
                        // Reload status to update balance
                        fetchStatus();
                    }
                }
            } catch (err) {
                console.error("Gagal memeriksa stop orders:", err);
            }
        }

        function updateLocalPositionsPnL() {
            activePositions.forEach(function(pos, idx) {
                if (pos.status === 'open') {
                    if (!isValidMarketPrice(currentLivePrice) || !isValidMarketPrice(pos.entry_price)) {
                        activePositions[idx].pnl = 0;
                        return;
                    }

                    // Hitung PnL
                    var priceChangePct = 0;
                    if (pos.direction === 'long') {
                        priceChangePct = (currentLivePrice - pos.entry_price) / pos.entry_price;
                    } else {
                        priceChangePct = (pos.entry_price - currentLivePrice) / pos.entry_price;
                    }
                    
                    var leverage = pos.leverage;
                    var size = pos.size;
                    var pnl = Math.round(priceChangePct * leverage * size) - estimateTradingFee(pos);
                    
                    activePositions[idx].pnl = pnl;

                    // Update trailing peak locally
                    var pnlPct = priceChangePct * leverage * 100;
                    if (pos.trailing_stop) {
                        if (pnlPct > (pos.trailing_peak || 0)) {
                            activePositions[idx].trailing_peak = pnlPct;
                        }
                    }
                }
            });
            renderActivePositions();
        }

        async function endSession() {
            if (playInterval) clearInterval(playInterval);
            playInterval = null;

            showToast("🏁 Sesi trading berakhir! Menghitung keuntungan...", "info");
            
            // Notify server to gracefully end session and auto-close positions
            try {
                await fetch('/trading/api/session/end', { method: 'POST' });
            } catch (e) {
                console.error("Gagal menutup sesi di server:", e);
            }
            
            // Reload status & history to see final values
            await fetchStatus();
            await fetchHistory();
            if (document.getElementById('tabLeaderboard').classList.contains('active')) {
                fetchLeaderboard();
            }

            // Fetch session summary from history to show pattern name and exact PnL
            try {
                var res = await fetch('/trading/api/history/sessions?limit=1');
                if (res.ok) {
                    var data = await res.json();
                    if (data && data.length > 0) {
                        var lastSession = data[0];
                        document.getElementById('recapPatternName').innerText = lastSession.pattern_name + ' (' + lastSession.pattern_type.toUpperCase() + ')';
                        
                        var pnlVal = lastSession.total_pnl;
                        var pnlText = pnlVal >= 0 ? '+' + pnlVal + ' Chips 🟢' : pnlVal + ' Chips 🔴';
                        document.getElementById('recapTotalPnL').innerText = pnlText;
                        document.getElementById('recapTotalPnL').className = "live-stat-val " + (pnlVal >= 0 ? "bullish" : "bearish");
                        
                        document.getElementById('recapNewBalance').innerText = userData.account.balance + " Chips";
                        
                        var emoji = pnlVal >= 0 ? "🏆" : "📉";
                        document.getElementById('recapIcon').innerText = emoji;
                        document.getElementById('recapResultTitle').innerText = pnlVal >= 0 ? "Profit Sesi Luar Biasa!" : "Kerugian Sesi.";
                        
                        openModal('recapModal');
                    }
                }
            } catch (e) {
                console.error("Gagal memuat recap:", e);
            }

            activeSession = null;
            activePositions = [];
            renderActivePositions();
            document.getElementById('chartOverlay').style.display = 'flex';
        }

        // =====================================================================
        // POSITION OPERATIONS
        // =====================================================================

        async function openPosition(direction) {
            if (!activeSession) return;
            if (isOpeningPosition) return;
            if (hasOpenPosition()) {
                showToast("Tutup posisi aktif dulu sebelum buka posisi baru.", "error");
                updateTradeButtons();
                return;
            }
            if (Date.now() < tradeCooldownUntil) {
                var remaining = Math.ceil((tradeCooldownUntil - Date.now()) / 1000);
                showToast("Tunggu cooldown " + remaining + " detik sebelum buka posisi lagi.", "error");
                updateTradeButtons();
                return;
            }

            currentLivePrice = isValidMarketPrice(currentLivePrice) ? currentLivePrice : getLastVisiblePrice();
            if (!isValidMarketPrice(currentLivePrice)) {
                showToast("Harga market belum siap. Tunggu candle pertama muncul.", "error");
                return;
            }
            
            var sizeInput = document.getElementById('txtOrderSize');
            var size = parseInt(sizeInput.value);
            var leverage = parseInt(document.getElementById('sliderLeverage').value);

            if (isNaN(size) || size <= 0) {
                showToast("Jumlah modal margin tidak valid.", "error");
                return;
            }

            // Stops
            var sl = null;
            var tp = null;
            var ts = null;

            if (document.getElementById('wrapperSL').style.display !== 'none') {
                var val = parseFloat(document.getElementById('txtOrderSL').value);
                if (!isNaN(val) && val > 0) sl = val;
            }
            if (document.getElementById('wrapperTP').style.display !== 'none') {
                var val = parseFloat(document.getElementById('txtOrderTP').value);
                if (!isNaN(val) && val > 0) tp = val;
            }
            if (document.getElementById('wrapperTS').style.display !== 'none') {
                var val = parseFloat(document.getElementById('txtOrderTS').value);
                if (!isNaN(val) && val > 0) ts = val;
            }

            var req = {
                session_id: activeSession.session_id,
                direction: direction,
                leverage: leverage,
                size: size,
                stop_loss: sl,
                take_profit: tp,
                trailing_stop: ts,
                entry_price: currentLivePrice
            };

            isOpeningPosition = true;
            setTradeButtonsDisabled(true);
            try {
                var res = await fetch('/trading/api/position/open', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(req)
                });

                if (!res.ok) {
                    var errData = await res.json();
                    throw new Error(errData.error || "Gagal membuka posisi.");
                }

                var data = await res.json();
                activePositions.push(data.position);
                startTradeCooldown();
                showToast('✅ BERHASIL BUKA ' + direction.toUpperCase() + ' #' + data.position.id + ' @ ' + data.position.entry_price.toFixed(2), 'success');
                
                // Clear input stops
                document.getElementById('txtOrderSL').value = '';
                document.getElementById('txtOrderTP').value = '';
                document.getElementById('txtOrderTS').value = '';
                
                fetchStatus(); // update locked balance info
                renderActivePositions();
            } catch (err) {
                showToast(err.message, 'error');
            } finally {
                isOpeningPosition = false;
                updateTradeButtons();
            }
        }

        async function closePosition(positionId) {
            if (!isValidMarketPrice(currentLivePrice)) {
                showToast("Harga market belum siap. Posisi belum bisa ditutup.", "error");
                return;
            }

            try {
                var res = await fetch('/trading/api/position/close', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ position_id: positionId, exit_price: currentLivePrice })
                });

                if (!res.ok) {
                    var errData = await res.json();
                    throw new Error(errData.error || "Gagal menutup posisi.");
                }

                var data = await res.json();
                
                // Mark locally closed
                activePositions = activePositions.filter(function(p) { return p.id !== positionId; });
                startTradeCooldown();
                showToast('🔒 POSISI #' + positionId + ' DITUTUP @ ' + currentLivePrice.toFixed(2) + ' (PnL: ' + (data.position.pnl > 0 ? '+' : '') + data.position.pnl + ' Chips)', data.position.pnl >= 0 ? 'success' : 'error');
                
                fetchStatus();
                renderActivePositions();
                fetchHistory();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        // =====================================================================
        // UI UPDATER FUNCTIONS
        // =====================================================================

        function updateHeaderUI() {
            if (!userData) return;

            document.getElementById('headerChipBalance').innerText = userData.chip_balance.toLocaleString() + " 🪙";
            document.getElementById('headerTradingBalance').innerText = userData.account.balance.toLocaleString() + " Chips";
            document.getElementById('headerName').innerText = userData.account.jid.split('@')[0]; // clean JID name
            document.getElementById('headerRankName').innerText = userData.rank_styled;
            document.getElementById('headerRankEmoji').innerText = userData.rank_emoji;
            
            document.getElementById('lblAvailableTradingMargin').innerText = 'Tersedia: ' + userData.account.balance.toLocaleString() + ' Chips';
            
            // Set max leverage label & input ranges
            var maxLev = userData.max_leverage;
            document.getElementById('sliderLeverage').max = maxLev;
            document.getElementById('lblMaxLeverageLimit').innerText = maxLev + "x";
            
            var sliderVal = parseInt(document.getElementById('sliderLeverage').value);
            if (sliderVal > maxLev) {
                document.getElementById('sliderLeverage').value = maxLev;
            }
            updateLeverageLabel();

            // Limits for modal deposits
            document.getElementById('lblDepositMaxLimit').innerText = 'Maksimal: ' + userData.chip_balance.toLocaleString() + ' Chips';
            document.getElementById('lblWithdrawMaxLimit').innerText = 'Tersedia: ' + userData.account.balance.toLocaleString() + ' Chips';
        }

        function updateLeverageLabel() {
            var val = document.getElementById('sliderLeverage').value;
            document.getElementById('lblLeverageMultiplier').innerText = val + "x";
        }

        function setOrderSizePct(pct) {
            if (!userData) return;
            var size = Math.floor(userData.account.balance * pct);
            document.getElementById('txtOrderSize').value = Math.max(10, size);
        }

        function toggleStopField(type) {
            var btn = document.getElementById('btnToggle' + type);
            var wrap = document.getElementById('wrapper' + type);
            
            if (wrap.style.display === 'none') {
                wrap.style.display = 'flex';
                btn.classList.add('active');
                
                // Pre-fill logical SL / TP price suggestions based on current live price
                if (type === 'SL' && currentLivePrice > 0) {
                    document.getElementById('txtOrderSL').value = (currentLivePrice * 0.98).toFixed(2);
                } else if (type === 'TP' && currentLivePrice > 0) {
                    document.getElementById('txtOrderTP').value = (currentLivePrice * 1.05).toFixed(2);
                } else if (type === 'TS') {
                    document.getElementById('txtOrderTS').value = "2.0";
                }
            } else {
                wrap.style.display = 'none';
                btn.classList.remove('active');
            }
        }

        function addNewsItem(news) {
            var newsFeed = document.getElementById('newsFeed');
            
            var empty = newsFeed.querySelector('.empty-state');
            if (empty) empty.remove();

            var item = document.createElement('div');
            item.className = 'news-item';
            
            var impactClass = 'neutral';
            if (news.sentiment === 'bullish') impactClass = 'bullish';
            else if (news.sentiment === 'bearish') impactClass = 'bearish';

            item.innerHTML = '\n' +
                '                <div class="news-header">\n' +
                '                    <span class="news-time">CANDLE #' + news.time + '</span>\n' +
                '                    <span class="news-badge ' + impactClass + '">' + news.impact.toUpperCase() + ' IMPACT</span>\n' +
                '                </div>\n' +
                '                <div class="news-text">' + news.headline + '</div>\n' +
                '            ';
            
            newsFeed.prepend(item);
        }

        function renderActivePositions() {
            var container = document.getElementById('activePositionsList');
            container.innerHTML = '';

            var openPos = activePositions.filter(function(p) { return p.status === 'open'; });
            if (openPos.length === 0) {
                container.innerHTML = '<div class="empty-state">Tidak ada posisi terbuka. Silakan analisa chart dan buka posisi!</div>';
                return;
            }

            openPos.forEach(function(pos) {
                var card = document.createElement('div');
                card.className = 'position-card ' + pos.direction;
                
                var pnl = Number.isFinite(pos.pnl) ? pos.pnl : 0;
                var pnlClass = pnl >= 0 ? 'bullish' : 'bearish';
                var pnlText = pnl >= 0 ? '+' + pnl + ' Chips' : pnl + ' Chips';
                
                var slLabel = pos.stop_loss ? pos.stop_loss.toFixed(2) : '-';
                var tpLabel = pos.take_profit ? pos.take_profit.toFixed(2) : '-';
                var tsLabel = pos.trailing_stop ? pos.trailing_stop.toFixed(1) + '%' : '-';
                var entryLabel = isValidMarketPrice(pos.entry_price) ? pos.entry_price.toFixed(2) : '-';
                var currentLabel = isValidMarketPrice(currentLivePrice) ? currentLivePrice.toFixed(2) : '-';

                card.innerHTML = '\n' +
                    '                    <div class="pos-header">\n' +
                    '                        <div class="pos-title-group">\n' +
                    '                            <span class="pos-badge ' + pos.direction + '">' + pos.direction.toUpperCase() + '</span>\n' +
                    '                            <span class="pos-size">' + pos.size + ' CHIP × ' + pos.leverage + 'x</span>\n' +
                    '                        </div>\n' +
                    '                        <button class="btn-close-pos" onclick="closePosition(' + pos.id + ')">SELL</button>\n' +
                    '                    </div>\n' +
                    '                    <div class="pos-details">\n' +
                    '                        <div class="pos-detail-item">\n' +
                    '                            <span class="pos-detail-label">Entry</span>\n' +
                    '                            <span class="pos-detail-val">' + entryLabel + '</span>\n' +
                    '                        </div>\n' +
                    '                        <div class="pos-detail-item">\n' +
                    '                            <span class="pos-detail-label">Stop Loss</span>\n' +
                    '                            <span class="pos-detail-val" style="color: var(--color-bearish);">' + slLabel + '</span>\n' +
                    '                        </div>\n' +
                    '                        <div class="pos-detail-item">\n' +
                    '                            <span class="pos-detail-label">Current</span>\n' +
                    '                            <span class="pos-detail-val">' + currentLabel + '</span>\n' +
                    '                        </div>\n' +
                    '                        <div class="pos-detail-item">\n' +
                    '                            <span class="pos-detail-label">Take Profit</span>\n' +
                    '                            <span class="pos-detail-val" style="color: var(--color-bullish);">' + tpLabel + '</span>\n' +
                    '                        </div>\n' +
                    '                        <div class="pos-detail-item" style="grid-column: span 2;">\n' +
                    '                            <span class="pos-detail-label">Trailing Stop</span>\n' +
                    '                            <span class="pos-detail-val" style="color: var(--color-primary-hover);">' + tsLabel + '</span>\n' +
                    '                        </div>\n' +
                    '                    </div>\n' +
                    '                    <div class="pos-pnl-row">\n' +
                    '                        <span class="pos-pnl-label">PnL Bersih</span>\n' +
                    '                        <span class="pos-pnl-val ' + pnlClass + '">' + pnlText + '</span>\n' +
                    '                    </div>\n' +
                    '                ';
                container.appendChild(card);
            });
        }

        function renderTradeHistory(trades) {
            var body = document.getElementById('tblTradeHistoryBody');
            body.innerHTML = '';

            if (!trades || trades.length === 0) {
                body.innerHTML = '<tr><td colspan="9" style="text-align: center; color: var(--text-muted);">Belum ada riwayat transaksi.</td></tr>';
                return;
            }

            trades.forEach(function(t) {
                var tr = document.createElement('tr');
                
                var time = new Date(t.opened_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
                var dirLabel = t.direction === 'long' ? '📈 LONG' : '📉 SHORT';
                var dirColor = t.direction === 'long' ? 'color: var(--color-bullish);' : 'color: var(--color-bearish);';
                
                var pnlClass = t.pnl >= 0 ? 'bullish' : 'bearish';
                var pnlText = t.pnl >= 0 ? '+' + t.pnl : t.pnl;

                tr.innerHTML = '\n' +
                    '                    <td class="mono">' + time + '</td>\n' +
                    '                    <td class="mono">#' + t.id + '</td>\n' +
                    '                    <td style="font-weight: 700; ' + dirColor + '">' + dirLabel + '</td>\n' +
                    '                    <td class="mono">' + t.leverage + 'x</td>\n' +
                    '                    <td class="mono">' + t.size + '</td>\n' +
                    '                    <td class="mono">' + t.entry_price.toFixed(2) + '</td>\n' +
                    '                    <td class="mono">' + t.exit_price.toFixed(2) + '</td>\n' +
                    '                    <td style="text-transform: uppercase; font-size: 0.7rem; font-weight:700;">' + t.status.replace('_', ' ') + '</td>\n' +
                    '                    <td class="pnl-indicator ' + pnlClass + ' mono">' + pnlText + '</td>\n' +
                    '                ';
                body.appendChild(tr);
            });
        }

        function renderSessionHistory(sessions) {
            var body = document.getElementById('tblSessionHistoryBody');
            body.innerHTML = '';

            if (!sessions || sessions.length === 0) {
                body.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--text-muted);">Belum ada riwayat sesi.</td></tr>';
                return;
            }

            sessions.forEach(function(s) {
                var tr = document.createElement('tr');
                
                var time = new Date(s.start_time).toLocaleDateString([], { month: 'short', day: 'numeric' }) + " " + new Date(s.start_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
                
                var pnlClass = s.total_pnl >= 0 ? 'bullish' : 'bearish';
                var pnlText = s.total_pnl >= 0 ? '+' + s.total_pnl : s.total_pnl;

                tr.innerHTML = '\n' +
                    '                    <td class="mono">' + time + '</td>\n' +
                    '                    <td class="mono">#' + s.id + '</td>\n' +
                    '                    <td style="text-transform: uppercase; font-size: 0.75rem;">' + s.pattern_type + '</td>\n' +
                    '                    <td style="font-weight: 700; color: var(--color-primary-hover);">' + s.pattern_name + '</td>\n' +
                    '                    <td class="mono">' + s.trade_count + ' trades</td>\n' +
                    '                    <td style="text-transform: uppercase; font-size: 0.7rem; font-weight:700;">' + s.status + '</td>\n' +
                    '                    <td class="pnl-indicator ' + pnlClass + ' mono">' + pnlText + '</td>\n' +
                    '                ';
                body.appendChild(tr);
            });

            // Update stats tab also
            if (userData && userData.stats) {
                document.getElementById('statWinRate').innerText = userData.stats.win_rate.toFixed(1) + "%";
                
                var pnl = userData.stats.total_pnl;
                document.getElementById('statTotalPnL').innerText = (pnl >= 0 ? '+' : '') + pnl.toLocaleString() + " Chips";
                document.getElementById('statTotalPnL').style.color = pnl >= 0 ? CSS_COLORS.bullish : CSS_COLORS.bearish;
                
                document.getElementById('statTotalTrades').innerText = userData.stats.total_trades;
                
                var avg = userData.stats.avg_pnl;
                document.getElementById('statAvgPnL').innerText = (avg >= 0 ? '+' : '') + avg.toFixed(1) + " Chips";
                document.getElementById('statAvgPnL').style.color = avg >= 0 ? CSS_COLORS.bullish : CSS_COLORS.bearish;
            }
        }

        // =====================================================================
        // CHART RENDERING ( Bespoke Canvas Draw )
        // =====================================================================

        function drawChart() {
            if (!activeSession || activeSession.observation.length === 0) return;

            var data = activeSession.observation;
            var indicators = activeSession.indicators;

            // Clear canvases
            mainCtx.fillStyle = CSS_COLORS.bg;
            mainCtx.fillRect(0, 0, mainCanvas.width, mainCanvas.height);
            indCtx.fillStyle = CSS_COLORS.bg;
            indCtx.fillRect(0, 0, indCanvas.width, indCanvas.height);

            // Min / Max prices
            var minPrice = Infinity;
            var maxPrice = -Infinity;
            
            data.forEach(function(c) {
                if (c.h > maxPrice) maxPrice = c.h;
                if (c.l < minPrice) minPrice = c.l;
            });

            // Spare space top and bottom
            var priceRange = maxPrice - minPrice;
            maxPrice += priceRange * 0.1;
            minPrice -= priceRange * 0.1;

            // X scale (candle width)
            var paddingRight = 60; // right padding for price scale
            var chartWidth = mainCanvas.width - paddingRight;
            var candleWidth = chartWidth / data.length;

            // Draw horizontal Grid lines
            mainCtx.strokeStyle = 'rgba(30, 41, 93, 0.15)';
            mainCtx.lineWidth = 1;
            for (var i = 0; i < 5; i++) {
                var gridPrice = minPrice + (maxPrice - minPrice) * (i / 4);
                var y = mainCanvas.height - ((gridPrice - minPrice) / (maxPrice - minPrice)) * mainCanvas.height;
                
                mainCtx.beginPath();
                mainCtx.moveTo(0, y);
                mainCtx.lineTo(chartWidth, y);
                mainCtx.stroke();

                // Draw price text label
                mainCtx.fillStyle = CSS_COLORS.textMuted;
                mainCtx.font = '9px var(--font-mono)';
                mainCtx.fillText(gridPrice.toFixed(2), chartWidth + 5, y + 3);
            }

            // Draw Observation/Resolution boundary line
            var boundaryX = activeSession.obs_candles * candleWidth;
            mainCtx.strokeStyle = 'rgba(139, 92, 246, 0.4)';
            mainCtx.setLineDash([5, 5]);
            mainCtx.beginPath();
            mainCtx.moveTo(boundaryX, 0);
            mainCtx.lineTo(boundaryX, mainCanvas.height);
            mainCtx.stroke();
            mainCtx.setLineDash([]); // Reset line dash

            // Draw labels for phases
            mainCtx.fillStyle = 'rgba(139, 92, 246, 0.3)';
            mainCtx.font = '10px var(--font-main)';
            mainCtx.fillText("ANALISIS HISTORIS", 15, 20);
            if (data.length > activeSession.obs_candles) {
                mainCtx.fillStyle = 'rgba(16, 185, 129, 0.3)';
                mainCtx.fillText("LIVE RESOLUTION TRADING", boundaryX + 15, 20);
            }

            // Draw Candlesticks & Indicators (MA20)
            data.forEach(function(c, idx) {
                var x = idx * candleWidth + (candleWidth / 2);
                
                // Price coordinates
                var yOpen = mainCanvas.height - ((c.o - minPrice) / (maxPrice - minPrice)) * mainCanvas.height;
                var yClose = mainCanvas.height - ((c.c - minPrice) / (maxPrice - minPrice)) * mainCanvas.height;
                var yHigh = mainCanvas.height - ((c.h - minPrice) / (maxPrice - minPrice)) * mainCanvas.height;
                var yLow = mainCanvas.height - ((c.l - minPrice) / (maxPrice - minPrice)) * mainCanvas.height;
                
                var isBullish = c.c >= c.o;
                var candleColor = isBullish ? CSS_COLORS.bullish : CSS_COLORS.bearish;
                
                // Draw Wick (shadow line)
                mainCtx.strokeStyle = candleColor;
                mainCtx.lineWidth = 1.5;
                mainCtx.beginPath();
                mainCtx.moveTo(x, yHigh);
                mainCtx.lineTo(x, yLow);
                mainCtx.stroke();

                // Draw Body rectangle
                mainCtx.fillStyle = candleColor;
                var bodyWidth = Math.max(2, candleWidth * 0.7);
                var rectHeight = Math.max(1.5, Math.abs(yClose - yOpen));
                var rectY = Math.min(yOpen, yClose);
                mainCtx.fillRect(x - (bodyWidth / 2), rectY, bodyWidth, rectHeight);
            });

            // Draw MA 20 Indicator Line
            if (indicators && indicators.MA20 && indicators.MA20.length > 0) {
                mainCtx.strokeStyle = '#a78bfa';
                mainCtx.lineWidth = 2;
                mainCtx.beginPath();

                indicators.MA20.forEach(function(ma, idx) {
                    if (ma === 0 || idx >= data.length) return; // guard
                    var x = idx * candleWidth + (candleWidth / 2);
                    var y = mainCanvas.height - ((ma - minPrice) / (maxPrice - minPrice)) * mainCanvas.height;
                    
                    if (idx === 0) mainCtx.moveTo(x, y);
                    else mainCtx.lineTo(x, y);
                });
                mainCtx.stroke();
            }

            // Draw Active position lines on chart
            activePositions.forEach(function(pos) {
                if (pos.status === 'open') {
                    var yEntry = mainCanvas.height - ((pos.entry_price - minPrice) / (maxPrice - minPrice)) * mainCanvas.height;
                    
                    // Entry Line (dotted purple)
                    mainCtx.strokeStyle = 'rgba(139, 92, 246, 0.6)';
                    mainCtx.setLineDash([4, 4]);
                    mainCtx.lineWidth = 1;
                    mainCtx.beginPath();
                    mainCtx.moveTo(0, yEntry);
                    mainCtx.lineTo(chartWidth, yEntry);
                    mainCtx.stroke();

                    // Text tag
                    mainCtx.fillStyle = 'rgba(139, 92, 246, 0.8)';
                    mainCtx.font = '8px var(--font-mono)';
                    mainCtx.fillText("POS #" + pos.id + " " + pos.direction.toUpperCase() + " @ " + pos.entry_price.toFixed(1), 5, yEntry - 4);
                    
                    // Stop Loss Line (dotted red)
                    if (pos.stop_loss) {
                        var ySL = mainCanvas.height - (pos.stop_loss - minPrice) / (maxPrice - minPrice) * mainCanvas.height;
                        mainCtx.strokeStyle = 'rgba(239, 68, 68, 0.5)';
                        mainCtx.beginPath();
                        mainCtx.moveTo(0, ySL);
                        mainCtx.lineTo(chartWidth, ySL);
                        mainCtx.stroke();
                        
                        mainCtx.fillStyle = 'rgba(239, 68, 68, 0.7)';
                        mainCtx.fillText("SL #" + pos.id + " @ " + pos.stop_loss.toFixed(1), chartWidth - 120, ySL - 4);
                    }
                    
                    // Take Profit Line (dotted green)
                    if (pos.take_profit) {
                        var yTP = mainCanvas.height - (pos.take_profit - minPrice) / (maxPrice - minPrice) * mainCanvas.height;
                        mainCtx.strokeStyle = 'rgba(16, 185, 129, 0.5)';
                        mainCtx.beginPath();
                        mainCtx.moveTo(0, yTP);
                        mainCtx.lineTo(chartWidth, yTP);
                        mainCtx.stroke();
                        
                        mainCtx.fillStyle = 'rgba(16, 185, 129, 0.7)';
                        mainCtx.fillText("TP #" + pos.id + " @ " + pos.take_profit.toFixed(1), chartWidth - 120, yTP - 4);
                    }

                    mainCtx.setLineDash([]); // Reset
                }
            });

            // ─────────────────────────────────────────────────────────────
            // INDICATOR CHART RENDER (RSI / MACD)
            // ─────────────────────────────────────────────────────────────
            
            // Draw RSI Chart if available
            if (indicators && indicators.RSI && indicators.RSI.length > 0) {
                // Clear and outline
                indCtx.strokeStyle = 'rgba(30, 41, 93, 0.2)';
                indCtx.lineWidth = 1;
                
                // Draw Overbought (70) / Oversold (30) boundary areas
                var y70 = indCanvas.height - (70 / 100) * indCanvas.height;
                var y30 = indCanvas.height - (30 / 100) * indCanvas.height;
                
                indCtx.fillStyle = 'rgba(139, 92, 246, 0.03)';
                indCtx.fillRect(0, y70, chartWidth, y30 - y70);
                
                indCtx.beginPath();
                indCtx.moveTo(0, y70); indCtx.lineTo(chartWidth, y70);
                indCtx.moveTo(0, y30); indCtx.lineTo(chartWidth, y30);
                indCtx.stroke();

                indCtx.fillStyle = 'rgba(255,255,255,0.15)';
                indCtx.font = '8px var(--font-mono)';
                indCtx.fillText("70 RSI (OB)", chartWidth + 5, y70 + 3);
                indCtx.fillText("30 RSI (OS)", chartWidth + 5, y30 + 3);

                // Draw RSI Line
                indCtx.strokeStyle = '#f59e0b'; // Amber line
                indCtx.lineWidth = 1.5;
                indCtx.beginPath();

                indicators.RSI.forEach(function(rsi, idx) {
                    if (rsi === 0 || idx >= data.length) return;
                    var x = idx * candleWidth + (candleWidth / 2);
                    var y = indCanvas.height - (rsi / 100) * indCanvas.height;
                    
                    if (idx === 0) indCtx.moveTo(x, y);
                    else indCtx.lineTo(x, y);
                });
                indCtx.stroke();
                
                indCtx.fillStyle = '#f59e0b';
                indCtx.font = '9px var(--font-main)';
                indCtx.fillText("Relative Strength Index (RSI): " + indicators.RSI[indicators.RSI.length-1].toFixed(1), 10, 15);
            }
        }

        // =====================================================================
        // AUXILIARY ACTIONS (Deposits, Withdrawals, Tabs, Modals)
        // =====================================================================

        function openModal(id) {
            document.getElementById(id).classList.add('active');
            
            // If debug input open, focus
            if (id === 'debugModal') {
                document.getElementById('txtDebugInput').focus();
            }
        }

        function closeModal(id) {
            document.getElementById(id).classList.remove('active');
        }

        function switchTab(tabId, btn) {
            // Hide all
            document.querySelectorAll('.tab-content').forEach(function(c) { c.classList.remove('active'); });
            document.querySelectorAll('.tab-btn').forEach(function(b) { b.classList.remove('active'); });
            
            // Show selected
            document.getElementById(tabId).classList.add('active');
            btn.classList.add('active');

            if (tabId === 'tabLeaderboard') {
                fetchLeaderboard();
            }
        }

        async function executeDeposit() {
            var amountInput = document.getElementById('txtDepositAmount');
            var amount = parseInt(amountInput.value);
            
            if (isNaN(amount) || amount <= 0) {
                showToast("Jumlah deposit tidak valid.", "error");
                return;
            }

            try {
                var res = await fetch('/trading/api/deposit', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ amount: amount })
                });

                if (!res.ok) {
                    var data = await res.json();
                    throw new Error(data.error || "Gagal melakukan deposit.");
                }

                showToast("💰 Sukses Deposit +" + amount + " Chips ke Akun Trading!", "success");
                closeModal('depositModal');
                amountInput.value = '';
                fetchStatus();
            } catch (err) {
                showToast(err.message, "error");
            }
        }

        async function executeWithdraw() {
            var amountInput = document.getElementById('txtWithdrawAmount');
            var amount = parseInt(amountInput.value);
            
            if (isNaN(amount) || amount <= 0) {
                showToast("Jumlah withdraw tidak valid.", "error");
                return;
            }

            try {
                var res = await fetch('/trading/api/withdraw', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ amount: amount })
                });

                if (!res.ok) {
                    var data = await res.json();
                    throw new Error(data.error || "Gagal melakukan withdraw.");
                }

                showToast("💸 Sukses Withdraw +" + amount + " Chips ke Dompet Utama!", "success");
                closeModal('withdrawModal');
                amountInput.value = '';
                fetchStatus();
            } catch (err) {
                showToast(err.message, "error");
            }
        }

        async function logout() {
            try {
                var res = await fetch('/trading/api/logout', { method: 'POST' });
                if (res.ok) {
                    showToast("Keluar sesi berhasil. Mengalihkan...", "info");
                    setTimeout(function() { window.location.reload(); }, 1500);
                }
            } catch (e) {
                window.location.reload();
            }
        }

        // Multi-step tutorial navigation
        function navTutorial(dir) {
            var nextStep = tutorialStep + dir;
            if (nextStep < 0 || nextStep > 6) return;
            
            // Mark step on server
            if (dir > 0) {
                fetch('/trading/api/tutorial/complete-step', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ step: nextStep })
                });
            }

            tutorialStep = nextStep;
            updateTutorialUI();
        }

        function handleNextClick() {
            if (tutorialStep === 6) {
                startTutorialPractice();
            } else {
                navTutorial(1);
            }
        }

        function updateTutorialUI() {
            if (tutorialStep < 0) tutorialStep = 0;
            if (tutorialStep > 6) tutorialStep = 6;

            // Hide all panes & deactivate dots
            document.querySelectorAll('.tutorial-content-pane').forEach(function(p) { p.classList.remove('active'); });
            document.querySelectorAll('.tutorial-step-dot').forEach(function(d, idx) {
                d.className = 'tutorial-step-dot';
                if (idx < tutorialStep) d.classList.add('completed');
                else if (idx === tutorialStep) d.classList.add('active');
            });

            // Show selected pane
            var pane = document.getElementById('tutPane' + tutorialStep);
            if (pane) {
                pane.classList.add('active');
            }

            // Button configurations
            var prev = document.getElementById('btnTutorialPrev');
            var next = document.getElementById('btnTutorialNext');

            prev.style.visibility = tutorialStep === 0 ? 'hidden' : 'visible';
            
            if (tutorialStep === 6) {
                next.innerText = "Mulai Latihan 🎮";
            } else {
                next.innerText = "Selanjutnya";
            }
        }

        async function startTutorialPractice() {
            showToast("Memulai sesi latihan...", "info");
            closeModal('tutorialModal');
            document.getElementById('chartOverlay').style.display = 'none';

            try {
                var res = await fetch('/trading/api/tutorial/practice', { method: 'POST' });
                if (!res.ok) throw new Error("Gagal memulai sesi latihan.");
                
                var data = await res.json();
                
                // Sesi Latihan: load exact resolved observation & resolution candles
                activeSession = {
                    session_id: 0,
                    observation: data.observation,
                    indicators: data.indicators,
                    obs_candles: data.observation.length,
                    news: data.news,
                    difficulty: "practice"
                };
                
                hiddenResolutionData = data.resolution;
                hiddenIndicators = null;
                hiddenPatternName = data.pattern;
                activePositions = [];
                currentTickIndex = 0;
                currentLivePrice = getLastVisiblePrice();
                
                // Clear news feed and load observation news
                var newsFeed = document.getElementById('newsFeed');
                newsFeed.innerHTML = '';
                if (data.news) data.news.forEach(addNewsItem);

                document.getElementById('sessionStatusBadge').innerText = "Akademik (Practice)";
                document.getElementById('sessionStatusBadge').className = "session-badge active";
                document.getElementById('sessionDetailID').innerText = "PRACTICE #0";
                document.getElementById('sessionCountdown').innerText = "Latihan Bebas Resiko";
                document.getElementById('sessionCandleProgress').innerText = "40 / 60";
                document.getElementById('sessionProgressFill').style.width = "66.6%";
                
                updateTradeButtons();

                resizeCanvases();
                
                // Tick 
                setTimeout(startResolutionTicks, 2000);

            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        // =====================================================================
        // DEBUG MODE TERMINAL ENGINE
        // =====================================================================

        function printToTerminal(text, isInput) {
            var screen = document.getElementById('debugTerminalScreen');
            var line = document.createElement('div');
            line.className = 'terminal-line';
            
            if (isInput) {
                line.innerHTML = '<span class="terminal-prompt">$</span><span>' + text + '</span>';
            } else {
                line.innerHTML = '<span>' + text + '</span>';
            }
            
            screen.appendChild(line);
            screen.scrollTop = screen.scrollHeight;
        }

        async function handleDebugCommand(event) {
            if (event.key !== 'Enter') return;
            
            var input = document.getElementById('txtDebugInput');
            var cmd = input.value.trim();
            input.value = '';

            if (cmd === '') return;

            printToTerminal(cmd, true);

            var args = cmd.split(' ');
            var primary = args[0].toLowerCase();

            // Check if authenticated
            if (!debugMode && primary !== 'auth') {
                printToTerminal("❌ Error: Developer authentication required. Type 'auth [password]' to login.");
                return;
            }

            switch (primary) {
                case 'help':
                    printToTerminal("🖥️ Available Commands:");
                    printToTerminal("  - auth [password] : Login to dev debug control panel.");
                    printToTerminal("  - reveal          : Reveal pattern code, name, noise level and resolve outcome.");
                    printToTerminal("  - setbalance [n]  : Instantly set trading balance.");
                    printToTerminal("  - patterns        : Lists all 23 available mathematical model patterns.");
                    printToTerminal("  - clear           : Clear dev terminal outputs.");
                    break;

                case 'auth':
                    if (args.length < 2) {
                        printToTerminal("❌ Error: Password required. Format: auth [password]");
                        break;
                    }
                    var pwd = args[1];
                    try {
                        var res = await fetch('/trading/api/debug/auth', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ password: pwd })
                        });
                        var data = await res.json();
                        if (data.success) {
                            debugMode = true;
                            debugPassword = pwd;
                            printToTerminal("🔓 Developer authorization successful! Full sandbox control active.");
                            showToast("Dev sandbox unlocked!", "success");
                        } else {
                            printToTerminal("❌ Error: Authentication failed. Invalid password.");
                        }
                    } catch (e) {
                        printToTerminal("❌ Error connecting to auth server: " + e.message);
                    }
                    break;

                case 'reveal':
                    if (!activeSession) {
                        printToTerminal("❌ Error: No active session is running to reveal.");
                        break;
                    }
                    try {
                        var res = await fetch('/trading/api/debug/reveal', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ password: debugPassword, session_id: activeSession.session_id })
                        });
                        var data = await res.json();
                        printToTerminal("🔍 REVEAL SESSION #" + activeSession.session_id + ":");
                        printToTerminal("  - Pattern: " + data.pattern_name);
                        printToTerminal("  - Seed: " + (activeSession.observation[0].Seed || '-'));
                        printToTerminal("  - Expected Direction: " + (data.resolution[data.resolution.length-1].c > data.resolution[0].o ? 'BULLISH (UP) 📈' : 'BEARISH (DOWN) 📉'));
                        printToTerminal("  - Future resolution data points loaded: " + data.resolution.length + " points.");
                        showToast("Secret Revealed: " + data.pattern_name, "warning");
                    } catch (e) {
                        printToTerminal("❌ Error: " + e.message);
                    }
                    break;

                case 'setbalance':
                    if (args.length < 2) {
                        printToTerminal("❌ Error: Amount value required. Format: setbalance [amount]");
                        break;
                    }
                    var amt = parseInt(args[1]);
                    try {
                        var res = await fetch('/trading/api/debug/set-balance', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ password: debugPassword, amount: amt })
                        });
                        var data = await res.json();
                        if (data.success) {
                            printToTerminal("💸 Saldo trading berhasil diubah menjadi: " + data.new_balance + " Chips!");
                            fetchStatus();
                        } else {
                            printToTerminal("❌ Error: Gagal mengubah saldo.");
                        }
                    } catch (e) {
                        printToTerminal("❌ Error: " + e.message);
                    }
                    break;

                case 'patterns':
                    try {
                        var res = await fetch('/trading/api/debug/patterns?password=' + debugPassword);
                        var data = await res.json();
                        printToTerminal("📈 LIST OF 23 MATHEMATICAL PATTERNS:");
                        data.forEach(function(p) {
                            printToTerminal("  - " + p.name + " [Type: " + p.type + "] -> Direction: " + p.direction + " (Diff: " + p.difficulty + ")");
                        });
                    } catch (e) {
                        printToTerminal("❌ Error: " + e.message);
                    }
                    break;

                case 'clear':
                    document.getElementById('debugTerminalScreen').innerHTML = '';
                    break;

                default:
                    printToTerminal("❌ Command not found: '" + primary + "'. Type 'help' for developer assistance.");
            }
        }

        // =====================================================================
        // TOAST ENGINE
        // =====================================================================

        function showToast(text, type) {
            var container = document.getElementById('toastContainer');
            
            var toast = document.createElement('div');
            toast.className = 'toast ' + (type || 'info');
            
            var emoji = '💡';
            if (type === 'success') emoji = '✅';
            else if (type === 'error') emoji = '❌';
            else if (type === 'warning') emoji = '⚠️';

            toast.innerHTML = '<span>' + emoji + '</span><span>' + text + '</span>';
            container.appendChild(toast);

            // Reflow to animate
            setTimeout(function() { toast.classList.add('show'); }, 50);

            // Auto dismiss after 4 seconds
            setTimeout(function() {
                toast.classList.remove('show');
                setTimeout(function() { toast.remove(); }, 300);
            }, 4000);
        }
    </script>
</body>
</html>`
