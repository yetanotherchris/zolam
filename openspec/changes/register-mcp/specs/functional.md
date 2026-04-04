# Functional Spec: `zolam mcp` command

## Requirements

1. `zolam mcp claude` registers the chroma-mcp MCP server at user scope in Claude Code
2. The command requires exactly one argument (the provider name)
3. For the `claude` provider, it runs `claude mcp add --scope user chroma -- uvx chroma-mcp --client-type http --host localhost --port 8000 --ssl false`
4. stdout and stderr from the provider CLI are forwarded to the terminal
5. The command returns a non-zero exit code if the provider CLI fails or is not found
6. Unsupported providers return a clear error listing supported providers
