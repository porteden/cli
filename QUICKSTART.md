# PortEden CLI - Quick Start Guide

## Prerequisites

- A PortEden account (sign up at [porteden.com](https://porteden.com))
- Go 1.21 or later (only if building from source)

## Installation

### Homebrew (Recommended - macOS/Linux)

```bash
brew install porteden/tap/porteden
```

### Quick Install Script

Downloads a pre-built binary:

```bash
curl -sSfL https://raw.githubusercontent.com/porteden/cli/main/install.sh | bash
```

### Go Install

```bash
go install github.com/porteden/cli/cmd/porteden@latest
```

### Build from Source

```bash
git clone https://github.com/porteden/cli.git
cd cli
make build
```

This creates the `porteden` binary in the current directory.

### Install to PATH

```bash
make install
```

This installs to `$GOPATH/bin` (typically `~/go/bin`).

## First-Time Setup

### 1. Authenticate

```bash
porteden auth login
```

This will:
- Open your browser for OAuth authentication
- Store the API key in `~/.config/porteden/credentials.json`

**Alternative**: Use a direct token (for CI/automation):

```bash
porteden auth login --token pe_your_api_key
```

### 2. Verify Authentication

```bash
porteden auth status
```

Expected output:
```
Profile: default
Authenticated as: your.email@example.com
Operator: Your Company Name
Key ID: 42
Key title: MacBook Pro
Key created: 2026-01-29
```

## Basic Usage

### List Calendars

```bash
porteden calendar calendars
```

### View Today's Events

```bash
porteden calendar events --today
```

### View This Week's Events

```bash
porteden calendar events --week
```

### Search Events

```bash
porteden calendar events -q "standup" --today
```

### View Events by Contact

```bash
porteden calendar by-contact "user@example.com"
porteden calendar by-contact --name "John"
```

### Create an Event

```bash
porteden calendar create \
  --calendar 1 \
  --summary "Team Meeting" \
  --from "2026-02-10T10:00:00Z" \
  --to "2026-02-10T11:00:00Z" \
  --description "Weekly team sync" \
  --attendees "colleague@example.com"
```

### Update an Event

```bash
porteden calendar update <eventId> --summary "New Title"
porteden calendar update <eventId> --add-attendees "new@example.com" --notify
```

### Delete an Event

```bash
porteden calendar delete <eventId>
```

### Respond to an Invitation

```bash
porteden calendar respond <eventId> accepted
porteden calendar respond <eventId> declined
porteden calendar respond <eventId> tentative
```

### Check Free/Busy

```bash
porteden calendar freebusy --today
porteden calendar freebusy --week
```

## Email

### List Recent Emails

```bash
porteden email messages
```

### View Today's Emails

```bash
porteden email messages --today
```

### Search Emails

```bash
porteden email messages -q "project update"
porteden email messages --from boss@example.com --unread
```

### Read a Single Email

```bash
porteden email message <emailId>
```

### View an Email Thread

```bash
porteden email thread <threadId>
```

### Send an Email

```bash
porteden email send --to user@example.com --subject "Hello" --body "Hi there"
```

### Reply to an Email

```bash
porteden email reply <emailId> --body "Thanks for the update"
porteden email reply <emailId> --body "Noted" --reply-all
```

### Forward an Email

```bash
porteden email forward <emailId> --to colleague@example.com
```

### Manage Emails

```bash
# Mark as read
porteden email modify <emailId> --mark-read

# Add labels
porteden email modify <emailId> --add-labels IMPORTANT

# Delete
porteden email delete <emailId>
```

## Output Formats

### Table (Default)

```bash
porteden calendar events --today
```

### JSON

```bash
porteden calendar events --today --json
# or
porteden calendar events --today -j
```

Great for piping to `jq`:

```bash
porteden calendar events --today --json | jq '.data[] | .summary'
```

### Plain Text (TSV)

```bash
porteden calendar events --today --plain
# or
porteden calendar events --today -p
```

### Compact (AI/Automation)

```bash
# JSON + compact (recommended for AI agents)
porteden calendar events --today -jc
```

## Environment Variables

### Quick API Key Setup (CI/CD)

Instead of browser login, you can set the API key directly:

```bash
export PE_API_KEY=pe_your_key_here
porteden calendar events --today
```

### Other Useful Variables

```bash
# Set default output format
export PE_FORMAT=json

# Set output timezone
export PE_TIMEZONE=America/New_York

# Enable verbose logging
export PE_VERBOSE=1

# Disable colors
export NO_COLOR=1

```

## Debugging

### Enable Verbose Mode

```bash
porteden -v calendar events --today
```

This shows:
- HTTP request details (Authorization header is REDACTED)
- Response status codes
- Rate limit headers
- Execution time

### Check API Connectivity

```bash
porteden -v auth status
```

If you see connection errors, verify:
1. No firewall blocking HTTPS connections
2. Correct API URL (check `PE_API_URL` if set)

## Common Issues

### "Not authenticated"

```bash
# Check auth status
porteden auth status

# If not authenticated, login
porteden auth login
```

### Rate Limiting

If you get rate limited (HTTP 429), the CLI will:
- Automatically retry with exponential backoff
- Respect `Retry-After` headers

Just wait a moment and try again.

## Next Steps

- Explore all calendar commands: `porteden calendar --help`
- Explore all email commands: `porteden email --help`
- Set up shell completions: `porteden completion --help`
- Check out advanced features in [README.md](README.md)

## Getting Help

```bash
# General help
porteden --help

# Command-specific help
porteden calendar events --help
porteden auth --help
```

## Testing Checklist

Once set up, verify these workflows:

- [ ] `porteden auth login` - Authentication works
- [ ] `porteden auth status` - Shows correct user info
- [ ] `porteden calendar calendars` - Lists calendars
- [ ] `porteden calendar events --today` - Shows today's events
- [ ] `porteden calendar events -q "meeting" --today` - Search works
- [ ] `porteden calendar create` - Can create events
- [ ] `porteden calendar update <eventId>` - Can update events
- [ ] `porteden calendar delete <eventId>` - Can delete events
- [ ] `porteden calendar respond <eventId> accepted` - Can respond to invites
- [ ] `porteden calendar freebusy --today` - Free/busy works
- [ ] `porteden calendar by-contact "user@example.com"` - Contact lookup works
- [ ] `porteden --json calendar events --today` - JSON output works
- [ ] `porteden -v calendar events --today` - Verbose mode works
- [ ] `porteden email messages --today` - Email listing works
- [ ] `porteden email message <emailId>` - Single email works
- [ ] `porteden email thread <threadId>` - Thread view works
- [ ] `porteden email messages --unread` - Unread filter works
- [ ] `porteden email messages -q "test"` - Email search works
- [ ] `porteden email modify <emailId> --mark-read` - Modify works
- [ ] `porteden --json email messages --today` - JSON email output works
- [ ] Color output in terminal
- [ ] No color when piped: `porteden calendar events --today | cat`

## Support

Found a bug? Have a feature request?
- Open an issue: https://github.com/porteden/cli/issues
- Email: support@porteden.com
