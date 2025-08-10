## Inspector Gadget OS — End-to-End Logging & Observability Plan

This document outlines a verbose, practical, and security-conscious logging strategy for Inspector Gadget OS. It is designed for an OS-like system with a web UI, integrated Go server (Gin), gadget command execution, RBAC, SafeFS, and MCP. The plan explains what to log, how to log, where logs flow, how to correlate across components, and how to operate and troubleshoot using the logs.

### Goals (in plain English)
- Provide enough detail in logs to debug complex issues quickly without leaking secrets.
- Make logs structured, searchable, and correlated from browser → API → gadget execution → filesystem.
- Standardize levels and fields so alerts and dashboards can be built reliably.
- Keep overhead low in production via sampling and correct defaults.

## 1) Log Levels and When to Use Them
- **TRACE**: Extremely chatty, step-by-step internal details. Use only temporarily in dev or under a diagnostic feature flag. Not enabled in prod by default.
- **DEBUG**: Developer diagnostics (decisions, input sizes, branch choices). Enable in dev and staging. Off in prod by default; enabled selectively via runtime config.
- **INFO**: High-level events and state changes. Always on in all environments. No noisy spam.
- **WARN**: Something unexpected but the system can continue (retries, partial failures, degraded mode).
- **ERROR**: Operation failed and needs attention; handle gracefully if possible. Include error type and context.
- **FATAL**: Process cannot continue and will exit. Emit once with full context.

Tip: If unsure, start with INFO and promote to WARN/ERROR based on user impact.

## 2) Structured Logging Standard (JSON)
All server and gadget logs should be JSON to enable machine parsing.

Required top-level fields in every log entry:
- `ts` (RFC3339 or epoch ms)
- `level` (trace|debug|info|warn|error|fatal)
- `message`
- `service` (e.g., "integrated-server", "gadget-runner", "safe-fs", "web-ui")
- `env` (dev|staging|prod)
- `version` (server/app version, e.g., 0.1.2)
- `request_id` (UUID set per incoming HTTP request)
- `trace_id` and `span_id` (if tracing enabled; see section 8)
- `user` (authenticated username or "anonymous")
- `ip` (client IP when available)
- `route` (e.g., /api/gadgets/:name/execute)
- `status` (HTTP status for API logs)
- `duration_ms` (for request/operation timing)

Sensitive fields (tokens, passwords, secrets) must never be written to logs. See Redaction (section 6).

## 3) Backend (Gin) Logging
Library choice: use a high-performance structured logger such as Zap or Zerolog. The plan assumes Zap.

Implementation steps:
1. Create a centralized `logger` package that initializes a `zap.Logger` with JSON encoder and environment-aware config (dev = pretty console + debug; prod = JSON + info).
2. Add a `request_id` middleware:
   - If header `X-Request-ID` is present, use it; else generate a UUID.
   - Store it in Gin context for all handlers.
   - Add it to response headers and to log entries.
3. Add a Gin access log middleware:
   - On request start, capture: method, route, path, query keys (not values for sensitive ones), user (if authenticated), ip, user-agent.
   - On finish, log INFO with: status, duration_ms, bytes_out, bytes_in, request_id, route params, outcome (="ok"/"error").
   - On 4xx/5xx, elevate to WARN/ERROR with error details.
4. Add structured logging inside handlers:
   - INFO on successful operations with key fields (e.g., gadget name, file path, counts). Avoid full payloads.
   - WARN on expected errors (validation failed); ERROR on unexpected exceptions.
5. Standardize error logging helper:
   - Wrap errors with type/category (validation, auth, io, process, external) for consistent analysis.
6. Include `version.Version` in logger fields at startup and in health responses.

Key routes and what to log (examples):
- `/api/auth/login` (INFO): `user`, `auth_result=success|failure`, `reason` on failure (never log passwords).
- `/api/auth/refresh` (INFO): `user`, `rotated=true`.
- `/api/gadgets` (INFO): `count`, `duration_ms`.
- `/api/gadgets/:name/info` (INFO|WARN): `gadget_name`, `found=true|false`.
- `/api/gadgets/:name/execute` (INFO|ERROR): `gadget_name`, `args_count`, `exit_code`, `success`, `duration_ms`, `exec_id`.
- `/api/fs/read` (INFO|WARN): `path`, `size`, `allowed=true|false`.
- `/api/fs/write` (INFO|WARN|ERROR): `path`, `size`, `allowed=true|false`.
- `/api/fs/list` (INFO): `path`, `count`.
- `/api/mcp/*` (INFO|WARN|ERROR): `server`, `tool`, `result=ok|error`, `duration_ms`.

Performance: include `duration_ms` for all handlers. For hot paths, include microsecond precision if needed.

## 4) Gadget Execution Logging
Every gadget run should have a unique `exec_id` and be fully traceable.

