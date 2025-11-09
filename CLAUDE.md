# go_llm_proxy Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-11-08

## Active Technologies

- Go 1.21+ (REQUIRED - see Constitution V: Performance [EXTRACTED FROM ALL PLAN.MD FILES] Resource Efficiency) + Gin (HTTP framework), spf13/viper (config), logrus (logging), prometheus/client_golang (metrics), gorilla/sessions (sessions) - All Go idiomatic and standard ecosystem (001-llm-proxy-go-refactor)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

# Add commands for Go 1.21+ (REQUIRED - see Constitution V: Performance [ONLY COMMANDS FOR ACTIVE TECHNOLOGIES] Resource Efficiency)

## Code Style

Go 1.21+ (REQUIRED - see Constitution V: Performance [LANGUAGE-SPECIFIC, ONLY FOR LANGUAGES IN USE] Resource Efficiency): Follow standard conventions

## Recent Changes

- 001-llm-proxy-go-refactor: Added Go 1.21+ (REQUIRED - see Constitution V: Performance [LAST 3 FEATURES AND WHAT THEY ADDED] Resource Efficiency) + Gin (HTTP framework), spf13/viper (config), logrus (logging), prometheus/client_golang (metrics), gorilla/sessions (sessions) - All Go idiomatic and standard ecosystem

## MCP (Model Context Protocol) Usage

### Available MCP Servers

The project is configured with the following MCP servers for enhanced development workflows:

- **Sequential Thinking** - Use for complex problem decomposition and step-by-step reasoning
- **Excel** - For Excel file operations (limit: 4000 cells per page)
- **context7** - Enhanced context retrieval (minimum tokens: 10000)
- **Maven** - Maven dependency management and analysis
- **AWS Knowledge Base** - AWS KB retrieval (requires AWS credentials)
- **Memory** - Persistent conversation/project memory (stored in memory.json)
- **Filesystem** - File system operations within allowed directories
- **Git** - Git repository operations and analysis
- **Playwright** - Browser automation and testing
- **Qdrant Server** - Vector database operations (embedding model: all-MiniLM-L6-v2)
- **Codegen** - Code generation from specifications

### MCP Usage Guidelines

1. **Windows Compatibility**: All npx/uvx-based MCP servers must use `cmd /c` wrapper on Windows systems
2. **Memory Management**: Use the Memory MCP server to persist important project context across sessions
3. **Sequential Thinking**: Leverage for architectural decisions, complex refactoring, and multi-step problem solving
4. **Git Operations**: Use the Git MCP for repository analysis, history exploration, and merge conflict resolution
5. **File Operations**: Use Filesystem MCP for batch file operations within allowed directories
6. **Browser Testing**: Use Playwright MCP for end-to-end testing and web scraping
7. **Code Generation**: Use Codegen MCP for generating code from natural language specifications
8. **Vector Search**: Use Qdrant MCP for semantic code search and knowledge retrieval

### Security Considerations

- **AWS Credentials**: The AWS Knowledge Base MCP requires valid AWS credentials (access key, secret key, region)
- **Environment Variables**: Some MCP servers use environment variables for configuration
- **File Access**: The Filesystem MCP is restricted to allowed directories only
- **Python Dependencies**: Codegen, Git, and Qdrant servers use uvx (Python package installer)

### Configuration

MCP servers are configured in `.mcp.json`. Modify this file to:
- Enable/disable specific servers
- Update environment variables
- Change allowed directories for Filesystem MCP
- Configure server-specific settings

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
