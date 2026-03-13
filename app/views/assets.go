package views

import "context"

type assetPathKey struct{}

// WithAssetPath stores the asset path function in the context.
func WithAssetPath(ctx context.Context, fn func(string) string) context.Context {
	return context.WithValue(ctx, assetPathKey{}, fn)
}

// AssetPath retrieves the named asset's URL path from the context.
// Falls back to "/static/{name}" if no asset path function is configured
// (e.g. in tests without the middleware).
func AssetPath(ctx context.Context, name string) string {
	if fn, ok := ctx.Value(assetPathKey{}).(func(string) string); ok {
		return fn(name)
	}
	return "/static/" + name
}
