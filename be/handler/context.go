package handler

import "context"

// Private context key types to avoid collisions.
type ctxKeyUser struct{}
type ctxKeyModel struct{}
type ctxKeyStream struct{}

// contextWith returns a new context carrying user, model, and streaming flag.
func contextWith(ctx context.Context, user, model string, isStream bool) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUser{}, user)
	ctx = context.WithValue(ctx, ctxKeyModel{}, model)
	ctx = context.WithValue(ctx, ctxKeyStream{}, isStream)
	return ctx
}