For each execution:
- Log an INFO "start" event: `exec_id`, `gadget_name`, `args_count`, `user`, `request_id`.
- Stream stdout/stderr lines tagged with `stream=stdout|stderr`, `line_no` (optional), `partial=true|false` for long lines.
- On completion, log an INFO event: `exec_id`, `exit_code`, `success`, `duration_ms`, `output_size`, `truncated=true|false`.
- On timeout or kill, log ERROR with `reason=timeout|killed`, `timeout_ms`.

Security:
- Redact secrets in gadget output (see Redaction). Apply line-by-line regex scrub before writing to logs or returning to clients if configured.
- Do not log full command lines with secrets; store `args_count` and optionally a scrubbed `args_preview`.

Retention:
- Store recent gadget execution metadata (not full output) as INFO logs. Full output can be streamed to the client and optionally persisted to object storage if needed (see Retention).

## 5) SafeFS Logging
For filesystem operations:
- INFO: successful read/write/list: `path`, `size`, `mode`, `is_dir`, `count`, `duration_ms`.
- WARN: denied by policy: `path`, `reason`, `policy_rule`.
- ERROR: IO failures: include `error_type` and brief `error` string.
- Avoid logging file contents; only lengths and hashes (optional). Absolutely avoid secret files (apply allowlist).

## 6) Redaction, PII, and Secrets Hygiene
- Build a redaction utility with configurable patterns:
  - JWTs/Bearer tokens: `Authorization: Bearer ...` → `Authorization: [REDACTED]`
  - Passwords/keys in JSON fields: `password|token|secret|api_key` → mask values.
  - File contents: never logged; log only lengths or SHA-256 hashes if needed.
- Apply redaction to:
  - Request logs (headers/queries) before logging.
  - Gadget stdout/stderr lines before logging or storing.
  - Error strings from external tools.
- Add a "do-not-log" allowlist of paths (e.g., `/etc/shadow`, `.env`, `.ssh`) in SafeFS layer.
- Run a CI check that scans diffs for forbidden patterns (accidental `SECRET=` strings in code or tests).

## 7) Frontend (Web UI) Logging
Purpose: help debug user-visible issues and correlate with backend.

Client logging guidelines:
- Use a small logging utility wrapping `console` with levels and feature flag.
- On login success, store `user` and include it with any error report.
- Emit INFO for route changes and key actions (e.g., gadget run click) with `request_id` if available.
- Capture unhandled errors and promise rejections; send to an error endpoint (optional) or console in dev.
- Propagate correlation headers:
  - Generate `X-Request-ID` per action or reuse server-provided one when possible.
  - Support W3C traceparent if tracing is enabled.

Privacy:
- Never log tokens or passwords client-side. Mask any displayed errors.

## 8) Correlation and Distributed Tracing (Optional but Recommended)
- Adopt W3C Trace Context (`traceparent` header). Use OpenTelemetry where practical.
- Server: instrument Gin with OTel middleware to create a span per request and child spans for gadget execution, SafeFS ops, and MCP calls.
- Frontend: use an Axios interceptor to forward `traceparent` and/or `X-Request-ID` and create client spans for major actions.
- Store `trace_id` and `span_id` in logs for all components.

## 9) Log Storage, Shipping, and Retention
Linux-first storage (recommended):
- Write JSON logs to stdout and/or `/var/log/igos/*.log`.
- Use systemd/journald or `rsyslog` to collect.
- Ship to one of:
  - Loki + Grafana (lightweight, great for labels)
  - OpenSearch/ELK (Elasticsearch) + Kibana
  - Cloud alternatives (if applicable)

Retention policy:
- Hot logs (7–14 days) in primary store.
- Warm logs (30–90 days) in cheaper storage.
- Cold archives (6–12 months) in object storage (e.g., MinIO/S3) as compressed JSON or ndjson.

Rotation:
- Use `logrotate` for file-based logs with size + time policies.
- Ensure rotation signals logger to reopen file if not using stdout.

Indexing and labels:
- Index fields: `service`, `level`, `route`, `status`, `user`, `gadget_name`, `exec_id`, `request_id`, `trace_id`.

## 10) Alerts and Dashboards
Metrics to watch (derive via logs or explicit metrics):
- API error rate (5xx per route).
- Auth failures (login failures per minute) and spikes.
- Gadget failures (non-zero `exit_code`) and timeouts.
- SafeFS denials per path/role.
- MCP connect/disconnect flaps.

Alert examples:
- ERROR rate > 2% over 5 minutes on any route.
- Gadget timeout count > N over 10 minutes.
- Auth failures spike 3× baseline for 5 minutes.

Dashboards (Grafana/Kibana):
- Overview: request rate, latency percentiles, error rate, top routes.
- Gadgets: executions by name, success rate, top failing gadgets, average duration.
- FS: reads/writes over time, denials by path.
- Auth/RBAC: login successes/failures, role changes.

## 11) Implementation Plan — Step by Step (Incremental)
Phase 0 (now):
- Adopt version `0.1.2` (already set). Keep version in `/health` and logs.

