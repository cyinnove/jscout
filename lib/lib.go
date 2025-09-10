package lib

import (
    "net/url"
    "time"

    "github.com/cyinnove/jscout/engine"
    "github.com/cyinnove/jscout/pkg/model"
    "github.com/cyinnove/jscout/utils"
)

// Options controls crawling when using JScout as a library.
type Options struct {
    // Seeds to crawl. If Normalize is true, seeds may be bare hosts and will be normalized.
    Seeds []string

    // AllowedHosts restricts crawl scope by host suffix. If empty, defaults to seed hosts.
    AllowedHosts []string

    // Browser/runtime
    ChromePath string
    Headless   bool
    UserAgent  string

    // Crawl behavior
    PageTimeout   time.Duration
    WaitAfterLoad time.Duration
    MaxDepth      int
    MaxPages      int
    Concurrency   int

    // Convenience
    Normalize      bool   // normalize seeds to URLs
    DefaultScheme  string // scheme to use when normalizing (default "https")
    FilterJSInScope bool  // keep only JS whose host is within AllowedHosts
}

// DefaultOptions returns a sensible default Options value.
func DefaultOptions() Options {
    return Options{
        Headless:       true,
        PageTimeout:    30 * time.Second,
        WaitAfterLoad:  3 * time.Second,
        MaxDepth:       1,
        MaxPages:       100,
        Concurrency:    4,
        Normalize:      true,
        DefaultScheme:  "https",
        FilterJSInScope: true,
    }
}

// Crawl runs the crawl with the provided options and returns discovered JS records.
func Crawl(o Options) ([]model.JSRecord, error) {
    seeds := make([]string, 0, len(o.Seeds))
    if o.Normalize {
        scheme := o.DefaultScheme
        if scheme == "" {
            scheme = "https"
        }
        for _, s := range o.Seeds {
            if ns, err := utils.NormalizeSeed(s, scheme); err == nil {
                seeds = append(seeds, ns)
            }
        }
    } else {
        seeds = append(seeds, o.Seeds...)
    }

    // Scope default to seed hosts if not provided
    allowed := append([]string(nil), o.AllowedHosts...)
    if len(allowed) == 0 {
        seen := map[string]struct{}{}
        for _, s := range seeds {
            if u, err := url.Parse(s); err == nil && u.Host != "" {
                h := u.Hostname()
                if _, ok := seen[h]; !ok {
                    allowed = append(allowed, h)
                    seen[h] = struct{}{}
                }
            }
        }
    }

    engOpt := engine.Options{
        AllowedHosts:  allowed,
        ChromePath:    o.ChromePath,
        Headless:      o.Headless,
        UserAgent:     o.UserAgent,
        PageTimeout:   o.PageTimeout,
        WaitAfterLoad: o.WaitAfterLoad,
        MaxDepth:      o.MaxDepth,
        MaxPages:      o.MaxPages,
        Concurrency:   o.Concurrency,
    }

    eng := engine.New(engOpt)
    records, err := eng.Crawl(seeds)
    if err != nil {
        return nil, err
    }

    if o.FilterJSInScope {
        filtered := records[:0]
        for _, r := range records {
            if ju, err := url.Parse(r.JSURL); err == nil {
                if utils.HostInScope(ju, allowed) {
                    filtered = append(filtered, r)
                }
            }
        }
        records = filtered
    }

    return records, nil
}

