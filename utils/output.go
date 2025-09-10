package utils

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"jscout/pkg/model"
)

func WriteOutput(w io.Writer, format string, unique bool, records []model.JSRecord) error {
	switch lower(format) {
	case "txt", "text":
		seen := map[string]struct{}{}
		bw := bufio.NewWriter(w)
		for _, r := range records {
			if unique {
				if _, ok := seen[r.JSURL]; ok {
					continue
				}
				seen[r.JSURL] = struct{}{}
			}
			if _, err := fmt.Fprintln(bw, r.JSURL); err != nil {
				return err
			}
		}
		return bw.Flush()
	case "jsonl", "ndjson":
		enc := json.NewEncoder(w)
		for _, r := range records {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	case "csv":
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"js_url", "source_page", "status", "mime", "from_cache"}); err != nil {
			return err
		}
		for _, r := range records {
			row := []string{r.JSURL, r.SourcePage, fmt.Sprintf("%d", r.Status), r.MIME, fmt.Sprintf("%v", r.FromCache)}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c = c + 32
		}
		b[i] = c
	}
	return string(b)
}
