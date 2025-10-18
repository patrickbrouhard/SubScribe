package yt

import "context"

// Interface est l'abstraction utilisée par l'application. Elle facilite le test
// en autorisant une implémentation factice dans les tests.
type Interface interface {
	CheckBinary() error
	GetVersion(ctx context.Context) (string, error)
	ExtractRaw(ctx context.Context, url string) (*ExtractedRaw, error)
}
