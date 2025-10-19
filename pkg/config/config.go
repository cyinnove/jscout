package config

// Config holds all runtime options for jscout.
type Config struct {
	// Input
	URL       string   // deprecated, use URLs instead
	URLs      []string // multiple URLs via -u flag
	SeedsFile string
	ReadStdin bool
	Scheme    string

	// Scope
	ScopeCSV  string
	ScopeFile string
	ScopeList []string // final computed list

	// Crawl controls
	MaxDepth       int
	MaxPages       int
	WaitSeconds    int
	PageTimeoutSec int
	Concurrency    int

	// Browser
	ChromePath string
	Headless   bool
	UserAgent  string

	// Output
	OutputPath string
	Format     string
	Unique     bool
	JSInScope  bool

	// Seeds (final normalized elsewhere)
	SeedsRaw []string

	// UI
	NoBanner bool
}

// Defaults returns a Config initialized with sane defaults.
func Defaults() Config {
	return Config{
		Scheme:         "https",
		MaxDepth:       1,
		MaxPages:       100,
		WaitSeconds:    3,
		PageTimeoutSec: 30,
		Concurrency:    4,
		Headless:       true,
		OutputPath:     "-",
		Format:         "txt",
		Unique:         true,
		JSInScope:      true,
		NoBanner:       false,
	}
}
