# 🕷️ JSCOUT – Headless JS Crawler for Bug Hunters

<div align="center">

```
   __  __                 _   
   \ \/ _\ ___ ___  _   _| |_ 
    \ \ \ / __/ _ \| | | | __|
 /\_/ /\ \ (_| (_) | |_| | |_ 
 \___/\__/\___\___/ \__,_|\__|  @CyInnove
```

**Fast, scope-aware, headless crawling framework to extract Dynamic JS files.**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com/)

</div>

---

## 🚀 Quick Start

```bash
# Install JSCOUT
go install github.com/cyinnove/jscout/cmd/jscout@latest

# Start crawling
jscout -u https://example.com -max-depth 1 -o -
```

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🕸️ **Headless Browser** | Chrome/Chromium powered crawling with `chromedp` |
| 🎯 **Scoped BFS** | Host suffix allow-list for targeted crawling |
| ⚡ **Dynamic JS Extraction** | Captures both static and dynamic JavaScript files |
| 📥 **Flexible Input** | URL, file list, or stdin with auto-normalization |
| 📊 **Multiple Formats** | txt, jsonl, csv output (unique txt by default) |
| 🔄 **Concurrency Control** | Configurable parallel crawling for speed |
| 🎛️ **Smart Filtering** | Optional JS-in-scope filtering |
| ⚙️ **Customizable** | User-Agent, timeouts, waits, and more |
| 🐳 **Docker Ready** | Pre-built image with Chromium included |

---

## 📦 Installation

### 🎯 Quick Install (Recommended)

```bash
go install github.com/cyinnove/jscout/cmd/jscout@latest
```

**Requirements:** Go 1.22+ and Chrome/Chromium

### 🔨 Build from Source

```bash
git clone https://github.com/cyinnove/jscout
cd jscout
go build -o jscout ./cmd/jscout
```

Binary will be at `./jscout` (Linux/macOS) or `jscout.exe` (Windows).

### 📚 Use as a Library

```bash
go get github.com/cyinnove/jscout@latest
```

**Example Usage:**

```go
package main

import (
    "fmt"
    "github.com/cyinnove/jscout/lib"
)

func main() {
    opts := lib.DefaultOptions()
    opts.Seeds = []string{"https://example.com"}
    recs, err := lib.Crawl(opts)
    if err != nil { panic(err) }
    fmt.Printf("found %d JS files\n", len(recs))
}
```

### 🐳 Docker

**Build locally:**
```bash
docker build -t cyinnove/jscout:latest .
```

**Run:**
```bash
docker run --rm -it \
  --network host \
  cyinnove/jscout:latest -u https://example.com -max-depth 1 -o -
```

> **📝 Notes:**
> - Image includes `chromium`; Chrome sandbox is disabled via `JSCOUT_NO_SANDBOX=1` for container compatibility
> - Use `-o -` to write results to stdout
> - To read local files, mount volumes: `-v "$PWD:/data"` and `-l /data/seeds.txt`

---

## 🎯 Usage Examples

### 🚀 Basic Usage

```bash
jscout -u https://news.ycombinator.com -max-depth 0 -format txt -o -

# See all available flags
jscout --help
```

### 🔍 Advanced Crawling

**Depth + scope file + concurrency:**
```bash
jscout -l seeds.txt \
  --scope-file scope.txt \
  --max-depth 2 --max-pages 500 \
  --concurrency 6 \
  -format jsonl -o results.jsonl
```

**Stdin seeds:**
```bash
cat domains.txt | jscout --stdin --scheme https -o -
```

**Include third-party JS:**
```bash
jscout -u https://example.com --js-in-scope=false -o -
```

**Custom User-Agent and Chrome path:**
```bash
jscout -u https://target.tld \
  --user-agent "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118 Safari/537.36" \
  --chrome-path /usr/bin/chromium-browser
```

---

## ⚙️ CLI Flags Reference

### 📥 Input Options
| Flag | Description | Default |
|------|-------------|---------|
| `-u` | Single seed (URL or host) | - |
| `-l` | File with seeds (one per line) | - |
| `--stdin` | Read seeds from STDIN | - |
| `--scheme` | Default scheme for host-only seeds | `https` |

### 🎯 Scope Options
| Flag | Description | Default |
|------|-------------|---------|
| `--scope` | Comma-separated allowed host suffixes | Seed hosts |
| `--scope-file` | File with allowed suffixes (one per line) | - |

### 🕷️ Crawl Options
| Flag | Description | Default |
|------|-------------|---------|
| `--max-depth` | Crawl depth from seeds | `1` |
| `--max-pages` | Limit pages (0 = unlimited) | `100` |
| `--concurrency` | Concurrent pages | `4` |
| `--wait` | Seconds after load for dynamic JS | `3` |
| `--page-timeout` | Per-page timeout in seconds | `30` |

### 🌐 Browser Options
| Flag | Description | Default |
|------|-------------|---------|
| `--headless` | Run headless | `true` |
| `--chrome-path` | Explicit Chrome/Chromium path | Auto-detect |
| `--user-agent` | Custom UA string | Default Chrome |

### 📊 Output Options
| Flag | Description | Default |
|------|-------------|---------|
| `-o` | Output path or `-` for stdout | `-` |
| `--format` | Output format: txt\|jsonl\|csv | `txt` |
| `--unique` | De-duplicate JS URLs in txt mode | `true` |
| `--js-in-scope` | Only output JS whose host matches scope | `true` |
| `--no-banner` | Disable the startup ASCII banner | `false` |

---

## 📝 Additional Information

### 🔍 Logging
Uses `github.com/cyinnove/logify`. To adjust verbosity in code, set `logify.MaxLevel` early in `main`. A `--log-level` flag can be added on request.

### ⚠️ Important Notes

- **Linux**: JSCOUT verifies Chrome/Chromium availability. If not found and interactive, it prompts for a path; otherwise it errors with install hints.
- **Docker**: `JSCOUT_NO_SANDBOX=1` is set by default to make Chromium work as root. Unset it by overriding env if you run with a user that can use the sandbox.

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Made with ❤️ by [@CyInnove](https://github.com/cyinnove)**

[⭐ Star this repo](https://github.com/cyinnove/jscout) • [🐛 Report Bug](https://github.com/cyinnove/jscout/issues) • [💡 Request Feature](https://github.com/cyinnove/jscout/issues)

</div>



