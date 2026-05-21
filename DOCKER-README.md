# Docker Quick Reference

Quick reference untuk menjalankan bot dengan Docker.

## 🚀 Quick Start (3 Steps)

```bash
# 1. Setup environment
make setup
nano .env  # Minimal GEMINI_API_KEY dan ALLOWED_GROUP_JID

# 2. Start bot
make start

# 3. View logs (scan QR code)
make logs
```

## 🔧 Commands

Semua command dijalankan dari **root project** menggunakan Makefile:

```bash
make start         # Start bot
make stop          # Stop bot
make restart       # Restart bot
make logs          # View logs
make status        # Check status
make update        # Update bot
make reset-session # Reset WhatsApp session
make backup        # Backup database
```

## 📁 Structure

```
.
├── Makefile                    # Commands (run from here)
├── docker/                     # Docker files
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   └── DOCKER*.md             # Documentation
├── data/                       # Database (persistent)
│   ├── wa-economy.db
│   └── wa-session.db
└── .env                        # Configuration (create from .env.example)
```

## 📊 Database Location

Database files ada di `data/`:

- `data/wa-economy.db` - Economy database
- `data/wa-session.db` - WhatsApp session

Folder ini sudah di `.gitignore`, aman untuk development.

## 🐛 Troubleshooting

### Bot tidak connect?

```bash
make reset-session
```

### Port 8080 sudah dipakai?

Edit `docker/docker-compose.yml`:

```yaml
ports:
  - "9090:8080" # Ganti 9090 dengan port yang available
```

### Update bot?

```bash
make update
```

### Lihat status?

```bash
make status
```

## 🎯 Production

Untuk production, gunakan config production:

```bash
make prod-start    # Start dengan production config
make prod-logs     # View production logs
make prod-stop     # Stop production
```

Production config includes:

- Resource limits (CPU: 1.0, Memory: 512M)
- Health checks
- Better logging
- Auto-restart always

---

**Ready to start?** Run `make start` 🚀
