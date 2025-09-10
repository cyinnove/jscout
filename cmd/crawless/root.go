package main

import (
	"github.com/cyinnove/logify"
	"github.com/spf13/cobra"

	"github.com/cyinnove/jscout/config"
	"github.com/cyinnove/jscout/runner"
	"github.com/cyinnove/jscout/utils"
)

func newRootCmd() *cobra.Command {
	cfg := config.Defaults()

	cmd := &cobra.Command{
		Use:   "jscout",
		Short: "Headless JS crawler for bug hunters",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.NoBanner {
				utils.PrintBanner()
			}

			// Linux-only Chrome verification
			p, err := utils.EnsureChromePathLinux(cfg.ChromePath)
			if err != nil {
				return err
			}
			cfg.ChromePath = p

			r := runner.New(cfg)
			if err := r.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	// Inputs
	cmd.Flags().StringVarP(&cfg.URL, "url", "u", cfg.URL, "Single seed URL or host (e.g. https://example.com or example.com)")
	cmd.Flags().StringVarP(&cfg.SeedsFile, "list", "l", cfg.SeedsFile, "File with seed URLs/hosts (one per line)")
	cmd.Flags().BoolVar(&cfg.ReadStdin, "stdin", cfg.ReadStdin, "Read seeds from STDIN (one per line)")
	cmd.Flags().StringVar(&cfg.Scheme, "scheme", cfg.Scheme, "Default scheme for seeds without scheme")

	// Scope
	cmd.Flags().StringVar(&cfg.ScopeCSV, "scope", cfg.ScopeCSV, "Comma-separated allowed host suffixes (e.g. example.com,cdn.example.com)")
	cmd.Flags().StringVar(&cfg.ScopeFile, "scope-file", cfg.ScopeFile, "File with allowed host suffixes (one per line)")

	// Crawl
	cmd.Flags().IntVar(&cfg.MaxDepth, "max-depth", cfg.MaxDepth, "Max crawl depth from seeds")
	cmd.Flags().IntVar(&cfg.MaxPages, "max-pages", cfg.MaxPages, "Max pages to visit (0 = unlimited)")
	cmd.Flags().IntVarP(&cfg.Concurrency, "concurrency", "c", cfg.Concurrency, "Concurrent pages (tabs) to process")
	cmd.Flags().IntVar(&cfg.WaitSeconds, "wait", cfg.WaitSeconds, "Seconds to wait after load for dynamic scripts")
	cmd.Flags().IntVar(&cfg.PageTimeoutSec, "page-timeout", cfg.PageTimeoutSec, "Per-page timeout in seconds")

	// Browser
	cmd.Flags().StringVar(&cfg.ChromePath, "chrome-path", cfg.ChromePath, "Path to Chrome/Chromium binary (optional)")
	cmd.Flags().BoolVar(&cfg.Headless, "headless", cfg.Headless, "Run browser in headless mode")
	cmd.Flags().StringVar(&cfg.UserAgent, "user-agent", cfg.UserAgent, "Custom User-Agent for requests (optional)")

	// Output
	cmd.Flags().StringVarP(&cfg.OutputPath, "output", "o", cfg.OutputPath, "Output path or '-' for STDOUT")
	cmd.Flags().StringVar(&cfg.Format, "format", cfg.Format, "Output format: txt|jsonl|csv")
	cmd.Flags().BoolVar(&cfg.Unique, "unique", cfg.Unique, "De-duplicate JS URLs in output (txt mode)")
	cmd.Flags().BoolVar(&cfg.JSInScope, "js-in-scope", cfg.JSInScope, "Only output JS URLs whose host matches scope")
	cmd.Flags().BoolVar(&cfg.NoBanner, "no-banner", cfg.NoBanner, "Disable startup banner")

	// Map -u to cfg.SeedsRaw for runner
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if cfg.URL != "" {
			cfg.SeedsRaw = append(cfg.SeedsRaw, cfg.URL)
		}
	}

	return cmd
}

func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		logify.Fatalf("%v", err)
	}
}

