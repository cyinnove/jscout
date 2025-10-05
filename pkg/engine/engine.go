package engine

import (
	"context"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cyinnove/jscout/pkg/model"
	"github.com/cyinnove/jscout/utils"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Options configure the crawling engine.
type Options struct {
	AllowedHosts  []string
	ChromePath    string
	Headless      bool
	UserAgent     string
	PageTimeout   time.Duration
	WaitAfterLoad time.Duration
	MaxDepth      int
	MaxPages      int
	Concurrency   int
}

type Engine struct {
	opt Options
}

func New(opt Options) *Engine { return &Engine{opt: opt} }

// Crawl runs a scoped crawl starting from seeds and returns discovered JS records.
func (e *Engine) Crawl(seeds []string) ([]*model.JSRecord, error) {
	rootCtx := context.Background()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", e.opt.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	)
	if v := os.Getenv("JSCOUT_NO_SANDBOX"); v == "1" || v == "true" || v == "TRUE" {
		opts = append(opts, chromedp.Flag("no-sandbox", true))
	}
	if e.opt.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(e.opt.ChromePath))
	}

	alloc, cancelAlloc := chromedp.NewExecAllocator(rootCtx, opts...)
	defer cancelAlloc()
	browserCtx, cancelBrowser := chromedp.NewContext(alloc)
	defer cancelBrowser()

	type qitem struct {
		u     string
		depth int
	}

	jobs := make(chan qitem, 256)
	var wg sync.WaitGroup

	visited := make(map[string]struct{})
	seen := make(map[string]struct{})
	var mu sync.Mutex

	results := make([]*model.JSRecord, 0, 256)
	var resMu sync.Mutex

	var processed int32
	maxPages := e.opt.MaxPages
	if maxPages < 0 {
		maxPages = 0
	}

	// Seed queue
	enqueue := func(u string, d int) {
		mu.Lock()
		if _, ok := seen[u]; ok {
			mu.Unlock()
			return
		}
		seen[u] = struct{}{}
		mu.Unlock()
		wg.Add(1)
		jobs <- qitem{u: u, depth: d}
	}

	for _, s := range seeds {
		enqueue(s, 0)
	}

	// Closer goroutine
	go func() {
		wg.Wait()
		close(jobs)
	}()

	// Workers
	workers := e.opt.Concurrency
	if workers <= 0 {
		workers = 1
	}
	var wwg sync.WaitGroup
	wwg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wwg.Done()
			for item := range jobs {
				// Page limit
				if maxPages > 0 && atomic.LoadInt32(&processed) >= int32(maxPages) {
					wg.Done()
					continue
				}

				// Scope gate & visited
				pu, err := url.Parse(item.u)
				if err != nil || !utils.HostInScope(pu, e.opt.AllowedHosts) {
					wg.Done()
					continue
				}
				mu.Lock()
				if _, ok := visited[item.u]; ok {
					mu.Unlock()
					wg.Done()
					continue
				}
				visited[item.u] = struct{}{}
				mu.Unlock()

				// Tab context with timeout
				ctx, cancel := context.WithTimeout(browserCtx, e.opt.PageTimeout)
				// Run collection
				js, links, err := collectJSOnPage(ctx, item.u, e.opt.WaitAfterLoad, e.opt.UserAgent)
				cancel()

				if err == nil {
					resMu.Lock()
					results = append(results, js...)
					resMu.Unlock()

					// Enqueue links if within depth and within scope
					if item.depth < e.opt.MaxDepth {
						for _, l := range links {
							lu, err := url.Parse(l)
							if err != nil {
								continue
							}
							if utils.HostInScope(lu, e.opt.AllowedHosts) {
								// Respect page limit at enqueue time to reduce pressure
								if maxPages == 0 || atomic.LoadInt32(&processed) < int32(maxPages) {
									enqueue(lu.String(), item.depth+1)
								}
							}
						}
					}
				}

				atomic.AddInt32(&processed, 1)
				wg.Done()
			}
		}()
	}

	wwg.Wait()
	return results, nil
}

// collectJSOnPage visits a URL and returns JS resources and discovered links.
func collectJSOnPage(ctx context.Context, pageURL string, waitAfterLoad time.Duration, userAgent string) ([]*model.JSRecord, []string, error) {
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		return nil, nil, err
	}

	// Track JS responses
	records := make([]*model.JSRecord, 0, 16)
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if recv, ok := ev.(*network.EventResponseReceived); ok {
			if recv.Type == network.ResourceTypeScript && recv.Response != nil {
				rec := &model.JSRecord{
					JSURL:      recv.Response.URL,
					SourcePage: pageURL,
					Status:     recv.Response.Status,
					MIME:       recv.Response.MimeType,
					FromCache:  recv.Response.FromDiskCache || recv.Response.FromPrefetchCache || recv.Response.FromServiceWorker,
				}
				records = append(records, rec)
			}
		}
	})

	tasks := chromedp.Tasks{}
	if userAgent != "" {
		tasks = append(tasks, emulation.SetUserAgentOverride(userAgent))
	}
	tasks = append(tasks,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	)
	if err := chromedp.Run(ctx, tasks); err != nil {
		return nil, nil, err
	}
	if waitAfterLoad > 0 {
		time.Sleep(waitAfterLoad)
	}

	var links []string
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href)`, &links))
	return records, links, nil
}
