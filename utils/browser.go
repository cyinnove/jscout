package utils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cyinnove/logify"
)

// DetectChromePath checks PATH for chrome/chromium binaries on all platforms.
func DetectChromePath() string {
	var candidates []string

	switch runtime.GOOS {
	case "windows":
		// First check common Windows installation paths
		commonPaths := []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Users\` + os.Getenv("USERNAME") + `\AppData\Local\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		}

		// Check if these paths exist
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		// Fallback to PATH lookup
		candidates = []string{
			"chrome.exe",
			"chrome",
			"google-chrome.exe",
			"google-chrome",
			"chromium.exe",
			"chromium",
			"msedge.exe",
			"msedge",
		}
	case "darwin": // macOS
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"google-chrome",
			"chromium",
		}
	default: // Linux and others
		candidates = []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
		}
	}

	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
	}
	return ""
}

// EnsureChromePath verifies Chrome/Chromium availability on all platforms.
// If a path is provided, it validates it; otherwise tries detection.
// When unresolved and running interactively, prompts the user for a path.
// Returns the resolved executable path or an error with install hints.
func EnsureChromePath(inputPath string) (string, error) {
	// If user provided a path or command, try it first.
	if strings.TrimSpace(inputPath) != "" {
		if p, err := resolveChrome(inputPath); err == nil {
			return p, nil
		}
	}

	// Try auto-detect on PATH.
	if p := DetectChromePath(); p != "" {
		return p, nil
	}

	// Skip interactive prompt to avoid hanging in WSL/containers
	// Users can use --chrome-path flag instead

	// Give up with install hints.
	return "", errors.New("chrome/chromium not found. Install it or pass --chrome-path.\n" + getInstallHints())
}

func resolveChrome(s string) (string, error) {
	// If it's an absolute/relative path, check it; otherwise try LookPath.
	if strings.Contains(s, string(os.PathSeparator)) {
		if st, err := os.Stat(s); err == nil && !st.IsDir() {
			return s, nil
		}
		return "", fmt.Errorf("not a file: %s", s)
	}
	p, err := exec.LookPath(s)
	if err != nil {
		return "", err
	}
	return p, nil
}

// ValidateChromePath tests if the Chrome binary actually works by running it with --version
func ValidateChromePath(chromePath string) error {
	if chromePath == "" {
		return errors.New("chrome path is empty")
	}

	// Test if Chrome can run by checking version
	cmd := exec.Command(chromePath, "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chrome binary validation failed: %w", err)
	}

	return nil
}

func readLineWithTimeout(timeout time.Duration) string {
	type result struct {
		line string
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		resultChan <- result{strings.TrimSpace(line), err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			logify.Debugf("Error reading input: %v", res.err)
			return ""
		}
		return res.line
	case <-time.After(timeout):
		logify.Debugf("Input timeout after %v, continuing without user input", timeout)
		return ""
	}
}

func isInteractiveStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func getInstallHints() string {
	// Check if running in WSL
	isWSL := false
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/version"); err == nil {
			isWSL = strings.Contains(strings.ToLower(string(data)), "microsoft")
		}
	}

	switch runtime.GOOS {
	case "windows":
		return strings.Join([]string{
			"Install Chrome/Chromium on Windows:",
			"  1. Download from: https://www.google.com/chrome/",
			"  2. Or install via Chocolatey: choco install googlechrome",
			"  3. Or install via Winget: winget install Google.Chrome",
			"  4. Or use Microsoft Edge (usually pre-installed)",
		}, "\n")
	case "darwin":
		return strings.Join([]string{
			"Install Chrome/Chromium on macOS:",
			"  1. Download from: https://www.google.com/chrome/",
			"  2. Or install via Homebrew: brew install --cask google-chrome",
			"  3. Or install Chromium: brew install --cask chromium",
		}, "\n")
	default: // Linux
		if isWSL {
			return strings.Join([]string{
				"Install Chrome/Chromium in WSL:",
				"  1. Install in Windows: Download from https://www.google.com/chrome/",
				"  2. Use Windows Chrome from WSL: --chrome-path \"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe\"",
				"  3. Or install in WSL: sudo apt-get install chromium-browser",
				"  4. Or use Windows Edge: --chrome-path \"/mnt/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe\"",
			}, "\n")
		}
		return strings.Join([]string{
			"Install Chrome/Chromium on Linux:",
			"  Debian/Ubuntu:  sudo apt-get install chromium-browser    # or install google-chrome-stable",
			"  Fedora:         sudo dnf install chromium",
			"  Arch:           sudo pacman -S chromium",
			"  Or download from: https://www.google.com/chrome/",
		}, "\n")
	}
}
