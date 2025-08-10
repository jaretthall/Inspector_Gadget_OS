# Logging and Monitoring Guide

Inspector Gadget OS includes comprehensive structured logging and monitoring capabilities built on Zap for performance and security observability.

## 📊 Logging Architecture

### Structured Logging with Zap
- **High Performance**: Zap provides minimal allocation structured logging
- **Request Tracing**: Each request gets unique X-Request-ID for tracing
- **Security Focus**: Sensitive data redaction and security event logging
- **Component Coverage**: Logging across all major components

### Log Structure
```json
{
  "level": "info",
  "timestamp": "2024-12-06T10:30:45.123Z",
  "caller": "auth/jwt.go:45",
  "message": "JWT token validated successfully",
  "request_id": "req_abc123def456",
  "user_id": "user_789",
  "component": "auth",
  "action": "token_validation",
  "duration_ms": 2.5,
  "success": true
}
```

## 🔍 Log Categories

### Authentication & Authorization
```go
// JWT token validation
logger.Info("JWT token validated successfully",
    zap.String("user_id", userID),
    zap.String("action", "token_validation"),
    zap.Duration("duration", time.Since(start)),
    zap.Bool("success", true))

// RBAC access control
logger.Warn("RBAC access denied",
    zap.String("user_id", userID),
    zap.String("resource", resource),
    zap.String("action", action),
    zap.String("reason", "insufficient_permissions"))
```

### File System Operations
```go
// SafeFS operations
logger.Info("File operation completed",
    zap.String("operation", "read"),
    zap.String("path", redactedPath),
    zap.Int64("size_bytes", fileSize),
    zap.Duration("duration", time.Since(start)))

// Security violations
logger.Error("Path traversal attempt detected",
    zap.String("attempted_path", redactedPath),
    zap.String("user_id", userID),
    zap.String("source_ip", clientIP))
```

### Gadget Lifecycle
```go
// Gadget execution
logger.Info("Gadget execution started",
    zap.String("gadget_name", gadgetName),
    zap.Strings("args", redactedArgs),
    zap.String("user_id", userID))

logger.Info("Gadget execution completed",
    zap.String("gadget_name", gadgetName),
    zap.Duration("duration", time.Since(start)),
    zap.Bool("success", result.Success),
    zap.String("result_type", result.Type))
```

## 🛡️ Security Logging

### Sensitive Data Redaction
The logging system automatically redacts sensitive information:

```go
// Redacted fields
type RedactConfig struct {
    PasswordFields []string // "password", "secret", "token"
    PathPrefixes   []string // "/etc", "/root", "/.ssh"
    EmailDomains   []string // Personal email domains
}

// Usage
redactedPath := redact.Path("/home/user/.ssh/id_rsa") // -> "/home/user/.ssh/[REDACTED]"
redactedData := redact.JSONFields(data, []string{"password", "api_key"})
```

### Security Events
All security-relevant events are logged with appropriate detail:

```go
// Failed authentication
logger.Error("Authentication failed",
    zap.String("reason", "invalid_credentials"),
    zap.String("source_ip", clientIP),
    zap.String("user_agent", userAgent),
    zap.Int("attempt_count", attemptCount))

// Privilege escalation attempts
logger.Error("Privilege escalation attempt",
    zap.String("user_id", userID),
    zap.String("requested_role", role),
    zap.String("current_role", currentRole),
    zap.String("resource", resource))
```

## 📈 Monitoring and Metrics

### Health Endpoint
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "version": "0.1.3",
  "timestamp": "2024-12-06T10:30:45Z",
  "components": {
    "database": "healthy",
    "gadgets": "healthy",
    "filesystem": "healthy"
  },
  "uptime_seconds": 3600,
  "request_count": 1250,
  "active_users": 5
}
```

### Log Analysis

**View real-time logs:**
```bash
# Follow all logs
tail -f /var/log/inspector-gadget-os/server.log

# Filter by component
grep '"component":"auth"' /var/log/inspector-gadget-os/server.log

# Security events only
grep '"level":"error"' /var/log/inspector-gadget-os/server.log | grep -E '(auth|rbac|safefs)'

# Performance analysis
grep '"duration_ms"' /var/log/inspector-gadget-os/server.log | jq '.duration_ms'
```

**Log rotation configuration:**
```yaml
# /etc/logrotate.d/inspector-gadget-os
/var/log/inspector-gadget-os/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 gadget gadget
    postrotate
        systemctl reload inspector-gadget-os
    endscript
}
```

## 🎯 Request Tracing

### Request ID Propagation
Every request gets a unique ID that flows through all components:

```http
GET /api/gadgets HTTP/1.1
X-Request-ID: req_abc123def456

HTTP/1.1 200 OK
X-Request-ID: req_abc123def456
Content-Type: application/json
```

### Distributed Tracing
Use request IDs to trace operations across components:

```bash
# Find all operations for a specific request
grep "req_abc123def456" /var/log/inspector-gadget-os/server.log

# Example trace flow:
# 1. HTTP request received
# 2. JWT token validated
# 3. RBAC permissions checked
# 4. Gadget execution started
# 5. SafeFS file read
# 6. Gadget execution completed
# 7. HTTP response sent
```

## 📊 Performance Monitoring

### Key Metrics to Monitor

**Response Times:**
```bash
# Average response times by endpoint
grep '"duration_ms"' server.log | jq -r '"\(.path) \(.duration_ms)"' | awk '{
    path[$1] += $2; count[$1]++
} END {
    for (p in path) print p, path[p]/count[p] "ms"
}'
```

**Error Rates:**
```bash
# Error rate by component
grep '"level":"error"' server.log | jq -r '.component' | sort | uniq -c
```

**Resource Usage:**
```bash
# File operations by path
grep '"operation":"read"' server.log | jq -r '.path' | sort | uniq -c

