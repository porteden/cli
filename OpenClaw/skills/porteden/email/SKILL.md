---
name: porteden-email
description: PortEden Email CLI for email management - list, search, read, send, reply, forward, delete, and modify emails across multiple accounts.
homepage: https://porteden.com
metadata: {"openclaw":{"emoji":"📧","requires":{"bins":["porteden"]},"install":[{"id":"manual","kind":"manual","label":"Build from source","instructions":"in https://porteden.com/docs/openclaw/"}]}}
---

# porteden email

Use `porteden email` (alias: `porteden mail`) for email management across multiple accounts. **Use `-jc` flags** for AI-optimized output.

Setup (once)

- **Browser login (recommended):** `porteden auth login` (opens browser)
- Direct token: `porteden auth login --token pe_your_key_here`
- OpenClaw gateway: Set `skills.entries.porteden.env.PE_API_KEY` in `~/.openclaw/openclaw.json`
- Shell profile: `export PE_API_KEY=pe_your_key_here` in `~/.zshrc` or `~/.bashrc`
- Verify: `porteden auth status`

Common commands

- List emails (or --today, --yesterday, --week, --days N): `porteden email messages -jc`
- Filter emails: `porteden email messages --from sender@example.com -jc` (also: --to, --subject, --label, --unread, --has-attachment)
- Search emails: `porteden email messages -q "keyword" --today -jc`
- Custom date range: `porteden email messages --after 2026-02-01 --before 2026-02-07 -jc`
- All emails (auto-pagination): `porteden email messages --week --all -jc`
- Get single email: `porteden email message <emailId> -jc`
- Get thread: `porteden email thread <threadId> -jc`
- Send email: `porteden email send --to user@example.com --subject "Hi" --body "Hello"` (also: --cc, --bcc, --body-file, --body-type text, --importance high)
- Send with named recipient: `porteden email send --to "John Doe <john@example.com>" --subject "Hi" --body "Hello"`
- Reply: `porteden email reply <emailId> --body "Thanks"` (add `--reply-all` for reply all)
- Forward: `porteden email forward <emailId> --to colleague@example.com` (optional `--body "FYI"`, --cc)
- Modify email: `porteden email modify <emailId> --mark-read` (also: --mark-unread, --add-labels IMPORTANT, --remove-labels INBOX)
- Delete email: `porteden email delete <emailId>`

Notes

- Set `PE_API_KEY` in gateway config or shell profile to avoid repeated auth.
- Set `PE_PROFILE=work` to avoid repeating `--profile`.
- `-jc` is shorthand for `--json --compact`: strips attachment details, truncates body previews, limits labels, reduces tokens.
- Use `--all` to auto-fetch all pages; check `hasMore` and `nextPageToken` in JSON output.
- Email IDs are provider-prefixed (e.g., `google:abc123`, `m365:xyz789`). Pass them as-is.
- `--include-body` on `messages` fetches full body (default: preview only). Single `message` includes body by default.
- `--body` and `--body-file` are mutually exclusive. Use `--body-type text` for plain text (default: html).
- Confirm before sending, replying, forwarding, or deleting emails.
- Environment variables: `PE_API_KEY`, `PE_PROFILE`, `PE_TIMEZONE`, `PE_FORMAT`, `PE_COLOR`, `PE_VERBOSE`.
