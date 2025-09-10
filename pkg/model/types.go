package model

// JSRecord represents a discovered JavaScript resource.
type JSRecord struct {
    JSURL      string `json:"js_url"`
    SourcePage string `json:"source_page"`
    Status     int64  `json:"status"`
    MIME       string `json:"mime"`
    FromCache  bool   `json:"from_cache"`
}

