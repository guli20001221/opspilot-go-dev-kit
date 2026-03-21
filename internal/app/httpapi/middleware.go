package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type contextKey string

const (
	requestIDKey    contextKey = "request_id"
	traceIDKey      contextKey = "trace_id"
	requestIDHeader            = "X-Request-Id"
	traceIDHeader              = "X-Trace-Id"
)

func withRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = newID()
		}

		traceID := r.Header.Get(traceIDHeader)
		if traceID == "" {
			traceID = requestID
		}

		r.Header.Set(requestIDHeader, requestID)
		r.Header.Set(traceIDHeader, traceID)
		w.Header().Set(requestIDHeader, requestID)
		w.Header().Set(traceIDHeader, traceID)

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		ctx = context.WithValue(ctx, traceIDKey, traceID)

		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))
		slog.Default().InfoContext(ctx, "http request completed",
			slog.String("request_id", requestID),
			slog.String("trace_id", traceID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

func traceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey).(string)
	return traceID
}

func requestIDFromRequest(r *http.Request) string {
	if requestID := requestIDFromContext(r.Context()); requestID != "" {
		return requestID
	}

	return r.Header.Get(requestIDHeader)
}

func traceIDFromRequest(r *http.Request) string {
	if traceID := traceIDFromContext(r.Context()); traceID != "" {
		return traceID
	}

	return r.Header.Get(traceIDHeader)
}

func newID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(buf)
}
