package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// countingReader compte le nombre d'octets lus via Read.
type countingReader struct {
	R io.Reader
	N int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.R.Read(p)
	if n > 0 {
		c.N += int64(n)
	}
	return n, err
}

// FetchJSONInto télécharge rawURL et décode le JSON directement dans dst (dst doit être un pointeur).
// - ctx peut être nil.
// - timeout : si <=0 on utilise DefaultTimeout.
// - maxBytes : limite de taille en octets; si <=0 on utilise DefaultMaxBytes.
// Utilise un json.Decoder sur un reader limité et détecte si le decode a nécessité
// plus de maxBytes en vérifiant le compteur.
func FetchJSONInto(ctx context.Context, rawURL string, timeout time.Duration, maxBytes int64, dst interface{}) error {
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
		return fmt.Errorf("fetch json: invalid url %q: %w", rawURL, err)
	}

	// context avec timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("fetch json: new request: %w", err)
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch json: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetch json: unexpected http status %s", resp.Status)
	}

	if resp.ContentLength > 0 && maxBytes > 0 && resp.ContentLength > maxBytes {
		return fmt.Errorf("fetch json: content-length %d exceeds limit %d", resp.ContentLength, maxBytes)
	}

	// on crée un reader qui limite et qui compte les octets lus
	limitReader := io.LimitReader(resp.Body, maxBytes+1) // +1 pour détecter dépassement
	cr := &countingReader{R: limitReader}
	dec := json.NewDecoder(cr)

	if err := dec.Decode(dst); err != nil {
		// erreur de décodage (JSON invalide, EOF inattendu, etc.)
		return fmt.Errorf("fetch json: decode: %w", err)
	}

	// si on a lu plus que maxBytes, le decode a consommé maxBytes+1 => overflow
	if cr.N > maxBytes {
		return ErrTooLarge
	}

	return nil
}

// FetchJSON générique : fetch + unmarshal dans une valeur typée.
func FetchJSON[T any](ctx context.Context, rawURL string, timeout time.Duration, maxBytes int64) (T, error) {
	var zero T
	var v T
	if err := FetchJSONInto(ctx, rawURL, timeout, maxBytes, &v); err != nil {
		return zero, err
	}
	return v, nil
}
