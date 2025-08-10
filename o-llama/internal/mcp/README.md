MCP integration is implemented across `client.go`, `manager.go`, `protocol.go`, and `transport.go`.

Highlights:
- JSON-RPC 2.0 messaging with request/response/notification helpers.
- Transports: stdio, socket (unix/tcp), in-memory (tests).
- Manager handles multi-server lifecycle, health checks, and listing tools/resources.
- Client implements initialize, list tools/resources/prompts, read resources, and call tools.

See `o-llama/cmd/integrated-server/main.go` for HTTP endpoints that expose MCP server lists and tool execution.

