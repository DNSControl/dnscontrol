# Documentation MCP Server

- [Documentation MCP Server](#documentation-mcp-server)
  - [Endpoint](#endpoint)
  - [Features](#features)
  - [Adding to Claude Code](#adding-to-claude-code)
  - [Adding to other MCP clients](#adding-to-other-mcp-clients)
  - [Further reading](#further-reading)

The DNSControl documentation site exposes an [MCP (Model Context Protocol)](https://modelcontextprotocol.io/introduction) server. This allows AI assistants such as Claude Code, Claude Desktop, and other MCP-compatible tools to search and query the DNSControl documentation directly.

## Endpoint

The MCP server is available at:

```text
https://docs.dnscontrol.org/~gitbook/mcp
```

## Features

- **Read-only access** to all published documentation (never drafts or unpublished changes).
- **HTTP transport** only (stdio and SSE are not supported).
- **No authentication required** since the DNSControl docs are public.
- Provides a `searchDocumentation` tool that searches across all pages, code examples, API references, and guides.

## Adding to Claude Code

To connect the DNSControl docs MCP server to Claude Code, run:

```bash
claude mcp add --transport http dnscontrol-docs https://docs.dnscontrol.org/~gitbook/mcp
```

To share the configuration with your team via the project's `.mcp.json` file:

```bash
claude mcp add --transport http dnscontrol-docs --scope project https://docs.dnscontrol.org/~gitbook/mcp
```

Once configured, Claude Code gains access to a `searchDocumentation` tool that can find relevant information across the entire DNSControl documentation.

## Adding to other MCP clients

Any MCP client that supports HTTP transport can connect to the server. Add the following to your MCP configuration:

```json
{
  "mcpServers": {
    "dnscontrol-docs": {
      "type": "http",
      "url": "https://docs.dnscontrol.org/~gitbook/mcp"
    }
  }
}
```

## Further reading

- [Model Context Protocol](https://modelcontextprotocol.io/introduction)
- [Claude Code MCP Documentation](https://code.claude.com/docs/en/mcp)
