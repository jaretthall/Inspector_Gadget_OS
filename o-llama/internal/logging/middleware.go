package logging

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

const RequestIDKey = "request_id"

// RequestIDMiddleware injects/propagates a request ID.
func RequestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        rid := c.GetHeader("X-Request-ID")
        if rid == "" {
            rid = uuid.New().String()
        }
        c.Set(RequestIDKey, rid)
        c.Writer.Header().Set("X-Request-ID", rid)
        c.Next()
    }
}

// AccessLogMiddleware logs every request with timing and outcome.
func AccessLogMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()

        dur := time.Since(start)
        rid, _ := c.Get(RequestIDKey)

        L().Infow("http request",
            "request_id", rid,
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "route", c.FullPath(),
            "status", c.Writer.Status(),
            "duration_ms", dur.Milliseconds(),
            "client_ip", c.ClientIP(),
            "user_agent", c.Request.UserAgent(),
        )
    }
}


