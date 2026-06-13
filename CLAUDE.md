# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o cakrawala ./cmd/...   # build
go run ./cmd/...                   # run directly
go test ./...                      # all tests
go test ./internal/fetcher/...     # single package
go mod tidy                        # sync dependencies
```

Binary subcommands (once built):
```bash
./cakrawala run           # one-shot run
./cakrawala run --dry-run # no vault write
./cakrawala start         # daemon with built-in cron (07:00 WIB on weekdays)
```

## Architecture

Pipeline triggered by cron or `run` command:

```
IDX Fetcher (colly)
  └── PDF Extractor (two-layer: text → Claude vision fallback)
News Fetcher (RSS, concurrent)
  └── Deduplicator (hash-based, cross-source)
  └── Ticker Filter (relevance + ticker extraction)
AI Classifier (Claude API, batched 20–30 items/call)
Merger (group Disclosure + Article by ticker)
Obsidian Writer (render markdown → vault)
```

All shared types live in `pkg/model`. The pipeline passes `[]Disclosure` and `[]Article` through to produce `DailyBrief` (bucketed into Positive/Negative/Neutral `ClassifiedItem` slices).

## Key implementation details

**PDF extraction** (`internal/extractor/pdf.go`): extract text from first 2 pages, hard-cap 3000 chars. If result < 150 chars (likely scanned), fall back to Claude vision. Skip `Laporan Keuangan Tahunan` and `Laporan Tahunan` entirely — too large, not useful for daily brief.

**Classifier** (`internal/classifier/claude.go`): use `claude-haiku-4-5`. Batch 20–30 items per API call. Each item gets `Sentiment`, `Confidence`, and `Reason`. Prompt must return structured JSON so it can be deserialized into `ClassifiedItem`.

**Dedup** (`internal/dedup/dedup.go`): hash-based on `Disclosure.Hash` / `Article.Hash`. Cross-source (same story from Bisnis + Kontan = one item).

**Config** (`config/config.go`): loads `config.yaml` from repo root. Fields: `anthropic_api_key`, `vault_path`, `vault_folder`, `cron_schedule`, `batch_size`, `max_pdf_chars`, `watchlist[]`, `news_sources{}`.

## Known issues / things to fix

- `pkg/model/model.go:18–19`: `ConfidenceMedium` and `ConfidenceLow` string values are **swapped** (`"low"` / `"medium"` assigned backwards).
- `pkg/model/model.go:49`: `ClassifiedItem.Reaseon` is a typo — should be `Reason`.
- Most `internal/` packages are stubs (empty or `panic("unimplemented")`). `internal/writer/` (Obsidian writer) doesn't exist yet.
- `config/config.go` is empty — config loader not yet implemented.
- `cmd/main.go` is `package cmd`, not `package main` — entrypoint needs fixing or the package needs renaming.
