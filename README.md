# PortEden CLI (`porteden`)

Command-line interface for PortEden calendar firewall.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install porteden/tap/porteden
```

### Quick Install Script

Downloads pre-built binary (no Go required):

```bash
curl -sSfL https://raw.githubusercontent.com/porteden/cli/main/install.sh | bash
```

### Go Install

If you have Go installed:

```bash
go install github.com/porteden/cli/cmd/porteden@latest
```

### Build from Source

```bash
git clone https://github.com/porteden/cli.git
cd cli
make build
sudo mv porteden /usr/local/bin/
```

## Quick Start

1. **Authenticate** (choose one):
   ```bash
   # Browser OAuth (interactive)
   porteden auth login

   # Direct token (non-interactive, for CI/automation)
   porteden auth login --token pe_your_api_key
   ```

2. **List your calendars**:
   ```bash
   porteden calendar calendars
   ```

3. **View today's events**:
   ```bash
   porteden calendar events --today
   ```

4. **Search events**:
   ```bash
   porteden calendar events -q "standup" --today
   ```

## Authentication

### Direct Token (Recommended for CI/Automation)

```bash
# Store API key directly (no browser required)
porteden auth login --token pe_abc123...

### Browser-Based Login

```bash
# Login with default profile
porteden auth login
```

### Environment Variables

For CI/CD pipelines, you can also use environment variables:

```bash
export PE_API_KEY=pe_abc123...
porteden calendar events --today
```

### Check Authentication Status

```bash
porteden auth status
```

### Logout

```bash
# Revoke key and remove local credentials
porteden auth logout

```

## Calendar Commands

### List Calendars

```bash
porteden calendar calendars
```

### List Events

```bash
# Today's events
porteden calendar events --today

# Tomorrow's events
porteden calendar events --tomorrow

# This week
porteden calendar events --week

# Next N days
porteden calendar events --days 7

# Specific date range
porteden calendar events --from 2026-02-01 --to 2026-02-28

# Filter by calendar
porteden calendar events --today --calendar 12345

# Include cancelled events
porteden calendar events --today --include-cancelled
```

### Search Events

Search uses the `-q` flag on the `events` command:

```bash
# Search today's events
porteden calendar events -q "standup" --today

# Search this week
porteden calendar events -q "meeting" --week

# Search with attendee filter
porteden calendar events -q "review" --week --attendees "john@acme.com,jane@acme.com"
```

### Pagination

```bash
# Fetch all pages automatically
porteden calendar events --week --all

# Manual pagination
porteden calendar events --week --limit 100 --offset 0
porteden calendar events --week --limit 100 --offset 100
```

### Get Single Event

```bash
porteden calendar event <eventId>
```

### Create Event

```bash
porteden calendar create \
  --calendar 1 \
  --summary "Team Meeting" \
  --from "2026-02-10T10:00:00Z" \
  --to "2026-02-10T11:00:00Z" \
  --description "Weekly sync" \
  --location "Conference Room A" \
  --attendees "john@acme.com,jane@acme.com"

# All-day event
porteden calendar create \
  --calendar 1 \
  --summary "Company Offsite" \
  --from "2026-03-01T00:00:00Z" \
  --to "2026-03-02T00:00:00Z" \
  --all-day
```

### Update Event

```bash
# Update title
porteden calendar update <eventId> --summary "New Title"

# Reschedule
porteden calendar update <eventId> \
  --from "2026-02-10T14:00:00Z" \
  --to "2026-02-10T15:00:00Z"

# Update location
porteden calendar update <eventId> --location "Room B"

# Add attendees (with notification)
porteden calendar update <eventId> --add-attendees "new@example.com" --notify

# Remove attendees
porteden calendar update <eventId> --remove-attendees "old@example.com"
```

### Delete Event

```bash
# Delete (notifies attendees by default)
porteden calendar delete <eventId>

# Delete without notifying attendees
porteden calendar delete <eventId> --no-notify
```

### Respond to Invitation

```bash
porteden calendar respond <eventId> accepted
porteden calendar respond <eventId> declined
porteden calendar respond <eventId> tentative
```

### Free/Busy

```bash
# Check free/busy for today
porteden calendar freebusy --today

# This week
porteden calendar freebusy --week

# Specific calendars
porteden calendar freebusy --week --calendars 123,456

# Custom date range
porteden calendar freebusy --from 2026-02-05 --to 2026-02-12
```

### Events by Contact

```bash
# Find events with a specific person
porteden calendar by-contact "user@example.com"

# Search by name
porteden calendar by-contact --name "John Smith"

# Partial email match (e.g., everyone at a domain)
porteden calendar by-contact "@acme.com"
```

## Email Commands

### List/Search Emails

```bash
# Recent emails
porteden email messages

# Today's emails
porteden email messages --today

# This week
porteden email messages --week

# Last N days
porteden email messages --days 30

# Specific date range
porteden email messages --after 2026-02-01 --before 2026-02-07

# Filter by sender
porteden email messages --from boss@example.com

# Filter by recipient
porteden email messages --to team@example.com

# Filter by subject
porteden email messages --subject "invoice"

# Search with keyword
porteden email messages -q "project update"

# Unread only
porteden email messages --unread

# Emails with attachments
porteden email messages --has-attachment

# Filter by label
porteden email messages --label IMPORTANT

# Include full email body
porteden email messages --include-body

# Combine filters
porteden email messages --from boss@example.com --unread --today
```

