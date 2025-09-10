package utils

import (
    "bufio"
    "os"
    "path/filepath"
    "strings"
)

func ReadLines(path string) ([]string, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    out := []string{}
    s := bufio.NewScanner(f)
    for s.Scan() {
        line := strings.TrimSpace(s.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        out = append(out, line)
    }
    return out, s.Err()
}

func EnsureDirOf(path string) error {
    dir := filepath.Dir(path)
    if dir == "." || dir == "" {
        return nil
    }
    return os.MkdirAll(dir, 0755)
}

