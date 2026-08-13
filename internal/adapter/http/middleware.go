package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
)

const (
	ctxRequestID = "request_id"
	ctxLogger    = "logger"
)

// RequestID assigns every request an id and echoes it, so a customer can quote
// one and support can find the request (99 §7).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 64 {
			id = uuid.NewString()
		}
		c.Set(ctxRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Logger emits one structured line per request.
//
// It logs the route TEMPLATE, never the resolved path: a path carries ids and
// sometimes an email, and PII does not belong in logs or URLs (99 §7).
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqLog := log.With("request_id", c.GetString(ctxRequestID))
		c.Set(ctxLogger, reqLog)

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		attrs := []any{
			"method", c.Request.Method,
			"route", route,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", c.Writer.Size(),
		}
		switch {
		case c.Writer.Status() >= 500:
			reqLog.Error("request failed", attrs...)
		case c.Writer.Status() >= 400:
			reqLog.Warn("request rejected", attrs...)
		default:
			reqLog.Info("request", attrs...)
		}
	}
}

// Recovery turns a panic into the standard error model rather than a dropped
// connection, and logs the stack internally only.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					"request_id", c.GetString(ctxRequestID),
					"route", c.FullPath(),
					"panic", r)
				Fail(c, apierror.Internal(nil))
				c.Abort()
			}
		}()
		c.Next()
	}
}

// SecurityHeaders applies the header set from 99 §7. The CSP has no
// 'unsafe-inline'; the SPA and the server-rendered pages both use nonces or
// external files.
func SecurityHeaders(isProduction bool) gin.HandlerFunc {
	csp := strings.Join([]string{
		"default-src 'self'",
		// Google Maps needs its own script and image origins; nothing else is
		// allowed, and there is no 'unsafe-inline'.
		"script-src 'self' https://maps.googleapis.com https://challenges.cloudflare.com",
		"style-src 'self'",
		"img-src 'self' data: blob: https://maps.gstatic.com https://maps.googleapis.com",
		"font-src 'self'",
		"connect-src 'self' https://maps.googleapis.com",
		"frame-src https://challenges.cloudflare.com",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"object-src 'none'",
	}, "; ")

	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), payment=(), geolocation=(self)")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		if isProduction {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// CORS allows only the configured origins. An empty list in development allows
// the local Vite dev server and nothing else.
func CORS(allowed []string) gin.HandlerFunc {
	set := map[string]bool{}
	for _, o := range allowed {
		set[strings.TrimSuffix(o, "/")] = true
	}
	return func(c *gin.Context) {
		origin := strings.TrimSuffix(c.GetHeader("Origin"), "/")
		if origin != "" && set[origin] {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, Idempotency-Key")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			h.Set("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// MaxBody caps the request body. Without it, "validate every input" is still
// beaten by a client that simply sends a gigabyte before any validation runs.
func MaxBody(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

// Fail renders an error through the single error model. Driver errors, causes
// and stack traces never reach the client (99 §7).
func Fail(c *gin.Context, err error) {
	e := apierror.From(err)
	if log, ok := c.Get(ctxLogger); ok {
		if l, ok := log.(*slog.Logger); ok && e.Status >= 500 {
			l.Error("internal error", "code", e.Code, "cause", e.Unwrap())
		}
	}
	body := gin.H{"error": gin.H{"code": e.Code, "message": e.Message}}
	if len(e.Details) > 0 {
		body["error"].(gin.H)["details"] = e.Details
	}
	if id := c.GetString(ctxRequestID); id != "" {
		body["request_id"] = id
	}
	c.AbortWithStatusJSON(e.Status, body)
}

// OK renders a success payload.
func OK(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}