# Most active users
grep '"user_id"' server.log | jq -r '.user_id' | sort | uniq -c | sort -nr
```

## 🚨 Alerting and Monitoring

### Log-based Alerts

**Authentication Failures:**
```bash
# Alert on 5+ failed auth attempts from same IP in 5 minutes
grep '"level":"error"' server.log | \
grep '"component":"auth"' | \
grep "$(date -d '5 minutes ago' '+%Y-%m-%dT%H:%M')" | \
jq -r '.source_ip' | sort | uniq -c | awk '$1 >= 5'
```

**Path Traversal Attempts:**
```bash
# Alert on path traversal attempts
grep '"level":"error"' server.log | grep "path_traversal" | \
jq -r '"\(.timestamp) \(.user_id) \(.attempted_path)"'
```

**Performance Degradation:**
```bash
# Alert on slow requests (>5 seconds)
grep '"duration_ms"' server.log | \
jq 'select(.duration_ms > 5000) | "\(.timestamp) \(.path) \(.duration_ms)ms"'
```

### Integration with Monitoring Systems

**Prometheus Metrics Export:**
```go
// metrics/exporter.go
func ExportLogsToPrometrics(logEntry LogEntry) {
    // Convert structured logs to Prometheus metrics
    requestDuration.WithLabelValues(logEntry.Path, logEntry.Method).Observe(logEntry.Duration)
    requestsTotal.WithLabelValues(logEntry.Path, logEntry.StatusCode).Inc()
    
    if logEntry.Level == "error" {
        errorsTotal.WithLabelValues(logEntry.Component, logEntry.ErrorType).Inc()
    }
}
```

**Grafana Dashboard Query Examples:**
```promql
# Average response time
rate(request_duration_seconds_sum[5m]) / rate(request_duration_seconds_count[5m])

# Error rate
rate(errors_total[5m]) / rate(requests_total[5m]) * 100

# Authentication failures
increase(auth_failures_total[5m])
```

## 🔧 Configuration

### Log Level Configuration
```yaml
# o-llama/configs/runtime.yaml
logging:
  level: "info"          # debug, info, warn, error
  output: "file"         # console, file, both
  file_path: "/var/log/inspector-gadget-os/server.log"
  max_size_mb: 100
  max_backups: 10
  max_age_days: 30
  compress: true
  
  redaction:
    enabled: true
    password_fields: ["password", "secret", "token", "api_key"]
    path_prefixes: ["/etc", "/root", "/.ssh", "/home/*/.ssh"]
    email_domains: ["gmail.com", "yahoo.com", "hotmail.com"]
```

### Environment Variables
```bash
# Log level override
export GADGET_LOG_LEVEL=debug

# Enable console output for development
export GADGET_LOG_OUTPUT=both

# Custom log file location
export GADGET_LOG_FILE=/custom/path/server.log
```

## 🛠️ Debugging and Troubleshooting

### Common Debug Scenarios

**Authentication Issues:**
```bash
# Enable debug logging for auth component
GADGET_LOG_LEVEL=debug ./ollama-server

# Look for auth-specific logs
grep '"component":"auth"' server.log | jq '.message, .user_id, .success'
```

**Performance Problems:**
```bash
# Find slowest operations
grep '"duration_ms"' server.log | jq 'select(.duration_ms > 1000)' | \
sort_by(.duration_ms) | reverse | .[0:10]

# Identify bottleneck components
grep '"duration_ms"' server.log | jq -r '"\(.component) \(.duration_ms)"' | \
awk '{sum[$1]+=$2; count[$1]++} END {for(c in sum) print c, sum[c]/count[c]"ms"}'
```

**Security Incidents:**
```bash
# Review security events for specific user
grep '"user_id":"suspicious_user"' server.log | grep '"level":"error"'

# Check for unusual file access patterns
grep '"component":"safefs"' server.log | jq '.path, .operation, .user_id'
```

### Development Tips

**Local Development:**
```bash
# Console logging for development
GADGET_LOG_OUTPUT=console GADGET_LOG_LEVEL=debug ./ollama-server

# Pretty-printed JSON logs
tail -f server.log | jq '.'

# Filter logs by request ID
tail -f server.log | jq 'select(.request_id=="req_abc123")'
```

**Testing Log Output:**
```go
// Test with structured logging
func TestGadgetExecution(t *testing.T) {
    // Capture logs during test
    logBuffer := &bytes.Buffer{}
    logger := zap.New(zapcore.NewCore(
        zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
        zapcore.AddSync(logBuffer),
        zapcore.InfoLevel,
    ))
    
    // Run test with logger
    gadget := NewTestGadget(logger)
    result := gadget.Execute(ctx, args)
    
    // Assert log messages
    logOutput := logBuffer.String()
    assert.Contains(t, logOutput, "execution started")
    assert.Contains(t, logOutput, "execution completed")
}
```

---

This comprehensive logging system provides the foundation for monitoring, debugging, and securing Inspector Gadget OS. The structured approach makes it easy to analyze system behavior and identify issues quickly.