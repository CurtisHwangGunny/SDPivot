package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxURLDocumentSize = int64(50 << 20)
	maxURLRedirects    = 5
)

var (
	errURLDocumentTooLarge      = errors.New("URL document exceeds size limit")
	errURLDocumentContentType   = errors.New("unsupported URL document content type")
	errURLDocumentInvalidTarget = errors.New("invalid URL target")
)

type urlIPResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type urlDialContext func(context.Context, string, string) (net.Conn, error)

func validateURLImportTarget(ctx context.Context, rawURL string, resolver urlIPResolver) (*url.URL, error) {
	target, err := url.Parse(rawURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, errURLDocumentInvalidTarget
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, errURLDocumentInvalidTarget
	}
	if target.User != nil || target.Hostname() == "" {
		return nil, errURLDocumentInvalidTarget
	}

	host := strings.TrimSuffix(strings.ToLower(target.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, errURLDocumentInvalidTarget
	}
	if err := validateURLImportHost(ctx, host, resolver); err != nil {
		return nil, err
	}
	return target, nil
}

func validateURLImportHost(ctx context.Context, host string, resolver urlIPResolver) error {
	if ip := net.ParseIP(host); ip != nil {
		if !isAllowedURLImportIP(ip) {
			return errURLDocumentInvalidTarget
		}
		return nil
	}

	addresses, err := resolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return errURLDocumentInvalidTarget
	}
	for _, address := range addresses {
		if !isAllowedURLImportIP(address.IP) {
			return errURLDocumentInvalidTarget
		}
	}
	return nil
}

func isAllowedURLImportIP(ip net.IP) bool {
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsLoopback() && !ip.IsPrivate() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}

func newURLImportHTTPClient(resolver urlIPResolver, dial urlDialContext) *http.Client {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           makeURLImportDialContext(resolver, dial),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxURLRedirects {
				return errors.New("too many redirects")
			}
			_, err := validateURLImportTarget(req.Context(), req.URL.String(), resolver)
			return err
		},
	}
}

func makeURLImportDialContext(resolver urlIPResolver, dial urlDialContext) urlDialContext {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errURLDocumentInvalidTarget
		}

		var addresses []net.IPAddr
		if ip := net.ParseIP(host); ip != nil {
			addresses = []net.IPAddr{{IP: ip}}
		} else {
			addresses, err = resolver.LookupIPAddr(ctx, host)
			if err != nil || len(addresses) == 0 {
				return nil, errURLDocumentInvalidTarget
			}
		}

		for _, candidate := range addresses {
			if !isAllowedURLImportIP(candidate.IP) {
				return nil, errURLDocumentInvalidTarget
			}
		}
		var lastErr error
		for _, candidate := range addresses {
			conn, dialErr := dial(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		return nil, lastErr
	}
}

var fetchSDPivotURLDocument = fetchURLDocument

func fetchURLDocument(ctx context.Context, rawURL string) ([]byte, *url.URL, int, error) {
	resolver := net.DefaultResolver
	target, err := validateURLImportTarget(ctx, rawURL, resolver)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	client := newURLImportHTTPClient(resolver, dialer.DialContext)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, http.StatusBadRequest, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, http.StatusBadRequest, fmt.Errorf("unexpected response status")
	}
	if resp.ContentLength > maxURLDocumentSize {
		return nil, nil, http.StatusRequestEntityTooLarge, errURLDocumentTooLarge
	}

	body, err := readLimitedURLDocument(resp.Body, maxURLDocumentSize)
	if err != nil {
		if errors.Is(err, errURLDocumentTooLarge) {
			return nil, nil, http.StatusRequestEntityTooLarge, err
		}
		return nil, nil, http.StatusBadRequest, err
	}
	if !isAllowedURLDocumentContentType(resp.Header.Get("Content-Type"), body) {
		return nil, nil, http.StatusUnsupportedMediaType, errURLDocumentContentType
	}
	return body, resp.Request.URL, http.StatusOK, nil
}

func readLimitedURLDocument(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errURLDocumentTooLarge
	}
	return body, nil
}

func isAllowedURLDocumentContentType(header string, body []byte) bool {
	contentType := header
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json", "application/ld+json", "application/xml", "application/xhtml+xml", "application/rss+xml", "application/atom+xml":
		return true
	default:
		return false
	}
}
