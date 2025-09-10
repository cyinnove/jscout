# JScout – Headless JS Crawler for Bug Hunters

```
   __  __                 _   
   \ \/ _\ ___ ___  _   _| |_ 
    \ \ \ / __/ _ \| | | | __|
 /\_/ /\ \ (_| (_) | |_| | |_ 
 \___/\__/\___\___/ \__,_|\__|  @CyInnove
    
Fast, scope-aware, headless crawling framework to extract Dynamic JS files.
```

Fast, scope-aware, headless crawling framework to extract JavaScript files from target sites. Built with `chromedp` for realistic page loads and dynamic JS capture.

## Features

- Headless browser crawl with Chrome/Chromium (chromedp)
- Scoped BFS crawling (host suffix allow-list)
- Extracts JS from network events (dynamic and static)
- Flexible inputs: URL, list file, stdin; scheme auto-normalization
- Output formats: txt, jsonl, csv (unique txt by default)
- Concurrency control for faster crawling
- Optional JS-in-scope filtering (keep only in-scope JS hosts)
- Custom User-Agent, timeouts, and waits
- Docker image with Chromium preinstalled

## Install

### Go

Requires Go 1.22+ and Chrome/Chromium.

```
git clone https://github.com/cyinnove/jscout
cd jscout
go build -o jscout ./cmd/crawless
```

Binary will be at `./jscout` (Linux/macOS) or `jscout.exe` (Windows).

### Docker

Build the image locally:

```
docker build -t cyinnove/jscout:latest .
```

Run:

```
docker run --rm -it \
  --network host \
  cyinnove/jscout:latest -u https://example.com -max-depth 1 -o -
```

Notes:
- Image includes `chromium`; Chrome sandbox is disabled via `CRAWLESS_NO_SANDBOX=1` for container compatibility.
- Use `-o -` to write results to stdout.
- To read local files, mount volumes, e.g. `-v "$PWD:/data"` and `-l /data/seeds.txt`.

## Usage

Basic:

```
jscout -u https://news.ycombinator.com -max-depth 0 -format txt -o -

# Tip: see all flags
jscout --help
```

Depth + scope file + concurrency:

```
jscout -l seeds.txt \
  --scope-file scope.txt \
  --max-depth 2 --max-pages 500 \
  --concurrency 6 \
  -format jsonl -o results.jsonl
```

Stdin seeds:

```
cat domains.txt | jscout --stdin --scheme https -o -
```

Include third-party JS:

```
jscout -u https://example.com --js-in-scope=false -o -
```

Custom User-Agent and Chrome path:

```
jscout -u https://target.tld \
  --user-agent "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118 Safari/537.36" \
  --chrome-path /usr/bin/chromium-browser
```

## CLI Flags

- Inputs
  - `-u` single seed (URL or host)
  - `-l` file with seeds (one per line)
  - `--stdin` read seeds from STDIN
  - `--scheme` default scheme for host-only seeds (default: https)
- Scope
  - `--scope` comma-separated allowed host suffixes (e.g., example.com,cdn.example.com)
  - `--scope-file` file with allowed suffixes (one per line)
  - Default scope is seed hosts
- Crawl
  - `--max-depth` crawl depth from seeds (default: 1)
  - `--max-pages` limit pages (default: 100, 0 = unlimited)
  - `--concurrency` concurrent pages (default: 4)
  - `--wait` seconds after load for dynamic JS (default: 3)
  - `--page-timeout` per-page timeout in seconds (default: 30)
- Browser
  - `--headless` run headless (default: true)
  - `--chrome-path` explicit Chrome/Chromium path (optional)
  - `--user-agent` custom UA string (optional)
- Output
  - `-o` output path or `-` for stdout (default: -)
  - `--format` txt|jsonl|csv (default: txt)
  - `--unique` de-duplicate JS URLs in txt mode (default: true)
  - `--js-in-scope` only output JS whose host matches scope (default: true)
  - `--no-banner` disable the startup ASCII banner

## Logging

Uses `github.com/cyinnove/logify`. To adjust verbosity in code, set `logify.MaxLevel` early in `main`. A `--log-level` flag can be added on request.

## Notes

- On Linux, JScout verifies Chrome/Chromium availability. If not found and interactive, it prompts for a path; otherwise it errors with install hints.
- In Docker, `CRAWLESS_NO_SANDBOX=1` is set by default to make Chromium work as root. Unset it by overriding env if you run with a user that can use the sandbox.

## License

MIT

