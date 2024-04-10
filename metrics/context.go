package metrics

import "context"

type ctxKey int

const (
	ctxMetrics ctxKey = iota
)

// NewContext adds Metrics to the Context.
func NewContext(parent context.Context, m Metrics) context.Context {
	return context.WithValue(parent, ctxMetrics, m)
}

// FromContext gets Logger from the Context.
func FromContext(ctx context.Context) Metrics {
	if val, ok := ctx.Value(ctxMetrics).(Metrics); ok {
		return val
	}

	return &devNullMetrics{}
}
