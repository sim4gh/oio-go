# OIO CLI (Go)

A fast, single-binary CLI tool for ephemeral content management. This is a Go port of the original Node.js CLI with significantly faster startup time.

## Features

- **Fast startup**: ~20ms vs ~300ms (15x faster than Node.js version)
- **Single binary**: No runtime dependencies
- **Cross-platform**: macOS, Linux, Windows
- **OAuth 2.0 Device Flow**: Secure authentication
- **Automatic token refresh**: Seamless authentication management
- **Multiple content types**: Text, files, screenshots (macOS)
- **TTL-based expiration**: Automatic content deletion
- **Pro features**: Sharing capabilities (Pro subscription)

## Installation

### Build from source

```bash
# Clone the repository
git clone https://github.com/sim4gh/oio-go.git
cd oio-go

# Build
make build

# Install to /usr/local/bin
make install
```

### Download binary

Download the latest release from the releases page and add to your PATH.

## Quick Start

```bash
# Login
oio auth login

# Add text content
oio a "Hello, World!"

# Add from clipboard
oio a    # or: oio c

# Take screenshot (macOS)
oio a sc  # or: oio sc

# Add file
oio a document.pdf

# List items
oio ls

# Get item
oio g <id>

# Delete item
oio d <id>
```

## Commands

### Authentication

```bash
oio auth login     # Login using device flow
oio auth logout    # Clear credentials
oio auth whoami    # Show current user
```

### Content Management

```bash
# Add content
oio a [input]              # Add from clipboard/file/text
oio a sc                   # Screenshot (macOS)
oio a document.pdf         # File upload
oio a "Hello"              # Text content
oio a --permanent          # No expiration
oio a --ttl 7d             # Custom TTL

# Get content
oio g <id>                 # Download/display item
oio g <id> --url           # Get URL only
oio g <id> --copy          # Copy URL to clipboard
oio g <id> -o ~/Downloads  # Save to directory

# List content
oio ls                     # List all items
oio ls --type text         # Filter by type
oio ls --search "query"    # Search items
oio ls --sort size         # Sort by size
oio ls --raw               # JSON output

# Delete content
oio d <id>                 # Delete with confirmation
oio d <id> --force         # Delete without confirmation

# Extend TTL
oio extend <id> --ttl 7d   # Extend to 7 days
oio extend <id> --permanent # Make permanent
```

### Sharing (Pro)

```bash
oio sh <id>                # Create public share
oio sh <id> --password pw  # Password-protected share
oio sh <id> --expires 7d   # Custom expiration
oio p <id>                 # Quick public share
```

### Configuration

```bash
oio config                 # Show all config
oio config get <key>       # Get specific value
oio config set <key> <val> # Set value
oio config path            # Show config file path
oio config reset           # Clear all config
```

### Other

```bash
oio health                 # Check API health
oio --version              # Show version
oio --help                 # Show help
```

## Shortcuts

| Shortcut | Full Command |
|----------|--------------|
| `oio c` | `oio a` (clipboard) |
| `oio sc` | `oio a sc` (screenshot) |
| `oio p <id>` | `oio sh <id> --public` |

## Configuration

Configuration is stored in:
- macOS: `~/Library/Application Support/oio/config.json`
- Linux: `~/.config/oio/config.json`
- Windows: `%APPDATA%/oio/config.json`

## TTL Format

- `30s` - 30 seconds
- `60m` - 60 minutes
- `24h` - 24 hours (default)
- `7d` - 7 days

Maximum TTL: 365 days (1 year)

## Development

```bash
# Build
make build

# Build for all platforms
make build-all

# Run tests
make test

# Format code
make fmt

# Development build with race detector
make dev
```

## Architecture

```
oio-go/
├── cmd/oio/main.go              # Entry point
├── internal/
│   ├── auth/                    # OAuth, JWT, Cognito
│   │   ├── device_flow.go       # OAuth 2.0 Device Flow
│   │   ├── token.go             # JWT handling
│   │   └── cognito.go           # Token refresh
│   ├── api/client.go            # HTTP client with auto-refresh
│   ├── cli/                     # Command implementations
│   ├── config/                  # Configuration management
│   ├── platform/                # Platform-specific code
│   ├── upload/                  # Multipart upload
│   └── util/                    # Utilities
├── go.mod
├── Makefile
└── README.md
```

## License

ISC
