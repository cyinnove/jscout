package lib

import (
    "io"

    "github.com/cyinnove/crawless/pkg/model"
    "github.com/cyinnove/crawless/utils"
)

// WriteOutput writes records using the same formats as the CLI (txt|jsonl|csv).
func WriteOutput(w io.Writer, format string, unique bool, records []*model.JSRecord) error {
    return utils.WriteOutput(w, format, unique, records)
}

