# lnk

> A fast LinkedIn CLI for posting, reading, and messaging via LinkedIn's Voyager API.

Inspired by [bird](https://github.com/steipete/bird) for X/Twitter.

## Features

- **Posts**: Create, read, like, and comment on posts
- **Feed**: Read your LinkedIn feed
- **Profiles**: View profiles and connections
- **Search**: Search people, companies, and jobs
- **Messages**: Send and read messages
- **Agent-Friendly**: JSON output mode for AI agent integration

## Installation

```bash
go install github.com/pp/lnk/cmd/lnk@latest
```

Or build from source:

```bash
git clone https://github.com/pp/lnk.git
cd lnk
go build -o lnk ./cmd/lnk
```

## Quick Start

```bash
# Authenticate using browser cookies
lnk auth login --browser safari

# Check auth status
lnk auth status

# View your profile
lnk profile me

# Read your feed
lnk feed --limit 10

# Create a post
lnk post create "Hello LinkedIn!"
```

## Authentication

lnk supports two authentication methods:

### Browser Cookie Extraction (Recommended)

Extract cookies from your browser where you're logged into LinkedIn:

```bash
lnk auth login --browser safari   # Safari
lnk auth login --browser chrome   # Chrome
lnk auth login --browser firefox  # Firefox
```

### Username/Password

```bash
lnk auth login --email you@example.com --password "your-password"
```

## Commands

| Command | Description |
|---------|-------------|
| `lnk auth login` | Authenticate with LinkedIn |
| `lnk auth status` | Check authentication status |
| `lnk auth logout` | Clear stored credentials |
| `lnk profile me` | View your profile |
| `lnk profile get <username>` | View a profile |
| `lnk feed` | Read your feed |
| `lnk post create <text>` | Create a post |
| `lnk post get <urn>` | Read a post |
| `lnk search people <query>` | Search people |
| `lnk messages list` | List conversations |

## Agent Integration

All commands support `--json` flag for structured output:

```bash
lnk profile me --json
```

```json
{
  "success": true,
  "data": {
    "firstName": "John",
    "lastName": "Doe",
    "headline": "Software Engineer"
  }
}
```

Exit codes:
- `0`: Success
- `1`: General error
- `2`: Authentication failure

## Disclaimer

This tool uses LinkedIn's unofficial Voyager API. It is not affiliated with, authorized, maintained, sponsored, or endorsed by LinkedIn or Microsoft. Use at your own risk. This may violate LinkedIn's Terms of Service.

## License

MIT
