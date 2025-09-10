package lib

import (
    "testing"
    "time"

    "github.com/cyinnove/jscout/pkg/model"
)

func TestDefaultOptions(t *testing.T) {
    o := DefaultOptions()
    if !o.Headless {
        t.Fatalf("expected Headless default true")
    }
    if o.MaxDepth != 1 || o.MaxPages != 100 || o.Concurrency != 4 {
        t.Fatalf("unexpected crawl defaults: depth=%d pages=%d conc=%d", o.MaxDepth, o.MaxPages, o.Concurrency)
    }
    if o.PageTimeout <= 0 || o.WaitAfterLoad < 0 {
        t.Fatalf("invalid timeouts: page=%v wait=%v", o.PageTimeout, o.WaitAfterLoad)
    }
}

func TestFilterJSInScope(t *testing.T) {
    recs := []*model.JSRecord{
        {JSURL: "https://a.example.com/app.js"},
        {JSURL: "https://static.example.com/x.js"},
        {JSURL: "https://cdn.other.com/y.js"},
        {JSURL: "not a url"},
    }
    allowed := []string{"example.com"}
    out := FilterJSInScope(recs, allowed)
    if len(out) != 2 {
        t.Fatalf("expected 2 in-scope records, got %d", len(out))
    }
    // basic sanity
    _ = time.Second
}

