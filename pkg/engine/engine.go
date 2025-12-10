package engine

import (
	"context"
	"net/url"
	"os"
	"strings"
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

	// Track JS responses from network events
	records := make([]*model.JSRecord, 0, 16)
	seenURLs := make(map[string]struct{})
	var mu sync.Mutex
	
	// Helper function to remove query parameters and fragments from URL
	cleanURL := func(urlStr string) string {
		// Remove query parameters and fragments
		return strings.Split(strings.Split(urlStr, "#")[0], "?")[0]
	}
	
	// Helper function to check if URL is a JavaScript file
	isJavaScriptFile := func(urlStr string, mimeType string) bool {
		// Remove query parameters and fragments
		urlWithoutQuery := cleanURL(urlStr)
		
		// Check MIME type first
		jsMimeTypes := []string{
			"application/javascript",
			"text/javascript",
			"application/x-javascript",
			"application/ecmascript",
			"text/ecmascript",
		}
		for _, mime := range jsMimeTypes {
			if strings.HasPrefix(mimeType, mime) {
				return true
			}
		}
		
		// Check file extension - must end with .js
		if strings.HasSuffix(strings.ToLower(urlWithoutQuery), ".js") {
			return true
		}
		
		// For Next.js chunks, verify it's actually a .js file
		if strings.Contains(urlStr, "/_next/static/chunks/") {
			// Extract the filename part
			parts := strings.Split(urlStr, "/")
			if len(parts) > 0 {
				filename := parts[len(parts)-1]
				filenameWithoutQuery := cleanURL(filename)
				if strings.HasSuffix(strings.ToLower(filenameWithoutQuery), ".js") {
					return true
				}
			}
		}
		
		return false
	}
	
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if recv, ok := ev.(*network.EventResponseReceived); ok {
			if recv.Response != nil {
				url := recv.Response.URL
				mimeType := recv.Response.MimeType
				
				// Only capture verified JS files
				// Always verify to ensure it's actually a .js file (not SVG, CSS, images, etc.)
				isJS := isJavaScriptFile(url, mimeType)
				
				if isJS {
					mu.Lock()
					// Clean URL (remove query params and fragments) for storage and deduplication
					cleanJsURL := cleanURL(url)
					if _, exists := seenURLs[cleanJsURL]; !exists {
						seenURLs[cleanJsURL] = struct{}{}
						rec := &model.JSRecord{
							JSURL:      cleanJsURL, // Store without query params
							SourcePage: pageURL,
							Status:     recv.Response.Status,
							MIME:       mimeType,
							FromCache:  recv.Response.FromDiskCache || recv.Response.FromPrefetchCache || recv.Response.FromServiceWorker,
						}
						records = append(records, rec)
					}
					mu.Unlock()
				}
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

	// Interact with the page to trigger route-specific JS loads (Next.js, etc.)
	// Scroll to trigger lazy-loaded content
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`
		window.scrollTo(0, document.body.scrollHeight / 2);
	`, nil))
	time.Sleep(500 * time.Millisecond)
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`
		window.scrollTo(0, document.body.scrollHeight);
	`, nil))
	time.Sleep(500 * time.Millisecond)
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`
		window.scrollTo(0, 0);
	`, nil))
	time.Sleep(500 * time.Millisecond)

	// Hover over links to trigger prefetch/preload (Next.js does this)
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`
		(function() {
			const links = Array.from(document.querySelectorAll('a[href]')).slice(0, 20);
			links.forEach(link => {
				try {
					const event = new MouseEvent('mouseenter', { bubbles: true, cancelable: true });
					link.dispatchEvent(event);
				} catch(e) {}
			});
		})()
	`, nil))
	time.Sleep(1 * time.Second)

	// Try to extract and trigger Next.js routes if available
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`
		(function() {
			// Try to access Next.js router and trigger route prefetching
			if (window.__NEXT_DATA__ && window.__NEXT_DATA__.buildId) {
				// Next.js is present, try to trigger route prefetching
				const links = Array.from(document.querySelectorAll('a[href]'));
				links.slice(0, 10).forEach(link => {
					try {
						// Trigger mouseenter which Next.js listens to for prefetching
						link.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }));
					} catch(e) {}
				});
			}
		})()
	`, nil))
	time.Sleep(1 * time.Second)

	// Wait for network to be idle to capture lazy-loaded chunks
	_ = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Wait a bit more for any pending network requests triggered by interactions
		time.Sleep(2 * time.Second)
		return nil
	}))

	// Extract JS files from multiple sources: script tags, preload links, HTML source, and Next.js data
	var allJSURLs []string
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`
		(function() {
			const jsURLs = new Set();
			const baseURL = window.location.href;
			const origin = window.location.origin;
			
			// Helper to clean URL (remove query params and fragments)
			function cleanURL(urlStr) {
				return urlStr.split('?')[0].split('#')[0];
			}
			
			// Helper to check if URL is a JavaScript file
			function isJSFile(urlStr) {
				if (!urlStr || !urlStr.startsWith('http')) return false;
				// Remove query params and fragments
				const urlWithoutQuery = cleanURL(urlStr).toLowerCase();
				// Must end with .js
				return urlWithoutQuery.endsWith('.js');
			}
			
			// Extract from script tags
			Array.from(document.querySelectorAll('script[src]')).forEach(s => {
				try {
					const url = new URL(s.src, baseURL).href;
					if (isJSFile(url)) jsURLs.add(cleanURL(url));
				} catch(e) {}
			});
			
			// Extract from preload/prefetch link tags (only if as="script" or ends with .js)
			Array.from(document.querySelectorAll('link[rel="preload"], link[rel="prefetch"], link[rel="modulepreload"]')).forEach(link => {
				const href = link.href;
				if (link.as === 'script' || isJSFile(href)) {
					try {
						const url = new URL(href, baseURL).href;
						if (isJSFile(url)) jsURLs.add(cleanURL(url));
					} catch(e) {}
				}
			});
			
			// Extract from Next.js __NEXT_DATA__ (contains all route chunks)
			try {
				if (window.__NEXT_DATA__) {
					const nextData = window.__NEXT_DATA__;
					// Extract from pageProps or other Next.js data structures
					if (nextData.buildId) {
						const buildId = nextData.buildId;
						// Next.js chunks are typically in /_next/static/chunks/ or /_next/static/{buildId}/
						// We can't enumerate all routes, but we can extract referenced ones
					}
					// Extract from __NEXT_DATA__.page if it contains chunk references
					if (nextData.page) {
						// Some Next.js setups expose chunk info here
					}
				}
			} catch(e) {}
			
			// Extract from HTML source (for Next.js __NEXT_DATA__ or other embedded references)
			try {
				const html = document.documentElement.outerHTML;
				// Look for script src patterns - must end with .js
				const scriptSrcRegex = /src=["']([^"']+\.js[^"']*)["']/gi;
				let match;
				while ((match = scriptSrcRegex.exec(html)) !== null) {
					try {
						const url = new URL(match[1], baseURL).href;
						if (isJSFile(url)) jsURLs.add(cleanURL(url));
					} catch(e) {}
				}
				// Look for href patterns in link tags - must end with .js
				const linkHrefRegex = /<link[^>]+href=["']([^"']+\.js[^"']*)["']/gi;
				while ((match = linkHrefRegex.exec(html)) !== null) {
					try {
						const url = new URL(match[1], baseURL).href;
						if (isJSFile(url)) jsURLs.add(cleanURL(url));
					} catch(e) {}
				}
				// Look for Next.js chunk patterns in the HTML - must end with .js
				const nextChunkRegex = /\/_next\/static\/[^"'\s>]+\.js[^"'\s>]*/gi;
				while ((match = nextChunkRegex.exec(html)) !== null) {
					try {
						const url = new URL(match[0], origin).href;
						if (isJSFile(url)) jsURLs.add(cleanURL(url));
					} catch(e) {}
				}
			} catch(e) {}
			
			// Also check for any script elements that might have been added dynamically
			try {
				const allScripts = document.querySelectorAll('script');
				allScripts.forEach(script => {
					if (script.src) {
						try {
							const url = new URL(script.src, baseURL).href;
							if (isJSFile(url)) jsURLs.add(cleanURL(url));
						} catch(e) {}
					}
				});
			} catch(e) {}
			
			return Array.from(jsURLs);
		})()
	`, &allJSURLs))
	
	// Add all discovered scripts that weren't captured by network events
	// Filter to ensure only .js files are added
	mu.Lock()
	for _, jsURL := range allJSURLs {
		// URLs from DOM extraction are already cleaned, but double-check
		cleanJsURL := cleanURL(jsURL)
		
		// Double-check it's actually a JS file
		if !strings.HasSuffix(strings.ToLower(cleanJsURL), ".js") {
			continue // Skip non-JS files
		}
		
		if _, exists := seenURLs[cleanJsURL]; !exists {
			seenURLs[cleanJsURL] = struct{}{}
			rec := &model.JSRecord{
				JSURL:      cleanJsURL, // Store without query params
				SourcePage: pageURL,
				Status:     200, // Assume success if referenced
				MIME:       "application/javascript",
				FromCache:  false,
			}
			records = append(records, rec)
		}
	}
	mu.Unlock()

	var links []string
	_ = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href)`, &links))
	return records, links, nil
}
