package handler

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

type staticURLResolver map[string][]string

func (r staticURLResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	values, ok := r[host]
	if !ok {
		return nil, errors.New("host not found")
	}
	addresses := make([]net.IPAddr, 0, len(values))
	for _, value := range values {
		addresses = append(addresses, net.IPAddr{IP: net.ParseIP(value)})
	}
	return addresses, nil
}

func TestValidateURLImportTargetRejectsUnsafeTargets(t *testing.T) {
	resolver := staticURLResolver{
		"localhost":       {"127.0.0.1"},
		"metadata.test":   {"169.254.169.254"},
		"private10.test":  {"10.0.0.1"},
		"private172.test": {"172.16.0.1"},
		"private192.test": {"192.168.0.1"},
		"loopback6.test":  {"::1"},
		"ula6.test":       {"fc00::1"},
		"linklocal6.test": {"fe80::1"},
	}
	tests := []string{
		"http://127.0.0.1/",
		"http://localhost/",
		"http://metadata.test/",
		"http://private10.test/",
		"http://private172.test/",
		"http://private192.test/",
		"http://[::1]/",
		"http://loopback6.test/",
		"http://ula6.test/",
		"http://linklocal6.test/",
		"ftp://public.test/file",
		"http://user:pass@public.test/",
	}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := validateURLImportTarget(context.Background(), rawURL, resolver); err == nil {
				t.Fatalf("expected %q to be rejected", rawURL)
			}
		})
	}
}

func TestValidateURLImportTargetAllowsPublicHost(t *testing.T) {
	resolver := staticURLResolver{"public.test": {"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"}}
	if _, err := validateURLImportTarget(context.Background(), "https://public.test/document", resolver); err != nil {
		t.Fatalf("expected public host to pass validation: %v", err)
	}
}

func TestURLImportRedirectToPrivateTargetRejected(t *testing.T) {
	resolver := staticURLResolver{
		"public.test":   {"93.184.216.34"},
		"internal.test": {"10.0.0.8"},
	}
	client := newURLImportHTTPClient(resolver, func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("dial should not be called")
	})
	first, err := http.NewRequest(http.MethodGet, "https://public.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	redirect, err := http.NewRequest(http.MethodGet, "http://internal.test/admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(redirect, []*http.Request{first}); err == nil {
		t.Fatal("expected redirect to private address to be rejected")
	}
}

func TestReadLimitedURLDocumentRejectsOversizedBody(t *testing.T) {
	body, err := readLimitedURLDocument(strings.NewReader("12345"), 5)
	if err != nil || string(body) != "12345" {
		t.Fatalf("body at limit should pass: body=%q err=%v", body, err)
	}

	if _, err := readLimitedURLDocument(strings.NewReader("123456"), 5); !errors.Is(err, errURLDocumentTooLarge) {
		t.Fatalf("expected oversized error, got %v", err)
	}
}

func TestLoadDocumentContentRejectsPrivateURLBeforeRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("secret"))
	}))
	defer server.Close()

	handler := &SDPivotDocumentHandler{}
	doc := &types.SDPivotDocument{FilePath: server.URL, FileType: ".html"}
	if _, err := handler.loadDocumentContent(context.Background(), doc); err == nil {
		t.Fatal("expected private reparse URL to be rejected")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("private URL was requested %d times", got)
	}
}

func TestAllowedURLDocumentContentTypes(t *testing.T) {
	for _, contentType := range []string{"text/html; charset=utf-8", "text/plain", "text/markdown", "application/json"} {
		if !isAllowedURLDocumentContentType(contentType, nil) {
			t.Fatalf("expected %q to be allowed", contentType)
		}
	}
	if isAllowedURLDocumentContentType("application/octet-stream", []byte{0, 1, 2}) {
		t.Fatal("expected binary content type to be rejected")
	}
}
