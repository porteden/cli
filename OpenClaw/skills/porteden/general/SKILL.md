---
name: porteden
description: Secure Calendar, Email, and Google Drive management - Gmail, Outlook & Exchange, Google Docs & Sheets - list, search, create, update, delete across multiple accounts (gog-cli secure alternative).
homepage: https://porteden.com
metadata: {"openclaw":{"emoji":"🔗","requires":{"bins":["porteden"],"env":["PE_API_KEY"]},"primaryEnv":"PE_API_KEY","install":[{"id":"brew","kind":"brew","formula":"porteden/tap/porteden","bins":["porteden"],"label":"Install porteden (brew)"},{"id":"go","kind":"go","module":"github.com/porteden/cli/cmd/porteden@latest","bins":["porteden"],"label":"Install porteden (go)"}]}}
---

# porteden

Use `porteden` for calendar, email, and Google Drive management across multiple accounts. **Use `-jc` flags** for AI-optimized output.

If `porteden` is not installed: `brew install porteden/tap/porteden` (or `go install github.com/porteden/cli/cmd/porteden@latest`).

Setup (once)

- **Browser login (recommended):** `porteden auth login` — opens browser, credentials stored in system keyring
- **Direct token:** `porteden auth login --token <key>` — stored in system keyring
- **Verify:** `porteden auth status`
- If `PE_API_KEY` is set in the environment, the CLI uses it automatically (no login needed).

## Calendar commands

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

Event Status Values

- `confirmed` - Accepted/scheduled
- `tentative` - Maybe attending
- `needsAction` - Requires response from user
- `cancelled` - Event was cancelled

Time Formats

- All times use RFC3339 UTC format: `2026-02-01T10:00:00Z`
- For all-day events, use midnight-to-midnight with `--all-day` flag
- JSON output includes `startUtc`, `endUtc`, `durationMinutes` fields

## Email commands

Use `porteden email` (alias: `porteden mail`) for email management.

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

## Drive commands

Use `porteden drive` for Google Drive file and folder management.
Drive access requires a token with `driveAccessEnabled: true` and a connected Google account with Drive scopes.

- List/search files: `porteden drive files -jc`
- Search: `porteden drive files -q "budget" -jc` (also: --folder, --mime-type, --name, --shared-with-me)
- All files (auto-paginate): `porteden drive files --all -jc`
- Get file metadata: `porteden drive file google:FILEID -jc`
- Get download/export links: `porteden drive download google:FILEID -jc`
- List permissions: `porteden drive permissions google:FILEID -jc`
- Upload: `porteden drive upload --file ./report.pdf --name "Q1 Report.pdf"`
- Create folder: `porteden drive mkdir --name "Project Files"`
- Rename: `porteden drive rename google:FILEID --name "New Name"`
- Move: `porteden drive move google:FILEID --destination google:0B7_DEST_FOLDER`
- Share: `porteden drive share google:FILEID --type user --role reader --email user@example.com`
- Delete (trash): `porteden drive delete google:FILEID -y`

## Docs commands

Use `porteden docs` for Google Docs content operations and file management.

- Create: `porteden docs create --name "Meeting Notes"`
- Read: `porteden docs read google:DOCID`
- Read structured: `porteden docs read google:DOCID --format structured -j`
- Append text: `porteden docs edit google:DOCID --append "New paragraph."`
- Find and replace: `porteden docs edit google:DOCID --find "old" --replace "new"`
- Insert at position: `porteden docs edit google:DOCID --insert "Header" --at 1`
- Bulk ops from file: `porteden docs edit google:DOCID --ops-file ./ops.json`
- Export links: `porteden docs download google:DOCID -jc`
- Share, permissions, rename, delete: same flags as `porteden drive` equivalents

## Sheets commands

Use `porteden sheets` for Google Sheets data operations and file management.

- Create: `porteden sheets create --name "Q1 Budget"`
- Metadata (tabs, dimensions): `porteden sheets info google:SHEETID -jc`
- Read range: `porteden sheets read google:SHEETID --range "Sheet1!A1:C10" -jc`
- Write (JSON): `porteden sheets write google:SHEETID --range "Sheet1!A1:B2" --values '[["Name","Score"],["Alice",95]]'`
- Write (CSV file): `porteden sheets write google:SHEETID --range "Sheet1!A1" --csv-file ./data.csv`
- Append rows: `porteden sheets append google:SHEETID --range "Sheet1!A:B" --csv "Bob,87"`
- Export links: `porteden sheets download google:SHEETID -jc`
- Share, permissions, rename, delete: same flags as `porteden drive` equivalents

## Notes

- Credentials persist in the system keyring after login. No repeated auth needed.
- Set `PE_PROFILE=work` to avoid repeating `--profile`.
- `-jc` is shorthand for `--json --compact`: filters noise, truncates fields, reduces tokens.
- Use `--all` to auto-fetch all pages; check `hasMore`/`nextPageToken` (email/drive) or `meta.hasMore`/`meta.totalCount` (calendar) in JSON output.
- Calendar manual pagination: `--limit 100 --offset 0`, then `--offset 100`, etc.
- `by-contact` supports partial matching: `"@acme.com"` for email domain, `--name "Smith"` for name.
- Email and Drive file IDs are provider-prefixed (e.g., `google:abc123`). Pass them as-is.
- `porteden drive download` returns **URLs only** — no binary content is streamed.
- `accessInfo` in Drive/Docs/Sheets responses describes active token restrictions.
- Drive `delete` moves to trash (reversible). Prompts unless `-y` flag is set.
- Sheets `--csv` inline: use `\n` as row separator. `--raw` disables formula evaluation.
- `--body` and `--body-file` are mutually exclusive for email. Use `--body-type text` for plain text (default: html).
- Confirm before sending emails, sharing files, or deleting.
- "invalid calendar ID": Get IDs with `porteden calendar calendars -jc`.
- Environment variables: `PE_API_KEY`, `PE_PROFILE`, `PE_TIMEZONE`, `PE_FORMAT`, `PE_COLOR`, `PE_VERBOSE`.
