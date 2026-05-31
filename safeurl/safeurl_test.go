package safeurl

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateURLAccepts(t *testing.T) {
	valid := []string{
		"http://example.com",
		"https://example.com",
		"https://example.com:8443/path?q=1",
		"http://sub.domain.example.org/img.png",
		"https://93.184.216.34/", // public IP literal
	}
	for _, u := range valid {
		if err := ValidateURL(u); err != nil {
			t.Errorf("ValidateURL(%q) = %v, want nil", u, err)
		}
	}
}

func TestValidateURLRejects(t *testing.T) {
	invalid := []string{
		"",                            // empty
		"ftp://example.com",           // wrong scheme
		"file:///etc/passwd",          // wrong scheme
		"gopher://example.com",        // wrong scheme
		"http://",                     // no host
		"https:///path",               // no host
		"http://localhost",            // internal name
		"http://localhost:8080/admin", // internal name
		"http://LOCALHOST",            // case-insensitive
		"http://service.localhost",    // .localhost suffix
		"http://127.0.0.1",            // loopback literal
		"http://127.0.0.1:9000",       // loopback literal
		"http://10.0.0.5",             // private
		"http://172.16.0.1",           // private
		"http://192.168.1.1",          // private
		"http://169.254.169.254",      // cloud metadata (link-local)
		"http://100.64.1.1",           // CGNAT
		"http://0.0.0.0",              // unspecified
		"http://[::1]",                // IPv6 loopback
		"http://[fe80::1]",            // IPv6 link-local
		"http://[::ffff:127.0.0.1]",   // IPv4-mapped loopback
	}
	for _, u := range invalid {
		if err := ValidateURL(u); err == nil {
			t.Errorf("ValidateURL(%q) = nil, want error", u)
		}
	}
}

func TestIsDisallowedIP(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1":        true,
		"10.1.2.3":         true,
		"172.20.5.5":       true,
		"192.168.0.1":      true,
		"169.254.169.254":  true,
		"100.64.0.1":       true,
		"100.127.255.255":  true,
		"0.0.0.0":          true,
		"255.255.255.255":  true,
		"224.0.0.1":        true, // multicast
		"::1":              true,
		"fe80::1":          true,
		"fc00::1":          true,
		"::ffff:10.0.0.1":  true, // IPv4-mapped private
		"8.8.8.8":          false,
		"93.184.216.34":    false,
		"1.1.1.1":          false,
		"100.63.255.255":   false, // just below CGNAT
		"100.128.0.0":      false, // just above CGNAT
		"2606:4700:4700::": false, // public IPv6 (Cloudflare)
	}
	for s, want := range cases {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("could not parse test IP %q", s)
		}
		if got := isDisallowedIP(ip); got != want {
			t.Errorf("isDisallowedIP(%s) = %v, want %v", s, got, want)
		}
	}
}

func TestSafeClientBlocksLoopback(t *testing.T) {
	// httptest servers always listen on loopback, so a hardened client must
	// refuse to connect to one.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewSafeClient(5 * time.Second)
	_, err := client.Get(srv.URL)
	if err == nil {
		t.Fatalf("expected the safe client to block a loopback request to %s", srv.URL)
	}
	if !strings.Contains(err.Error(), "disallowed") {
		t.Errorf("expected a 'disallowed address' error, got: %v", err)
	}
}

func TestCheckRedirectLimit(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	via := make([]*http.Request, maxRedirects-1)
	if err := checkRedirect(req, via); err != nil {
		t.Errorf("redirect %d should be allowed, got: %v", len(via), err)
	}

	via = make([]*http.Request, maxRedirects)
	if err := checkRedirect(req, via); err == nil {
		t.Errorf("redirect %d should be blocked", len(via))
	}
}

func TestCheckRedirectScheme(t *testing.T) {
	bad, _ := http.NewRequest(http.MethodGet, "file:///etc/passwd", nil)
	if err := checkRedirect(bad, nil); err == nil {
		t.Errorf("redirect to file:// scheme should be blocked")
	}

	good, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err := checkRedirect(good, nil); err != nil {
		t.Errorf("redirect to https:// should be allowed, got: %v", err)
	}
}
