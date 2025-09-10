package runner

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"jscout/config"
	"jscout/engine"

	// "jscout/pkg/model"
	"jscout/utils"

	"github.com/cyinnove/logify"
)

type Runner struct {
	Cfg config.Config
}

func New(cfg config.Config) *Runner { return &Runner{Cfg: cfg} }

func (r *Runner) Run() error {
	// Collect all seeds
	seedsRaw := make([]string, 0, len(r.Cfg.SeedsRaw)+4)
	seedsRaw = append(seedsRaw, r.Cfg.SeedsRaw...)
	if r.Cfg.ReadStdin {
		if info, _ := os.Stdin.Stat(); (info.Mode() & os.ModeCharDevice) == 0 {
			s := bufio.NewScanner(os.Stdin)
			for s.Scan() {
				seedsRaw = append(seedsRaw, s.Text())
			}
			if err := s.Err(); err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
		}
	}
	if r.Cfg.SeedsFile != "" {
		lines, err := utils.ReadLines(r.Cfg.SeedsFile)
		if err != nil {
			return fmt.Errorf("read seeds file: %w", err)
		}
		seedsRaw = append(seedsRaw, lines...)
	}
	if len(seedsRaw) == 0 {
		return fmt.Errorf("no seeds provided; use -u, -l or --stdin")
	}

	// Normalize seeds
	seeds := make([]string, 0, len(seedsRaw))
	for _, s := range seedsRaw {
		ns, err := utils.NormalizeSeed(s, r.Cfg.Scheme)
		if err == nil {
			seeds = append(seeds, ns)
		}
	}
	if len(seeds) == 0 {
		return fmt.Errorf("no valid seeds after normalization")
	}

	// Build scope
	allowed := make([]string, 0, 8)
	if r.Cfg.ScopeCSV != "" {
		for _, p := range strings.Split(r.Cfg.ScopeCSV, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				allowed = append(allowed, p)
			}
		}
	}
	if r.Cfg.ScopeFile != "" {
		lines, err := utils.ReadLines(r.Cfg.ScopeFile)
		if err != nil {
			return fmt.Errorf("read scope file: %w", err)
		}
		allowed = append(allowed, lines...)
	}
	if len(allowed) == 0 {
		// Default to seed hosts
		seenHosts := map[string]struct{}{}
		for _, s := range seeds {
			u, err := url.Parse(s)
			if err != nil || u.Host == "" {
				continue
			}
			h := strings.ToLower(u.Host)
			if _, ok := seenHosts[h]; !ok {
				allowed = append(allowed, h)
				seenHosts[h] = struct{}{}
			}
		}
	}
	r.Cfg.ScopeList = allowed

	// Build engine options
	opt := engine.Options{
		AllowedHosts:  allowed,
		ChromePath:    r.Cfg.ChromePath,
		Headless:      r.Cfg.Headless,
		UserAgent:     r.Cfg.UserAgent,
		PageTimeout:   time.Duration(r.Cfg.PageTimeoutSec) * time.Second,
		WaitAfterLoad: time.Duration(r.Cfg.WaitSeconds) * time.Second,
		MaxDepth:      r.Cfg.MaxDepth,
		MaxPages:      r.Cfg.MaxPages,
		Concurrency:   r.Cfg.Concurrency,
	}

	eng := engine.New(opt)
	records, err := eng.Crawl(seeds)
	if err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	// Optional JS host filtering by scope
	if r.Cfg.JSInScope {
		filtered := records[:0]
		for _, rec := range records {
			ju, err := url.Parse(rec.JSURL)
			if err != nil {
				continue
			}
			if utils.HostInScope(ju, allowed) {
				filtered = append(filtered, rec)
			}
		}
		records = filtered
	}

	// Write output
	var out io.Writer
	var file *os.File
	if r.Cfg.OutputPath == "-" || r.Cfg.OutputPath == "" {
		out = os.Stdout
	} else {
		if err := utils.EnsureDirOf(r.Cfg.OutputPath); err != nil {
			return err
		}
		fh, err := os.Create(r.Cfg.OutputPath)
		if err != nil {
			return err
		}
		defer fh.Close()
		file = fh
		out = fh
	}

	if err := utils.WriteOutput(out, r.Cfg.Format, r.Cfg.Unique, records); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if file != nil {
		logify.Infof("Saved %d records to %s", len(records), r.Cfg.OutputPath)
	}
	return nil
}

// DetectChromePath remains in utils/ or could be in engine; omitted here for brevity.
