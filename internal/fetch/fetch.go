// Package fetch fournit des utilitaires légers et testables pour télécharger
// des ressources HTTP.
// Pour le téléchargement de binaires (yt-dlp) on ajoutera une
//
//	fonction FetchToFile/FetchToFileWithProgress qui écrit en streaming
//	dans un tmp puis os.Rename (ou réutilise WriteStreamAtomic si tu l’ajoutes).
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultTimeout   = 15 * time.Second
	DefaultMaxBytes  = 10_000_000
	DefaultUserAgent = "SubScribe/1.0"
)

// Erreurs exportées
var (
	ErrStatus   = errors.New("unexpected HTTP status")
	ErrTooLarge = errors.New("response body too large")
)

// FetchBytesWithTimeout télécharge l'URL et retourne les octets.
// - ctx peut être nil.
// - timeout : si <=0 on utilise DefaultTimeout.
// - maxBytes : si <=0 on utilise DefaultMaxBytes.
// Note : cette fonction lit tout en mémoire (OK pour JSON youtube).
func FetchBytesWithTimeout(ctx context.Context, rawURL string, timeout time.Duration, maxBytes int64) ([]byte, error) {
	// defaults
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}

	// valider l'URL tôt
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return nil, fmt.Errorf("fetch: invalid url %q: %w", rawURL, err)
	}

	// timeout via context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch: new request: %w", err)
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	client := &http.Client{} // pour tests on pourra passer un client en paramètre si besoin
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch: unexpected http status %s", resp.Status)
	}

	// si Content-Length connu et supérieur à maxBytes -> échouer vite
	if resp.ContentLength > 0 && resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("fetch: content-length %d exceeds limit %d", resp.ContentLength, maxBytes)
	}

	r := io.LimitReader(resp.Body, maxBytes+1) // +1 pour détecter dépassement
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("fetch: read body: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("fetch: body too large (>%d bytes)", maxBytes)
	}
	return data, nil
}
