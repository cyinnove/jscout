package lib

import (
    "bytes"
    "encoding/csv"
    "strings"
    "testing"

    "github.com/cyinnove/jscout/pkg/model"
)

func TestWriteOutputTXTUnique(t *testing.T) {
    recs := []*model.JSRecord{{JSURL: "https://a/x.js"}, {JSURL: "https://a/x.js"}, {JSURL: "https://a/y.js"}}
    var buf bytes.Buffer
    if err := WriteOutput(&buf, "txt", true, recs); err != nil {
        t.Fatalf("write txt: %v", err)
    }
    lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
    if len(lines) != 2 {
        t.Fatalf("expected 2 lines after dedupe, got %d", len(lines))
    }
}

func TestWriteOutputJSONL(t *testing.T) {
    recs := []*model.JSRecord{{JSURL: "https://a/x.js", SourcePage: "https://a/"}}
    var buf bytes.Buffer
    if err := WriteOutput(&buf, "jsonl", false, recs); err != nil {
        t.Fatalf("write jsonl: %v", err)
    }
    s := buf.String()
    if !strings.Contains(s, "\"js_url\"") || !strings.Contains(s, "x.js") {
        t.Fatalf("unexpected jsonl: %s", s)
    }
}

func TestWriteOutputCSV(t *testing.T) {
    recs := []*model.JSRecord{{JSURL: "https://a/x.js", SourcePage: "https://a/"}}
    var buf bytes.Buffer
    if err := WriteOutput(&buf, "csv", false, recs); err != nil {
        t.Fatalf("write csv: %v", err)
    }
    r := csv.NewReader(strings.NewReader(buf.String()))
    rows, err := r.ReadAll()
    if err != nil {
        t.Fatalf("parse csv: %v", err)
    }
    if len(rows) != 2 {
        t.Fatalf("expected header + 1 row, got %d", len(rows))
    }
}

