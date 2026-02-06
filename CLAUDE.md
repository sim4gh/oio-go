# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OIO CLI (Go) is a fast, single-binary command-line tool for ephemeral content management. This is a Go port of the original Node.js CLI (`oio-cli/`) with significantly faster startup time (~25x).

Features:
- OAuth 2.0 Device Flow authentication
- Unified content management (text, files, screenshots)
- TTL-based auto-deletion (default: 24h)
- Sharing capabilities (Pro subscription)
- Cross-platform support (macOS, Linux, Windows)

## Development Commands

### Build
```bash
make build          # Build for current platform
make build-all      # Build for all platforms (macOS, Linux, Windows)
go build -o oio ./cmd/oio  # Direct Go build
```

### Test Locally
```bash
# Link for local testing
ln -sf "$(pwd)/oio" ~/bin/oio-go

# Test commands
./oio --version
./oio health
./oio auth login
./oio a "Hello"
./oio ls
```

### Install
```bash
make install        # Install to /usr/local/bin (requires sudo)
```

### Clean
```bash
make clean          # Remove build artifacts
```

## Architecture

### Directory Structure
```
oio-go/
├── cmd/oio/main.go              # Entry point
├── internal/
│   ├── api/client.go            # HTTP client with auto-refresh
│   ├── auth/
│   │   ├── cognito.go           # Token refresh via Cognito
│   │   ├── device_flow.go       # OAuth 2.0 Device Flow
│   │   └── token.go             # JWT decode, expiration check
│   ├── cli/                     # Command implementations (Cobra)
│   │   ├── root.go              # Root command setup
│   │   ├── auth.go              # login, logout, whoami
│   │   ├── add.go               # add command (text, file, screenshot)
│   │   ├── get.go               # get/download command
│   │   ├── list.go              # list command
│   │   ├── delete.go            # delete command
│   │   ├── extend.go            # extend TTL command
│   │   ├── share.go             # share command (Pro)
│   │   ├── config.go            # config management
│   │   ├── health.go            # health check
│   │   └── shortcuts.go         # c, sc, p aliases
│   ├── config/
│   │   ├── config.go            # JSON config management
│   │   └── paths.go             # Platform-specific paths
│   ├── platform/                # Platform-specific code
│   │   ├── clipboard.go         # Clipboard detection
│   │   ├── clipboard_darwin.go  # macOS clipboard (pngpaste)
│   │   ├── clipboard_other.go   # Stub for other platforms
│   │   ├── screenshot_darwin.go # macOS screenshot (screencapture)
│   │   └── screenshot_other.go  # Stub for other platforms
│   ├── upload/multipart.go      # S3 multipart upload
│   └── util/
│       ├── format.go            # Byte formatting, progress bars
│       └── ttl.go               # TTL parsing and formatting
├── go.mod
├── Makefile
└── README.md
```

### Key Libraries
| Library | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/fatih/color` | Colored output |
| `github.com/briandowns/spinner` | Loading spinners |
| `github.com/olekukonko/tablewriter` | Table rendering |
| `github.com/atotto/clipboard` | Cross-platform clipboard |
| `github.com/pkg/browser` | Open URLs in browser |

### Hardcoded Values
```go
// internal/api/client.go
const DefaultBaseURL = "https://auth.yumaverse.com"

// internal/auth/cognito.go
const CognitoDomain = "oio-70676d07.auth.us-west-2.amazoncognito.com"
const ClientID = "5s958v222hp10p0qe86duks7ku"
```

### Configuration Storage
Platform-specific paths defined in `internal/config/paths.go`:
- macOS: `~/Library/Application Support/oio/config.json`
- Linux: `~/.config/oio/config.json`
- Windows: `%APPDATA%/oio/config.json`

## Command Structure

```
oio
├── auth
│   ├── login              # OAuth device flow
│   ├── logout             # Clear credentials
│   └── whoami             # Show current user
├── a [input]  (alias: add)
│   ├── oio a              # From clipboard
│   ├── oio a sc           # Screenshot (macOS)
│   ├── oio a <path>       # From file
│   └── oio a "text"       # Text content
├── g <id>  (alias: get)   # Get/download item
├── ls  (alias: list)      # List all items
├── d <id>  (alias: delete)# Delete item
├── extend <id>            # Extend TTL
├── sh <id>  (alias: share)# Share item (Pro)
├── config                 # Configuration management
├── health                 # Health check
├── c                      # Quick clipboard (alias for "oio a")
├── sc                     # Quick screenshot (alias for "oio a sc")
└── p <id>                 # Quick public share
```

## Adding New Commands

1. Create command file in `internal/cli/`:
```go
// internal/cli/example.go
package cli

