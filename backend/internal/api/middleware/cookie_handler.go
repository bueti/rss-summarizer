package middleware

import (
	"context"
	"net/http"
)

type cookieContextKey string

const responseWriterKey cookieContextKey = "response_writer"

// InjectResponseWriter middleware makes http.ResponseWriter available in context
func InjectResponseWriter() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), responseWriterKey, w)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetResponseWriter extracts http.ResponseWriter from context
func GetResponseWriter(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(responseWriterKey).(http.ResponseWriter)
	return w, ok
}