### Pagination

```bash
# Fetch all pages automatically
porteden email messages --week --all

# Manual pagination (limit per page)
porteden email messages --limit 10
```

### Get Single Email

```bash
porteden email message <emailId>

# Without body content
porteden email message <emailId> --include-body=false
```

### Get Email Thread

```bash
porteden email thread <threadId>
```

### Send Email

```bash
# Basic send
porteden email send --to user@example.com --subject "Hello" --body "Hi there"

# With CC/BCC
porteden email send \
  --to user@example.com \
  --cc team@example.com \
  --bcc manager@example.com \
  --subject "Update" \
  --body "Here's the update"

# With named recipients
porteden email send --to "John Doe <john@example.com>" --subject "Hi" --body "Hello John"

# Read body from file
porteden email send --to user@example.com --subject "Report" --body-file report.html

# Plain text body
porteden email send --to user@example.com --subject "Note" --body "Plain text" --body-type text

# High importance
porteden email send --to user@example.com --subject "Urgent" --body "Please review" --importance high

# Send from specific connection
porteden email send --to user@example.com --subject "Hi" --body "Hello" --connection-id 42
```

### Reply to Email

```bash
# Reply to sender
porteden email reply <emailId> --body "Thanks for the update"

# Reply all
porteden email reply <emailId> --body "Noted, thanks everyone" --reply-all

# Reply with body from file
porteden email reply <emailId> --body-file reply.html
```

### Forward Email

```bash
# Forward to one recipient
porteden email forward <emailId> --to colleague@example.com

# Forward with a message
porteden email forward <emailId> --to colleague@example.com --body "FYI - see below"

# Forward to multiple recipients with CC
porteden email forward <emailId> --to user1@example.com --cc user2@example.com
```

### Delete Email

```bash
porteden email delete <emailId>
```

### Modify Email Properties

```bash
# Mark as read
porteden email modify <emailId> --mark-read

# Mark as unread
porteden email modify <emailId> --mark-unread

# Add labels
porteden email modify <emailId> --add-labels IMPORTANT,STARRED

# Remove labels
porteden email modify <emailId> --remove-labels INBOX

# Combine modifications
porteden email modify <emailId> --mark-read --add-labels IMPORTANT
```

## Output Formats

### Table (Default)

```bash
porteden calendar events --today
```

Human-readable formatted table with colors (auto-disabled when piped).

### JSON

```bash
porteden calendar events --today --json
# or
porteden calendar events --today -j
```

### Plain Text (TSV)

```bash
porteden calendar events --today --plain
# or
porteden calendar events --today -p
```

### Compact Mode

Filters noise, truncates long fields, and reduces output size. Ideal for AI agents and automation:

```bash
porteden calendar events --today --compact
# or
porteden calendar events --today -c

# Combined with JSON (recommended for AI agents)
porteden calendar events --today -jc
```

### Color Control

```bash
# Disable colors
porteden calendar events --today --color never

# Force colors (even in non-TTY)
porteden calendar events --today --color always

# Auto-detect (default)
porteden calendar events --today --color auto

# Or use environment variables
export NO_COLOR=1        # Disable colors
export FORCE_COLOR=1     # Force colors
```

## OpenClaw Skill

