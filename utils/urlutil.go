package utils

import (
    "fmt"
    "net/url"
    "strings"
)

func NormalizeSeed(s, defaultScheme string) (string, error) {
    s = strings.TrimSpace(s)
    if s == "" {
        return "", fmt.Errorf("empty seed")
    }
    if !strings.Contains(s, "://") {
        s = defaultScheme + "://" + s
    }
    u, err := url.Parse(s)
    if err != nil {
        return "", err
    }
    if u.Scheme == "" || u.Host == "" {
        return "", fmt.Errorf("invalid URL: %s", s)
    }
    return u.String(), nil
}

func HostInScope(u *url.URL, allowed []string) bool {
    if u == nil || u.Host == "" {
        return false
    }
    h := strings.ToLower(u.Host)
    for _, a := range allowed {
        a = strings.ToLower(strings.TrimSpace(a))
        if a == "" {
            continue
        }
        if h == a || strings.HasSuffix(h, "."+a) {
            return true
        }
    }
    return false
}

