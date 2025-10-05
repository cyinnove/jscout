package lib

import (
    "io"

    "github.com/cyinnove/jscout/pkg/model"
    "github.com/cyinnove/jscout/utils"
)

// WriteOutput writes records using the same formats as the CLI (txt|jsonl|csv).
func WriteOutput(w io.Writer, format string, unique bool, records []*model.JSRecord) error {
    return utils.WriteOutput(w, format, unique, records)
}