PortEden is available as an [OpenClaw](https://openclaw.com) skill for AI-optimized calendar firewall & management. **Use `-jc` flags** for AI-optimized output.

### Setup (once)

- Direct token: `porteden auth login --token pe_your_key_here`
- OpenClaw gateway: Set `skills.entries.porteden.env.PE_API_KEY` in `~/.openclaw/openclaw.json`
- Shell profile: `export PE_API_KEY=pe_your_key_here` in `~/.zshrc` or `~/.bashrc`
- Browser OAuth: `porteden auth login` (opens browser)
- Verify: `porteden auth status`

### Common Commands

- List calendars: `porteden calendar calendars -jc`
- Events today (or --tomorrow, --week): `porteden calendar events --today -jc`
- Events custom range: `porteden calendar events --from 2026-02-01 --to 2026-02-07 -jc`
- All events (auto-pagination): `porteden calendar events --week --all -jc`
- Search events: `porteden calendar events -q "meeting" --today -jc`
- Events by contact: `porteden calendar by-contact "user@example.com" -jc` (or --name "John Smith")
- Get single event: `porteden calendar event <eventId> -jc`
- Create event: `porteden calendar create --calendar <id> --summary "Meeting" --from "..." --to "..." --location "Room A" --attendees "a@b.com,c@d.com"`
- Update event: `porteden calendar update <eventId> --summary "New Title"` (also: --from, --to, --location)
- Update attendees: `porteden calendar update <eventId> --add-attendees "new@example.com"` (or --remove-attendees; add --notify to send notifications)
- Delete event: `porteden calendar delete <eventId>` (add --no-notify to skip attendee notifications)
- Respond to invite: `porteden calendar respond <eventId> accepted` (or: declined, tentative)
- Free/busy: `porteden calendar freebusy --today -jc`

### Email Commands

- List emails: `porteden email messages -jc`
- Today's emails: `porteden email messages --today -jc`
- Unread emails: `porteden email messages --unread -jc`
- Search emails: `porteden email messages -q "keyword" -jc`
- Filter by sender: `porteden email messages --from boss@example.com -jc`
- Filter by subject: `porteden email messages --subject "invoice" -jc`
- With attachments: `porteden email messages --has-attachment -jc`
- All emails (auto-pagination): `porteden email messages --week --all -jc`
- Get single email: `porteden email message <emailId> -jc`
- Get thread: `porteden email thread <threadId> -jc`
- Send email: `porteden email send --to user@example.com --subject "Hi" --body "Hello"`
- Reply: `porteden email reply <emailId> --body "Thanks"` (add `--reply-all` for reply all)
- Forward: `porteden email forward <emailId> --to colleague@example.com`
- Mark read: `porteden email modify <emailId> --mark-read`
- Add labels: `porteden email modify <emailId> --add-labels IMPORTANT`
- Delete email: `porteden email delete <emailId>`

### Notes

- `-jc` is shorthand for `--json --compact`: filters noise, truncates descriptions, limits attendees, reduces tokens.
- Use `--all` to auto-fetch all pages; check `meta.hasMore` and `meta.totalCount` in JSON output.
- Calendar pagination: `--limit 100 --offset 0`, then `--offset 100`, etc. Email pagination is token-based and handled automatically with `--all`.
- `by-contact` supports partial matching: `"@acme.com"` for email domain, `--name "Smith"` for name.
- "invalid calendar ID": Get IDs with `porteden calendar calendars -jc`.
- Email alias: `porteden mail` works as shorthand for `porteden email`.

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PE_API_KEY` | API key (overrides stored key) |
| `PE_PROFILE` | Default profile name |
| `PE_TIMEZONE` | Output timezone for display |
| `PE_FORMAT` | Default output format (`json`, `table`, `plain`) |
| `PE_API_URL` | API base URL (for development) |
| `PE_VERBOSE` | Enable verbose output (`1` or `true`) |
| `PE_COLOR` | Color mode: `auto`, `always`, `never` |
| `NO_COLOR` | Disable colors (standard) |
| `FORCE_COLOR` | Force colors even in non-TTY |
| `CI` | Allow insecure file-based credential storage |

### Flag Precedence

For most settings: **CLI flag > Environment variable > Default**

## Security

### Credential Storage

API keys are stored in `~/.config/porteden/credentials.json` with restricted file permissions (0600).

For CI/CD environments, set the `PE_API_KEY` environment variable directly:

```bash
export PE_API_KEY=pe_xxxxx
porteden calendar events --today
```

## Debugging

Enable verbose mode to see API requests and responses:

```bash
porteden --verbose calendar events --today
# or
porteden -v calendar events --today
```

**Security Note**: Authorization headers are redacted in verbose output.

## Building

### Development Build

```bash
go build -o porteden ./cmd/porteden
```

### Production Build with Version

```bash
go build -ldflags "-X github.com/porteden/cli/internal/config.Version=1.0.0" -o porteden ./cmd/porteden
```

### Cross-Compilation

```bash
# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o porteden-darwin-arm64 ./cmd/porteden

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o porteden-darwin-amd64 ./cmd/porteden

# Linux
GOOS=linux GOARCH=amd64 go build -o porteden-linux-amd64 ./cmd/porteden

# Windows
GOOS=windows GOARCH=amd64 go build -o porteden-windows-amd64.exe ./cmd/porteden
```

## Shell Completions

Generate shell completion scripts:

```bash
# Bash
porteden completion bash > /etc/bash_completion.d/porteden

# Zsh
porteden completion zsh > "${fpath[1]}/_porteden"

# Fish
porteden completion fish > ~/.config/fish/completions/porteden.fish

# PowerShell
porteden completion powershell > porteden.ps1
```

## Examples

### Daily Workflow

```bash
# Check today's schedule
porteden calendar events --today

# Search for a specific meeting
porteden calendar events -q "standup" --today

# Check free/busy before scheduling
porteden calendar freebusy --tomorrow

# Create a quick meeting
porteden calendar create \
  --calendar 1 \
  --summary "Quick sync" \
  --from "2026-02-10T14:00:00Z" \
  --to "2026-02-10T14:30:00Z"

# Respond to an invitation
porteden calendar respond 12345 accepted
```

### CI/CD Integration

```bash
#!/bin/bash
# Set API key from secret
export PE_API_KEY="${PORTEDEN_API_KEY}"

# Get events for the next 7 days (JSON output)
porteden calendar events --days 7 --json > events.json

# Process the events
jq '.data[] | select(.summary | contains("deploy"))' events.json
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- Issues: https://github.com/porteden/cli/issues
- Documentation: https://docs.porteden.com
- Email: support@porteden.com
