// Package safeurl provides URL validation and an HTTP client hardened against
// Server-Side Request Forgery (SSRF).
//
// There are two layers of defense:
//
//  1. ValidateURL performs cheap, synchronous checks on a user-supplied URL
//     (scheme is http/https, a host is present, the host literal is not an
//     obviously-internal address). This is meant for the API input layer so
//     bad requests are rejected before any work is done.
//
//  2. NewSafeClient returns an *http.Client whose dialer inspects the *resolved*
//     IP address of every connection (including those created while following
//     redirects) and refuses to connect to loopback, private, link-local and
//     other non-public ranges. Because the check happens at dial time, after
//     DNS resolution, it is the authoritative defense: it closes the
//     DNS-rebinding (TOCTOU) hole that a name- or parse-time-only check leaves
//     open, and it cannot be bypassed by HTTP redirects.
package safeurl

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// maxRedirects is the maximum number of redirects the safe client will follow.
const maxRedirects = 5

// ValidateURL performs cheap, synchronous validation of a user-supplied URL.
//
// It guarantees the URL is well-formed, uses an http or https scheme and has a
// host. If the host is an IP literal it is checked against the disallowed
// ranges immediately. Hostnames that resolve to internal addresses are caught
// later, at dial time, by the client returned from NewSafeClient.
func ValidateURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL must include a host")
	}

	// Reject well-known internal hostnames up front for a friendlier, faster
	// failure. The dial-time check below is what actually enforces this.
	if isInternalHostname(host) {
		return fmt.Errorf("URL host %q is not allowed", host)
	}

	// If the host is an IP literal, reject disallowed ranges immediately.
	if ip := net.ParseIP(host); ip != nil && isDisallowedIP(ip) {
		return fmt.Errorf("URL host %q resolves to a disallowed address", host)
	}

	return nil
}

// NewSafeClient returns an *http.Client that is hardened against SSRF. The
// supplied timeout bounds the whole request (including redirects); it should be
// greater than zero.
func NewSafeClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		// Control runs after DNS resolution but before the socket is dialed,
		// on the concrete IP being connected to. This is what makes the
		// protection robust against DNS rebinding and redirect-based bypasses.
		Control: safeControl,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Timeout:       timeout,
		Transport:     transport,
		CheckRedirect: checkRedirect,
	}
}

// checkRedirect limits the number of redirects and refuses to follow a redirect
// to anything other than http/https. The destination IP is still validated by
// safeControl when the next connection is dialed.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	switch req.URL.Scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("redirect to unsupported scheme %q blocked", req.URL.Scheme)
	}
}

// safeControl is installed as net.Dialer.Control. It receives the concrete
// resolved address (host:port, host being an IP literal) and rejects the
// connection if the IP is not publicly routable.
func safeControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid dial address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("dial address host %q is not an IP", host)
	}
	if isDisallowedIP(ip) {
		return fmt.Errorf("connection to disallowed address %s blocked", ip)
	}
	return nil
}

// isInternalHostname catches a few common internal names cheaply. It is a
// convenience, not a security boundary; the dial-time IP check is authoritative.
func isInternalHostname(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	switch h {
	case "localhost", "ip6-localhost", "ip6-loopback":
		return true
	}
	return strings.HasSuffix(h, ".localhost")
}

// isDisallowedIP reports whether ip is in a range that must never be reached by
// a user-controlled request (loopback, private, link-local, CGNAT, multicast,
// unspecified, etc.).
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// Normalize IPv4-in-IPv6 (e.g. ::ffff:127.0.0.1) to its IPv4 form so the
	// checks below apply correctly.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	if ip.IsUnspecified() || // 0.0.0.0, ::
		ip.IsLoopback() || // 127.0.0.0/8, ::1
		ip.IsPrivate() || // 10/8, 172.16/12, 192.168/16, fc00::/7
		ip.IsLinkLocalUnicast() || // 169.254.0.0/16 (incl. 169.254.169.254), fe80::/10
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() {
		return true
	}

	if v4 := ip.To4(); v4 != nil {
		// Carrier-grade NAT 100.64.0.0/10 (RFC 6598) — not publicly routable.
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
		// Limited broadcast 255.255.255.255.
		if v4[0] == 255 && v4[1] == 255 && v4[2] == 255 && v4[3] == 255 {
			return true
		}
	}

	return false
}