import (
    "fmt"
    "github.com/sim4gh/oio-go/internal/api"
    "github.com/spf13/cobra"
)

func addExampleCommand() {
    exampleCmd := &cobra.Command{
        Use:   "example <arg>",
        Short: "Example command description",
        Args:  cobra.ExactArgs(1),
        RunE:  runExample,
    }

    exampleCmd.Flags().BoolVar(&exampleFlag, "flag", false, "Flag description")
    rootCmd.AddCommand(exampleCmd)
}

func runExample(cmd *cobra.Command, args []string) error {
    resp, err := api.Post("/endpoint", map[string]interface{}{
        "data": args[0],
    })
    if err != nil {
        return err
    }

    if resp.StatusCode != 200 {
        return fmt.Errorf("failed: %s", resp.GetString("message"))
    }

    fmt.Printf("Result: %s\n", resp.GetString("result"))
    return nil
}
```

2. Register in `internal/cli/root.go`:
```go
func init() {
    // ... existing commands
    addExampleCommand()
}
```

## API Request Pattern

All API calls use the `api` package which handles:
- Automatic token refresh when expired (60-second buffer)
- Authorization header injection
- JSON marshaling/unmarshaling
- Error handling

```go
// Authenticated requests
resp, err := api.Get("/shorts")
resp, err := api.Post("/shorts", body)
resp, err := api.Patch("/shorts/"+id, body)
resp, err := api.Delete("/shorts/"+id)

// Unauthenticated requests
resp, err := api.GetNoAuth("/health")

// Response handling
if resp.StatusCode == 200 {
    var result MyStruct
    resp.Unmarshal(&result)
}
```

## Token Management

- ID tokens are used for API authorization (not access tokens)
- Tokens are automatically refreshed in `api.Request()` before each call
- Refresh happens when token expires within 60 seconds
- JWT decoding in `internal/auth/token.go`
- Cognito refresh in `internal/auth/cognito.go`

## Platform-Specific Code

Use build tags for platform-specific implementations:

```go
//go:build darwin
// darwin-only code

//go:build !darwin
// non-darwin fallback
```

Screenshot and clipboard image features are macOS-only:
- `screencapture` command for screenshots
- `pngpaste` (brew install pngpaste) for clipboard images

## Error Handling

Commands should return errors rather than calling `os.Exit()`:
```go
func runCommand(cmd *cobra.Command, args []string) error {
    if err := doSomething(); err != nil {
        return fmt.Errorf("failed to do something: %w", err)
    }
    return nil
}
```

The root command handles displaying errors and exit codes.

## Testing

Currently no automated tests. Test manually:
```bash
./oio health                    # No auth required
./oio auth login                # Complete device flow
./oio a "test content"          # Add text
./oio ls                        # List items
./oio g <id>                    # Get item
./oio d <id> --force            # Delete item
./oio auth logout               # Clear credentials
```

## Performance

| Metric | Go | Node.js | Improvement |
|--------|-----|---------|-------------|
| Startup | ~13ms | ~327ms | 25x faster |
| Binary | 12MB | N/A | Single file |

## Release Process

1. Update version in `internal/cli/root.go`:
   ```go
   var Version = "1.0.1"
   ```

2. Build release binaries:
   ```bash
   VERSION=1.0.1 make build-all
   ```

3. Binaries are created in `build/`:
   - `oio-darwin-arm64`
   - `oio-darwin-amd64`
   - `oio-linux-amd64`
   - `oio-linux-arm64`
   - `oio-windows-amd64.exe`

4. Create GitHub release with binaries

5. Update Homebrew formula (if distributing via Homebrew)
