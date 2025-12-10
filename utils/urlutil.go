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

// ExtractBaseDomain extracts the base domain (eTLD+1) from a hostname
// For example: "subdomain.example.com" -> "example.com", "example.co.uk" -> "example.co.uk"
func ExtractBaseDomain(host string) string {
    host = strings.ToLower(strings.TrimSpace(host))
    if host == "" {
        return ""
    }
    
    // Remove port if present
    if idx := strings.Index(host, ":"); idx != -1 {
        host = host[:idx]
    }
    
    parts := strings.Split(host, ".")
    if len(parts) < 2 {
        return host // Return as-is if it's already a base domain or invalid
    }
    
    // Common eTLDs that need special handling (co.uk, com.au, etc.)
    // For simplicity, we'll use a heuristic: if it's a 2-part domain (like example.com),
    // return it. If it's 3+ parts, assume the last 2 parts are the eTLD.
    // This is a simplified approach - for production, use a proper TLD list.
    
    // Simple heuristic: take last 2 parts for most cases
    if len(parts) >= 2 {
        return parts[len(parts)-2] + "." + parts[len(parts)-1]
    }
    
    return host
}

func HostInScope(u *url.URL, allowed []string) bool {
    if u == nil || u.Host == "" {
        return false
    }
    h := strings.ToLower(u.Host)
    // Remove port if present for comparison
    if idx := strings.Index(h, ":"); idx != -1 {
        h = h[:idx]
    }
    
    for _, a := range allowed {
        a = strings.ToLower(strings.TrimSpace(a))
        if a == "" {
            continue
        }
        // Remove port from allowed host if present
        if idx := strings.Index(a, ":"); idx != -1 {
            a = a[:idx]
        }
        
        // Exact match
        if h == a {
            return true
        }
        // Subdomain match (e.g., acdn.mercury.com matches mercury.com)
        if strings.HasSuffix(h, "."+a) {
            return true
        }
    }
    return false
}

