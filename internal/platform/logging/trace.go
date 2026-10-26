package logging

import (
	"context"
	"log/slog"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// TraceAttrs returns trace_id and span_id for the span stored in ctx.
// It returns nil when ctx holds no valid span.
func TraceAttrs(ctx context.Context) []slog.Attr {
	sc := oteltrace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []slog.Attr{
		slog.String("trace_id", sc.TraceID().String()),
		slog.String("span_id", sc.SpanID().String()),
	}
}

// handlerOp is one WithAttrs or WithGroup call, replayed on a handler.
type handlerOp func(slog.Handler) slog.Handler

// traceHandler adds trace_id and span_id to every record whose context holds a valid span.
//
// Trace attributes must stay at the top level of the record, even after WithGroup. A group
// swallows the attributes added to it, so when a span is present the handler is rebuilt from
// base with the trace attributes first and the recorded operations replayed on top.
type traceHandler struct {
	base  slog.Handler // The root handler, with no With calls applied.
	ops   []handlerOp  // Every WithAttrs and WithGroup call, in order.
	inner slog.Handler // base with ops applied. Used when the context holds no valid span.
}

// newTraceHandler wraps base. The returned handler adds trace attributes to traced records.
func newTraceHandler(base slog.Handler) *traceHandler {
	return &traceHandler{base: base, inner: base}
}

// Enabled reports whether the wrapped handler accepts the level.
func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle writes the record. It adds the trace attributes at the top level when ctx holds a valid span.
func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := TraceAttrs(ctx)
	if attrs == nil {
		return h.inner.Handle(ctx, r)
	}
	next := h.base.WithAttrs(attrs)
	for _, op := range h.ops {
		next = op(next)
	}
	return next.Handle(ctx, r)
}

// WithAttrs returns a handler that adds attrs to every record.
func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.with(func(x slog.Handler) slog.Handler { return x.WithAttrs(attrs) })
}

// WithGroup returns a handler that opens a group for later attributes.
func (h *traceHandler) WithGroup(name string) slog.Handler {
	return h.with(func(x slog.Handler) slog.Handler { return x.WithGroup(name) })
}

// with returns a copy of h with op appended to its operations.
func (h *traceHandler) with(op handlerOp) *traceHandler {
	ops := make([]handlerOp, 0, len(h.ops)+1)
	ops = append(ops, h.ops...)
	ops = append(ops, op)
	return &traceHandler{base: h.base, ops: ops, inner: op(h.inner)}
}
