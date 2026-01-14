# lnk

> A fast LinkedIn CLI for posting, reading, and messaging via LinkedIn's Voyager API.

Inspired by [bird](https://github.com/steipete/bird) for X/Twitter.

## Features

- **Posts**: Create and read posts
- **Feed**: Read your LinkedIn feed
- **Profiles**: View profiles by username or URN
- **Agent-Friendly**: JSON output mode for AI agent integration
- **Cross-Platform**: Works on macOS and Linux

## Installation

### From Source

```bash
git clone https://github.com/pp/lnk.git
cd lnk
go build -o lnk ./cmd/lnk
```

### Using Go Install

```bash
go install github.com/pp/lnk/cmd/lnk@latest
```

### Homebrew (Coming Soon)

```bash
brew tap pp/tap
brew install lnk
```

## Quick Start

```bash
# 1. Authenticate (auto-detects your default browser)
lnk auth login

# 2. Check auth status
lnk auth status

# 3. View your profile
lnk profile me

# 4. Read your feed
lnk feed --limit 10

# 5. Create a post
lnk post create "Hello LinkedIn!"
```

## Authentication

lnk auto-detects your default browser and extracts LinkedIn cookies.

### Auto-Detect (Recommended)

```bash
lnk auth login
```

This will automatically detect and use your default browser (Safari, Chrome, Helium, Brave, Arc, Firefox, etc.).

### Specify Browser Manually

```bash
lnk auth login --browser safari   # macOS only
lnk auth login --browser chrome   # macOS/Linux
lnk auth login --browser helium   # macOS
lnk auth login --browser brave    # macOS/Linux
lnk auth login --browser arc      # macOS
lnk auth login --browser firefox  # macOS/Linux
```

**Note**: May require granting Full Disk Access to your terminal application in System Preferences > Privacy & Security.

### Environment Variables

```bash
# Set cookies as environment variables
export LNK_LI_AT="your-li_at-cookie"
export LNK_JSESSIONID="your-jsessionid-cookie"

# Or use combined format
export LNK_COOKIES="li_at=xxx; JSESSIONID=yyy"

# Then login
lnk auth login --env
```

### Getting Cookies Manually

1. Open LinkedIn in your browser and log in
2. Open Developer Tools (F12)
3. Go to Application > Cookies > linkedin.com
4. Copy the values of `li_at` and `JSESSIONID`

## Commands Reference

### Authentication

| Command | Description |
|---------|-------------|
| `lnk auth login --browser <name>` | Authenticate using browser cookies |
| `lnk auth login --env` | Authenticate using environment variables |
| `lnk auth status` | Check authentication status |
| `lnk auth logout` | Clear stored credentials |

### Profiles

| Command | Description |
|---------|-------------|
| `lnk profile me` | View your own profile |
| `lnk profile get <username>` | View a profile by username |
| `lnk profile get --urn <urn>` | View a profile by URN |

### Feed

| Command | Description |
|---------|-------------|
| `lnk feed` | Read your feed (default 10 items) |
| `lnk feed --limit 20` | Read more feed items |

### Posts

| Command | Description |
|---------|-------------|
| `lnk post create <text>` | Create a new post |
| `lnk post create --file post.txt` | Create post from file |
| `lnk post get <urn>` | Read a post by URN |

## Agent Integration

All commands support `--json` flag for structured output, making it easy to integrate with AI agents like Claude Code.

### JSON Output Examples

```bash
# Get profile as JSON
lnk profile me --json
```

```json
{
  "success": true,
  "data": {
    "urn": "urn:li:fsd_profile:ACoAAAA",
    "firstName": "John",
    "lastName": "Doe",
    "headline": "Software Engineer",
    "profileUrl": "https://www.linkedin.com/in/johndoe"
  }
}
```

```bash
# Read feed as JSON
lnk feed --limit 5 --json
```

### Error Format

```json
{
  "success": false,
  "error": {
    "code": "AUTH_EXPIRED",
    "message": "Session cookie expired. Run: lnk auth login"
  }
}
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication failure |

## Configuration

Credentials are stored in:
- **macOS/Linux**: `~/.config/lnk/credentials.json`

You can customize the location using the `XDG_CONFIG_HOME` environment variable.

## Supported Platforms

| Platform | Safari | Chrome | Firefox | Brave | Edge | Arc | Helium | Opera | Vivaldi |
|----------|--------|--------|---------|-------|------|-----|--------|-------|---------|
| macOS | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Linux | No | Yes | Yes | Yes | Yes | No | No | Yes | Yes |

## Troubleshooting

### "Permission denied reading Safari cookies"

Grant Full Disk Access to your terminal:
1. Open System Preferences > Privacy & Security > Full Disk Access
2. Add your terminal application (Terminal, iTerm2, etc.)

### "Chrome decryption failed"

On macOS, Chrome stores its encryption key in Keychain. Make sure:
1. Chrome is properly installed
2. You've logged into Chrome at least once

On Linux, the key is stored in GNOME Keyring or uses a default key.

### "No LinkedIn cookies found"

Make sure you:
1. Are logged into LinkedIn in the specified browser
2. The browser is closed (or database is not locked)
3. Have the correct permissions to read browser data

## Development

### Building

```bash
go build -o lnk ./cmd/lnk
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## Disclaimer

This tool uses LinkedIn's unofficial Voyager API. It is:

- **Not affiliated with** LinkedIn or Microsoft
- **Not authorized, maintained, sponsored, or endorsed** by LinkedIn
- **Use at your own risk** - may violate LinkedIn's Terms of Service

LinkedIn may temporarily or permanently ban accounts that use unofficial APIs. Use responsibly.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

- Inspired by [steipete/bird](https://github.com/steipete/bird)
- Uses LinkedIn's internal Voyager API patterns from [linkedin-api](https://github.com/tomquirk/linkedin-api)
