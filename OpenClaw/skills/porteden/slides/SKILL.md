---
name: porteden-slides
description: Secure Google Slides Management. Use when the user wants to read presentation content, list slides, create a new deck, or manage sharing, permissions, renames, and deletes on Google Slides.
homepage: https://porteden.com
metadata: {"openclaw":{"emoji":"📽️","requires":{"bins":["porteden"],"env":["PE_API_KEY"]},"primaryEnv":"PE_API_KEY","install":[{"id":"brew","kind":"brew","formula":"porteden/tap/porteden","bins":["porteden"],"label":"Install porteden (brew)"},{"id":"go","kind":"go","module":"github.com/porteden/cli/cmd/porteden@latest","bins":["porteden"],"label":"Install porteden (go)"}]}}
---

# porteden slides

Use `porteden slides` for Google Slides content operations and file management. **Use `-jc` flags** for AI-optimized output.

If `porteden` is not installed: `brew install porteden/tap/porteden` (or `go install github.com/porteden/cli/cmd/porteden@latest`).

## Setup (once)

- **Browser login (recommended):** `porteden auth login` — opens browser, credentials stored in system keyring
- **Direct token:** `porteden auth login --token <key>` — stored in system keyring
- **Verify:** `porteden auth status`
- If `PE_API_KEY` is set in the environment, the CLI uses it automatically (no login needed).
- Slides access requires a token with `driveAccessEnabled: true` and a connected Google account with Drive scopes.

## Slides commands (`porteden slides`)

### Content

- Deck metadata (slide index + first-line titles): `porteden slides info google:DECKID -jc`
- Read deck text + speaker notes: `porteden slides read google:DECKID`
- Read structured (full Google Slides API JSON): `porteden slides read google:DECKID --format structured -j`
- Create new deck (blank): `porteden slides create --name "Q1 Review"`
- Create in folder: `porteden slides create --name "Kickoff" --folder google:0B7_FOLDER`
- Create with inline text outline: `porteden slides create --name "Kickoff" --content "Agenda\nGoals\nNext steps"`
- Create from text file: `porteden slides create --name "Kickoff" --content-file ./outline.txt`

### File management

- Get export links (pptx, pdf, txt): `porteden slides download google:DECKID -jc`
- Share with user: `porteden slides share google:DECKID --type user --role reader --email user@example.com`
- Share publicly: `porteden slides share google:DECKID --type anyone --role reader`
- List permissions: `porteden slides permissions google:DECKID -jc`
- Rename: `porteden slides rename google:DECKID --name "Q1 Review (final)"`
- Delete (trash): `porteden slides delete google:DECKID -y`

## Reading slides — text vs. structured

- **`--format text` (default):** slide bodies joined with `\n\n---\n\n` separators; speaker notes appended under each slide under a `[Speaker notes]` header. Cheap to call, agent-friendly.
- **`--format structured`:** raw Google Slides API JSON — slide object IDs, page elements, layouts, masters, character ranges. Use only when you need positions/IDs to feed back into another tool. The payload can be many KB even for small decks.

## Notes

- Credentials persist in the system keyring after login. No repeated auth needed.
- Set `PE_PROFILE=work` to avoid repeating `--profile`.
- `-jc` is shorthand for `--json --compact`: strips noise, limits fields, reduces tokens for AI agents.
- **File IDs are always provider-prefixed** (e.g., `google:1BxiMVs0XRA5...`). Pass them as-is.
- `porteden slides info` lists slides by index + the first non-empty text element on each slide as a working title. Use it to plan a content fetch or surface a navigable slide list.
- `porteden slides read --format text` is the default and the cheapest path; only ask for `structured` when you genuinely need slide IDs or page elements.
- **No edit endpoint today.** Slides cannot be modified through the CLI — `presentations.batchUpdate` is not exposed. Use the Slides UI for changes.
- **Layouts, masters, and themes** are only accessible via `--format structured`. Text mode strips them to keep the response useful.
- `porteden slides create` accepts optional `--content`/`--content-file` to seed the deck from a text outline. Default `--content-mime-type` is `text/plain`. Without content flags, a blank deck is created.
- `--content` and `--content-file` are mutually exclusive on `slides create`.
- `porteden slides download` returns **URLs only** — no binary content is streamed. URLs are short-lived (~1 hour) and unauthenticated; re-fetch on every download.
- `accessInfo` in responses describes active token restrictions.
- `delete` moves to trash (reversible). Files can be restored from Google Drive trash.
- Confirm before sharing or deleting.
- Environment variables: `PE_API_KEY`, `PE_PROFILE`, `PE_FORMAT`, `PE_COLOR`, `PE_VERBOSE`.