Phase 1 (server foundation):
1. Add `logger` package (Zap) with env config and JSON encoder.
2. Add `request_id` middleware and standard Gin access log middleware.
3. Add redaction helpers and apply in middleware (headers/queries) and central error handling.
4. Replace `log.New` with shared structured logger throughout server packages.

Phase 2 (gadgets and SafeFS):
1. Wrap gadget execution with `exec_id` and start/finish logs, streaming stdout/stderr through a sanitizer.
2. Add SafeFS logs on read/write/list and policy denials.
3. Ensure RBAC denials log include `user`, `role(s)`, and `policy_rule`.

Phase 3 (frontend):
1. Add a small logging utility with levels and a debug flag.
2. Add Axios interceptor to include `X-Request-ID` (and `traceparent` if enabled) and to log API errors (sanitized) with the route and request_id.
3. Capture global errors and promise rejections. In dev, show toast; in prod, aggregate or sample.

Phase 4 (shipping & dashboards):
1. Configure journald/rsyslog to ship to Loki or OpenSearch.
2. Create basic dashboards and alerts.
3. Add logrotate configs for any file logs.

Phase 5 (tracing & polish):
1. Add OpenTelemetry to Gin and Axios, propagate trace context.
2. Bind `trace_id` to logs.
3. Tune sampling (e.g., 1–5% in prod).

## 12) Configuration and Feature Flags
- `LOG_LEVEL` (trace|debug|info|warn|error) default `info`.
- `LOG_FORMAT` (json|console) default `json`.
- `LOG_SAMPLING` (e.g., 1 per 100 debug logs) for prod.
- `LOG_REDACTION_PROFILE` (strict|standard) default `standard`.
- `ENABLE_TRACING` boolean, default false.
- `TRACING_EXPORTER_URL` (OTLP endpoint), optional.
- `LOG_SINK` (stdout|file|both), default stdout.
- `LOG_FILE_PATH` default `/var/log/igos/server.log`.

## 13) Example Log Entries (JSON)
API request (success):
```
{ "ts":"2025-08-10T12:00:01Z", "level":"info", "service":"integrated-server", "env":"prod", "version":"0.1.2", "request_id":"f2c5...", "route":"/api/gadgets/network/execute", "user":"admin", "status":200, "duration_ms":182, "gadget_name":"network", "args_count":2, "message":"gadget executed" }
```

API request (denied):
```
{ "ts":"2025-08-10T12:03:22Z", "level":"warn", "service":"integrated-server", "route":"/api/fs/read", "user":"readonly", "status":403, "path":"/etc/shadow", "reason":"policy_denied", "policy_rule":"filesystem:read:restricted", "message":"access denied" }
```

Gadget execution (timeout):
```
{ "ts":"2025-08-10T12:05:11Z", "level":"error", "service":"gadget-runner", "exec_id":"8a7b...", "gadget_name":"scan", "user":"user", "duration_ms":30050, "exit_code":-1, "reason":"timeout", "message":"gadget execution failed" }
```

Frontend action:
```
{ "ts":"2025-08-10T12:00:00Z", "level":"info", "service":"web-ui", "env":"prod", "version":"0.1.2", "route":"/gadgets", "action":"execute_click", "request_id":"f2c5...", "user":"admin", "message":"user triggered gadget execution" }
```

## 14) Security and Compliance Checklist
- Do not log secrets, tokens, passwords, or file contents.
- Mask known secret keys in headers and bodies.
- Respect allowlists and deny sensitive paths.
- Provide a runtime "panic button" to raise log level and enable TRACE for N minutes with auto-revert.
- Log access to admin-only routes and RBAC changes as INFO with actor and target.

## 15) Testing the Logging
- Unit tests for redaction functions with common patterns.
- Integration tests: simulate 401/403/500 and confirm logs contain required fields and no secrets.
- Load tests: verify sampling works and logging cost is acceptable.
- Chaos tests: timeouts and process failures produce expected ERROR logs.

## 16) Operations: Runbooks
- Incident: spike in gadget failures
  - Filter logs: `service=integrated-server level=error route=/api/gadgets/*`.
  - Group by `gadget_name` and `exit_code`.
  - Drill into `exec_id` to see stderr lines.
- Incident: auth failures
  - Query: `route=/api/auth/login level=info auth_result=failure`.
  - Inspect `reason` and client IPs.
- Incident: filesystem denials
  - Query: `route=/api/fs/* level=warn reason=policy_denied`.

## 17) Rollout Notes
- Start in dev with DEBUG level to validate fields.
- In prod, use INFO with sampling for DEBUG/TRACE.
- After enabling new logging features, bump version to the next patch (e.g., 0.1.3) and the footer will reflect it via `/health.version`.

---
If you want, I can implement Phase 1 (server logging middleware + request IDs + redaction) next and bump the version.


