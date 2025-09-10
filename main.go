package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// detectChromePath checks PATH for chrome/chromium binaries
func detectChromePath() string {
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

// askUserForChromePath prompts until user enters a valid path
func askUserForChromePath() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("❓ Chrome not found in PATH. Please enter the full path to Chrome: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if stat, err := os.Stat(input); err == nil && !stat.IsDir() {
			return input
		}
		fmt.Println("⚠️ Invalid path. Try again.")
	}
}

// collectJSFiles visits a URL and returns all JS file URLs
func collectJSFiles(chromePath, url string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	if err := chromedp.Run(taskCtx, network.Enable()); err != nil {
		return nil, err
	}

	jsFiles := []string{}
	chromedp.ListenTarget(taskCtx, func(ev interface{}) {
		if ev, ok := ev.(*network.EventResponseReceived); ok {
			if ev.Type == network.ResourceTypeScript {
				jsFiles = append(jsFiles, ev.Response.URL)
			}
		}
	})

	if err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
	); err != nil {
		return nil, err
	}

	return jsFiles, nil
}

func main() {
	// Define CLI flags
	urlFlag := flag.String("u", "", "Single URL to scan")
	listFlag := flag.String("l", "", "File containing list of URLs (one per line)")
	outputFlag := flag.String("o", "collected_js.txt", "Output file to save results")
	flag.Parse()

	// Collect URLs from args
	urls := []string{}
	if *urlFlag != "" {
		urls = append(urls, *urlFlag)
	}
	if *listFlag != "" {
		file, err := os.Open(*listFlag)
		if err != nil {
			log.Fatalf("❌ Failed to open list file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				urls = append(urls, line)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("❌ Failed to read list file: %v", err)
		}
	}

	if len(urls) == 0 {
		fmt.Println("Usage: go run main.go -u <url> OR -l <file> [-o output.txt]")
		os.Exit(1)
	}

	// Detect Chrome
	chromePath := detectChromePath()
	if chromePath == "" {
		chromePath = askUserForChromePath()
	}
	fmt.Println("✔️ Using Chrome at:", chromePath)

	// Open output file
	f, err := os.Create(*outputFlag)
	if err != nil {
		log.Fatal("❌ Failed to create output file:", err)
	}
	defer f.Close()
	writer := bufio.NewWriter(f)

	// Process each URL
	for _, url := range urls {
		fmt.Println("🔍 Visiting:", url)
		jsFiles, err := collectJSFiles(chromePath, url)
		if err != nil {
			fmt.Fprintf(writer, "URL: %s\nError: %v\n\n", url, err)
			continue
		}

		for _, js := range jsFiles {
			fmt.Fprintf(writer, "%s\n", js)
		}
		fmt.Fprintln(writer)
	}

	writer.Flush()
	fmt.Println("✔️ Results saved to", *outputFlag)
}
