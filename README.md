# cakrawala

Automated IDX morning brief — fetches disclosures & news, classifies market sentiment with AI, and writes daily notes to Obsidian.

---

## What it does

`cakrawala` runs every weekday morning at 07:00 WIB and produces a structured daily brief in your Obsidian vault containing:

- **IDX disclosures** — keterbukaan informasi dari idx.co.id, diekstrak dari PDF dan diklasifikasikan otomatis
- **Market news** — berita dari Bisnis.com, Kontan, CNBC Indonesia, Reuters, dan Bloomberg yang relevan dengan emiten IDX
- **Sentiment classification** — tiap item dikategorikan sebagai `PRICE_MOVING_UP`, `PRICE_MOVING_DOWN`, atau `NEUTRAL` oleh Claude AI

Output dikelompokkan per emiten sehingga satu saham kelihatan semua konteksnya — disclosure IDX dan berita — dalam satu tempat.

---

## Output example

```markdown
---
date: 2026-06-10
tags: [pasar-modal, daily-brief]
source: cakrawala
---

## Morning Brief — 10 Juni 2026

### 🟢 Price Moving (3)

**BBRI**
- [IDX] Pengumuman dividen interim Rp 42/saham — *high confidence*
- [Bisnis.com] Analis rekomendasikan buy BBRI jelang ex-date

**AMMN**
- [Reuters] Copper prices hit 3-month high on supply concerns

---

### ⚪ Neutral (8)
...

### 🔴 Negative (1)
...
```

---

## Architecture

```
Cron (07:00 WIB)
    ├── IDX Fetcher        → scrape keterbukaan informasi
    │     └── PDF Extractor → extract teks, fallback ke vision
    ├── News Fetcher        → RSS multi-source (concurrent)
    │     └── Deduplicator  → hash-based, cross-source
    │     └── Ticker Filter → relevance filter + ticker extraction
    │
    ├── AI Classifier       → Claude API, batched (20-30 items/call)
    │
    ├── Merger              → group per ticker, gabung IDX + news
    └── Obsidian Writer     → render markdown + write ke vault
```

---

## Prerequisites

- Go 1.22+
- Anthropic API key
- Obsidian vault (local path)

---

## Installation

```bash
git clone https://github.com/yourusername/cakrawala.git
cd cakrawala
go mod tidy
go build -o cakrawala ./cmd/brief
```

---

## Configuration

Buat file `config.yaml` di root project:

```yaml
anthropic_api_key: "sk-ant-..."
vault_path: "/Users/yourname/Documents/ObsidianVault"
vault_folder: "Pasar Modal/Daily Brief"
cron_schedule: "0 7 * * 1-5"
batch_size: 20
max_pdf_chars: 3000
watchlist:
  - BBRI
  - AMMN
  - BMRI
  - TLKM
news_sources:
  bisnis: "https://www.bisnis.com/rss/market"
  kontan: "https://insight.kontan.co.id/rss"
  cnbc: "https://www.cnbcindonesia.com/rss/market"
  reuters: "https://feeds.reuters.com/reuters/businessNews"
```

---

## Usage

**Run sekali (manual):**
```bash
./cakrawala run
```

**Jalankan sebagai daemon dengan built-in cron:**
```bash
./cakrawala start
```

**Test output tanpa nulis ke vault:**
```bash
./cakrawala run --dry-run
```

---

## Project structure

```
cakrawala/
├── cmd/brief/           # entrypoint
├── internal/
│   ├── fetcher/         # idx.go, news.go
│   ├── extractor/       # pdf.go
│   ├── classifier/      # claude.go
│   ├── dedup/           # dedup.go
│   ├── filter/          # ticker.go
│   ├── merger/          # merger.go
│   └── writer/          # obsidian.go
├── pkg/model/           # shared structs
├── config/              # config loader
└── go.mod
```

---

## PDF handling

Disclosure IDX mayoritas PDF digital (text-based). `cakrawala` pakai strategi dua layer:

1. **Extract teks** via `ledongthuc/pdf` — ambil max 2 halaman pertama, hard cap 3000 karakter
2. **Fallback ke Claude vision** — kalau hasil extract < 150 karakter (kemungkinan scan)

Laporan keuangan full (`Laporan Keuangan Tahunan`, `Laporan Tahunan`) di-skip karena tidak relevan untuk daily brief.

---

## Cost estimate

Dengan asumsi ~50 disclosure + ~30 artikel per hari, batched 20-30 item per API call:

| Model | Est. calls/day | Est. cost/day |
|---|---|---|
| claude-haiku-4-5 | ~5 calls | ~$0.01 |

---

## License

MIT
