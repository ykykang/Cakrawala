package fetcher

import (
	"cakrawala/pkg/model"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	idxBaseURL  = "https://www.idx.co.id"
	idxAPIPath  = "/primary/ListedCompany/GetAnnouncement"
	idxPageSize = 100
)

var skipTitlePrefixes = []string{
	"laporan keuangan",
	"laporan tahunan",
	"annual report",
}

type IDXFetcher interface {
	FetchDisclosures(ctx context.Context, from, to time.Time) ([]model.Disclosure, error)
}

type idxFetcher struct{}

func NewIDXFetcher() IDXFetcher {
	return &idxFetcher{}
}

type idxResponse struct {
	ResultCount int        `json:"ResultCount"`
	Replies     []idxReply `json:"Replies"`
}

type idxReply struct {
	Pengumuman  idxPengumuman  `json:"pengumuman"`
	Attachments []idxFile      `json:"attachments"`
}

type idxPengumuman struct {
	KodeEmiten string `json:"Kode_Emiten"`
	Judul      string `json:"JudulPengumuman"`
	Tanggal    string `json:"TglPengumuman"`
	Jenis      string `json:"JenisPengumuman"`
}

type idxFile struct {
	FullSavePath string `json:"FullSavePath"`
	IsAttachment bool   `json:"IsAttachment"`
}

// FetchDisclosures menggunakan headless Chrome (chromedp) bukan net/http biasa.
// IDX dilindungi Cloudflare yang blokir semua raw HTTP request (return 403).
// Cloudflare solve JS challenge hanya kalau request datang dari browser asli.
// Strateginya:
//  1. Launch Chrome headless → navigate ke halaman IDX
//  2. Chrome solve Cloudflare JS challenge secara otomatis (karena dia browser beneran)
//  3. Chrome dapat CF cookies (__cf_bm, cf_clearance)
//  4. Kita jalankan fetch() API lewat JS di dalam Chrome yang sudah punya cookies itu
//  5. Request API lolos karena bawa cookies valid dari browser yang sudah diverifikasi CF
func (f *idxFetcher) FetchDisclosures(ctx context.Context, from, to time.Time) ([]model.Disclosure, error) {
	chromeCtx, cancel, err := newChromeCtx(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	fromStr := from.Format("20060102")
	toStr := to.Format("20060102")
	log.Printf("[idx] date range: %s → %s", fromStr, toStr)

	var all []model.Disclosure
	for indexFrom := 0; ; indexFrom += idxPageSize {
		batch, total, err := f.fetchPage(chromeCtx, fromStr, toStr, indexFrom)
		if err != nil {
			return nil, fmt.Errorf("indexFrom %d: %w", indexFrom, err)
		}
		all = append(all, batch...)

		if len(all) >= total || len(batch) == 0 {
			break
		}
	}

	return all, nil
}

func (f *idxFetcher) fetchPage(ctx context.Context, fromStr, toStr string, indexFrom int) ([]model.Disclosure, int, error) {
	url := fmt.Sprintf("%s%s?kodeEmiten=&emitenType=*&indexFrom=%d&pageSize=%d&dateFrom=%s&dateTo=%s&lang=id&keyword=",
		idxBaseURL, idxAPIPath, indexFrom, idxPageSize, fromStr, toStr)

	log.Printf("[idx] fetching indexFrom=%d", indexFrom)

	body, err := jsFetch(ctx, url)
	if err != nil {
		return nil, 0, err
	}

	log.Printf("[idx] response preview: %.200s", body)

	var result idxResponse
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return nil, 0, fmt.Errorf("decode response (body: %.100s): %w", body, err)
	}
	log.Printf("[idx] indexFrom=%d: total=%d replies=%d", indexFrom, result.ResultCount, len(result.Replies))

	var disclosures []model.Disclosure
	for _, r := range result.Replies {
		d, ok := toDisclosure(r)
		if ok {
			disclosures = append(disclosures, d)
		}
	}

	return disclosures, result.ResultCount, nil
}

// newChromeCtx launches headless Chrome with anti-detection flags and navigates
// to the IDX disclosure page so Cloudflare cookies are set before any API calls.
func newChromeCtx(ctx context.Context) (context.Context, context.CancelFunc, error) {
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", "new"),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
			chromedp.Flag("disable-infobars", true),
			chromedp.Flag("excludeSwitches", "enable-automation"),
			chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
		)...,
	)

	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	cancel := func() {
		cancelChrome()
		cancelAlloc()
	}

	// Mask navigator.webdriver before any navigation.
	if err := chromedp.Run(chromeCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil).Do(ctx)
	})); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("stealth init: %w", err)
	}

	log.Println("[idx] navigating to IDX...")
	if err := chromedp.Run(chromeCtx,
		chromedp.Navigate(idxBaseURL+"/id/perusahaan-tercatat/keterbukaan-informasi"),
		chromedp.WaitReady("body"),
		chromedp.Sleep(5*time.Second),
	); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("navigate IDX: %w", err)
	}
	log.Println("[idx] page loaded")

	return chromeCtx, cancel, nil
}

// jsFetch runs a fetch() call inside the Chrome context, inheriting its cookies.
func jsFetch(ctx context.Context, url string) (string, error) {
	js := fmt.Sprintf(`
		(async () => {
			const r = await fetch(%q, {
				headers: {
					'Accept': 'application/json, text/plain, */*',
					'Referer': 'https://www.idx.co.id/id/perusahaan-tercatat/keterbukaan-informasi',
				}
			});
			return await r.text();
		})()
	`, url)

	var body string
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &body,
		func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		},
	))
	if err != nil {
		return "", fmt.Errorf("JS fetch %s: %w", url, err)
	}
	return body, nil
}

func toDisclosure(r idxReply) (model.Disclosure, bool) {
	p := r.Pengumuman
	if shouldSkip(p.Judul) {
		return model.Disclosure{}, false
	}

	date, err := parseIDXDate(p.Tanggal)
	if err != nil {
		return model.Disclosure{}, false
	}

	// Use the first non-attachment file as the main PDF.
	var pdfURL string
	for _, f := range r.Attachments {
		if !f.IsAttachment {
			pdfURL = f.FullSavePath
			break
		}
	}

	emiten := strings.TrimSpace(p.KodeEmiten)
	return model.Disclosure{
		Emiten:   emiten,
		Title:    strings.TrimSpace(p.Judul),
		Date:     date,
		PDFURL:   pdfURL,
		Category: p.Jenis,
		Hash:     hashDisclosure(emiten, p.Judul, p.Tanggal),
	}, true
}

func shouldSkip(judul string) bool {
	title := strings.ToLower(judul)
	for _, kw := range skipTitlePrefixes {
		if strings.Contains(title, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func parseIDXDate(s string) (time.Time, error) {
	s = strings.TrimSuffix(s, "Z")
	for _, f := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04:05.000", "2006-01-02"} {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date format: %q", s)
}

func hashDisclosure(emiten, title, date string) string {
	h := sha256.Sum256([]byte(emiten + "|" + title + "|" + date))
	return fmt.Sprintf("%x", h[:8])
}
