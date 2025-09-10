package utils

import (
    "bufio"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "runtime"
    "strings"

    "github.com/cyinnove/logify"
)

// DetectChromePath checks PATH for chrome/chromium binaries (Linux/macOS).
// On Windows, chromedp often resolves automatically if empty.
func DetectChromePath() string {
    candidates := []string{
        "google-chrome",
        "google-chrome-stable",
        "chromium",
        "chromium-browser",
    }
    for _, c := range candidates {
        if path, err := exec.LookPath(c); err == nil {
            return path
        }
    }
    return ""
}

// EnsureChromePathLinux verifies Chrome/Chromium availability on Linux.
// If a path is provided, it validates it; otherwise tries detection.
// When unresolved and running interactively, prompts the user for a path.
// Returns the resolved executable path or an error with install hints.
func EnsureChromePathLinux(inputPath string) (string, error) {
    if runtime.GOOS != "linux" {
        // No-op for non-Linux platforms; return input as-is (may be empty).
        return inputPath, nil
    }

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

    // If interactive TTY, prompt user.
    if isInteractiveStdin() {
        reader := bufio.NewReader(os.Stdin)
        for i := 0; i < 3; i++ {
            logify.Warningf("Chrome/Chromium not found. Enter full path to chrome (or leave blank to abort): ")
            line, _ := reader.ReadString('\n')
            line = strings.TrimSpace(line)
            if line == "" {
                break
            }
            if p, err := resolveChrome(line); err == nil {
                return p, nil
            }
            logify.Warningf("Invalid path or command. Try again.")
        }
    }

    // Give up with install hints.
    return "", errors.New("chrome/chromium not found on Linux. Install it or pass --chrome-path.\n" + linuxInstallHints())
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

func isInteractiveStdin() bool {
    info, err := os.Stdin.Stat()
    if err != nil { return false }
    return (info.Mode() & os.ModeCharDevice) != 0
}

func linuxInstallHints() string {
    return strings.Join([]string{
        "Install Chrome/Chromium:",
        "  Debian/Ubuntu:  sudo apt-get install chromium-browser    # or install google-chrome-stable",
        "  Fedora:         sudo dnf install chromium",
        "  Arch:           sudo pacman -S chromium",
        "Alternatively, download Google Chrome: https://www.google.com/chrome/",
    }, "\n")
}
