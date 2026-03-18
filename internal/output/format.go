package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/porteden/cli/internal/api"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatPlain Format = "plain"
)

// PrintOptions configures output behavior
type PrintOptions struct {
	Compact bool
}

func Print(data interface{}, format Format) {
	PrintWithOptions(data, format, PrintOptions{})
}

func PrintWithOptions(data interface{}, format Format, opts PrintOptions) {
	// Apply compact transformations if enabled
	if opts.Compact {
		data = applyCompact(data)
	}

	switch format {
	case FormatJSON:
		printJSON(data)
	case FormatPlain:
		printPlain(data)
	default:
		printTable(data)
	}
}

// applyCompact applies compact transformations to supported data types
func applyCompact(data interface{}) interface{} {
	compactOpts := DefaultCompactOptions()

	switch v := data.(type) {
	case *api.EventsResponse:
		return CompactEventsResponse(v, compactOpts)
	case *api.Event:
		return CompactEvent(v, compactOpts)
	case *api.SingleEventResponse:
		compacted := CompactEvent(&v.Event, compactOpts)
		return &api.SingleEventResponse{
			Event:                    *compacted,
			AccessInfo:               v.AccessInfo,
			CurrentUserCalendarEmail: v.CurrentUserCalendarEmail,
		}
	case *api.EmailsResponse:
		return CompactEmailsResponse(v, compactOpts)
	case *api.SingleEmailResponse:
		compactedEmail := CompactEmail(&v.Email, compactOpts)
		return &api.SingleEmailResponse{
			Email:      *compactedEmail,
			AccessInfo: v.AccessInfo,
		}
	case *api.Email:
		return CompactEmail(v, compactOpts)
	case *api.ThreadResponse:
		return CompactThreadResponse(v, compactOpts)
	case *api.DriveFilesResponse:
		return CompactDriveFilesResponse(v, compactOpts)
	default:
		return data
	}
}

func printJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

func printPlain(data interface{}) {
	switch v := data.(type) {
	case *api.EventsResponse:
		printEventsPlain(v.Events)
	case *api.CalendarsResponse:
		printCalendarsPlain(v.Data)
	case []api.Event:
		printEventsPlain(v)
	case []api.Calendar:
		printCalendarsPlain(v)
	case *api.Event:
		printEventPlain(*v)
	case *api.SingleEventResponse:
		printEventPlain(v.Event)
		if v.AccessInfo != "" {
			fmt.Printf("Access: %s\n", v.AccessInfo)
		}
	case *api.FreeBusyResponse:
		for _, cal := range v.Calendars {
			for _, b := range cal.Busy {
				fmt.Printf("%d\t%s\t%s\t%s\t%dm\n",
					cal.CalendarID, cal.CalendarName,
					FormatLocalTime(b.StartUtc), FormatLocalTime(b.EndUtc),
					b.DurationMinutes)
			}
		}
	case *api.DeleteEventResponse:
		fmt.Printf("%s\n", v.Message)
	case *api.EmailsResponse:
		printEmailsPlain(v.Emails)
	case *api.SingleEmailResponse:
		printEmailPlain(v.Email)
		if v.AccessInfo != "" {
			fmt.Printf("Access: %s\n", v.AccessInfo)
		}
	case *api.Email:
		printEmailPlain(*v)
	case *api.ThreadResponse:
		printThreadPlain(v)
	// Drive
	case *api.DriveFilesResponse:
		printDriveFilesPlain(v.Files)
		printDriveAccessWarnings(v.AccessInfo, v.AuthWarnings)
	case *api.SingleDriveFileResponse:
		if v.File != nil {
			printDriveFilePlain(*v.File)
		}
		printDriveAccessWarnings(v.AccessInfo, nil)
	case *api.DrivePermissionsResponse:
		for _, p := range v.Permissions {
			email := derefStr(p.EmailAddress)
			domain := derefStr(p.Domain)
			contact := email
			if contact == "" {
				contact = domain
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", p.Type, p.Role, contact, derefStr(p.DisplayName))
		}
	case *api.DriveFileLinkResponse:
		if v.WebViewLink != nil {
			fmt.Printf("web\t%s\n", *v.WebViewLink)
		}
		if v.DownloadUrl != nil {
			fmt.Printf("download\t%s\n", *v.DownloadUrl)
		}
		for format, link := range v.ExportLinks {
			fmt.Printf("export:%s\t%s\n", format, link)
		}
	case *api.DriveOperationResult:
		if v.Success {
			if v.FileID != nil {
				fmt.Printf("success\t%s\n", *v.FileID)
			} else {
				fmt.Println("success")
			}
		} else {
			fmt.Printf("error\t%s\n", derefStr(v.ErrorMessage))
		}
	// Docs
	case *api.DocContentResponse:
		if v.PlainText != nil {
			fmt.Print(*v.PlainText)
		} else if v.StructuredContent != nil {
			printJSON(v.StructuredContent)
		}
		printDriveAccessWarnings(v.AccessInfo, nil)
	// Sheets
	case *api.SheetMetadataResponse:
		title := derefStr(v.Title)
		fmt.Printf("%s\t%s\n", v.SpreadsheetID, title)
		for _, s := range v.Sheets {
			fmt.Printf("%d\t%s\t%d\t%d\n", s.SheetID, s.Title, s.RowCount, s.ColumnCount)
		}
		printDriveAccessWarnings(v.AccessInfo, nil)
	case *api.SheetValuesResponse:
		printSheetValuesPlain(v)
	}
}

func printTable(data interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	switch v := data.(type) {
	// Handle wrapped API responses
	case *api.EventsResponse:
		printEventsTable(w, v.Events, v.Meta)
		if v.AccessInfo != "" {
			fmt.Fprintf(w, "\nAccess: %s\n", v.AccessInfo)
		}
	case *api.CalendarsResponse:
		printCalendarsTable(w, v.Data)
		if v.AccessInfo != "" {
			fmt.Fprintf(w, "\nAccess: %s\n", v.AccessInfo)
		}
	// Handle unwrapped slices (for backward compatibility)
	case []api.Event:
		printEventsTable(w, v, nil)
	case []api.Calendar:
		printCalendarsTable(w, v)
	case *api.Event:
		printEventDetail(w, *v)
	case *api.SingleEventResponse:
		printEventDetail(w, v.Event)
		if v.AccessInfo != "" {
			fmt.Fprintf(w, "\nAccess:\t%s\n", v.AccessInfo)
		}
	case *api.FreeBusyResponse:
		printFreeBusyTable(w, v)
	case *api.DeleteEventResponse:
		fmt.Fprintf(w, "%s\n", v.Message)
	case *api.EmailsResponse:
		printEmailsTable(w, v.Emails, v.TotalCount, v.HasMore)
		if v.AccessInfo != "" {
			fmt.Fprintf(w, "\nAccess: %s\n", v.AccessInfo)
		}
	case *api.SingleEmailResponse:
		printEmailDetail(w, v.Email)
		if v.AccessInfo != "" {
			fmt.Fprintf(w, "\nAccess:\t%s\n", v.AccessInfo)
		}
	case *api.Email:
		printEmailDetail(w, *v)
	case *api.ThreadResponse:
		printThreadTable(w, v)
	// Drive
	case *api.DriveFilesResponse:
		printDriveFilesTable(w, v.Files, v.HasMore)
		printDriveAccessWarningsTable(w, v.AccessInfo, v.AuthWarnings)
	case *api.SingleDriveFileResponse:
		if v.File != nil {
			printDriveFileDetail(w, *v.File)
		}
		printDriveAccessWarningsTable(w, v.AccessInfo, nil)
	case *api.DrivePermissionsResponse:
		printDrivePermissionsTable(w, v.Permissions)
		printDriveAccessWarningsTable(w, v.AccessInfo, nil)
	case *api.DriveFileLinkResponse:
		printDriveFileLinksTable(w, v)
	case *api.DriveOperationResult:
		printDriveOperationResult(v)
	// Docs
	case *api.DocContentResponse:
		w.Flush() // flush tabwriter before raw output
		if v.PlainText != nil {
			fmt.Print(*v.PlainText)
		} else if v.StructuredContent != nil {
			printJSON(v.StructuredContent)
		}
		if v.AccessInfo != nil && *v.AccessInfo != "" {
			fmt.Fprintf(os.Stderr, "\nAccess: %s\n", *v.AccessInfo)
		}
	// Sheets
	case *api.SheetMetadataResponse:
		printSheetMetadataTable(w, v)
	case *api.SheetValuesResponse:
		printSheetValuesTable(w, v)
	}
}

func printEventsTable(w *tabwriter.Writer, events []api.Event, meta *api.Meta) {
	fmt.Fprintln(w, "ID\tDATE\tTIME\tDURATION\tTITLE\tSTATUS")
	fmt.Fprintln(w, "──\t────\t────\t────────\t─────\t──────")
	for _, e := range events {
		localStart := GetLocalStart(e.StartLocal, e.StartUtc)
		title := e.Title
		if title == "" {
			title = e.Summary // Fallback to summary if title is empty
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%dm\t%s\t%s\n",
			e.ID,
			safeDate(localStart),
			safeTime(localStart),
			e.DurationMinutes,
			truncate(title, 30),
			ColorStatus(e.Status),
		)
	}

	// Display pagination info if available
	if meta != nil && meta.TotalCount > 0 {
		start := meta.Offset + 1
		end := meta.Offset + meta.Count
		if meta.HasMore {
			fmt.Fprintf(w, "\nShowing %d-%d of %d (use --offset %d for more)\n",
				start, end, meta.TotalCount, end)
		} else {
			fmt.Fprintf(w, "\nShowing %d-%d of %d\n", start, end, meta.TotalCount)
		}
	}
}

func printEventDetail(w *tabwriter.Writer, e api.Event) {
	title := e.Title
	if title == "" {
		title = e.Summary
	}
	fmt.Fprintf(w, "ID:\t%s\n", e.ID)
	fmt.Fprintf(w, "Title:\t%s\n", title)
	fmt.Fprintf(w, "Start:\t%s\n", GetLocalStart(e.StartLocal, e.StartUtc))
	fmt.Fprintf(w, "End:\t%s\n", GetLocalEnd(e.EndLocal, e.EndUtc))
	fmt.Fprintf(w, "Duration:\t%d minutes\n", e.DurationMinutes)
	fmt.Fprintf(w, "Status:\t%s\n", ColorStatus(e.Status))
	if e.Location != "" {
		fmt.Fprintf(w, "Location:\t%s\n", e.Location)
	}
	if e.Description != "" {
		fmt.Fprintf(w, "Description:\t%s\n", e.Description)
	}
	if e.Organizer != "" {
		fmt.Fprintf(w, "Organizer:\t%s\n", e.Organizer)
	}
	if e.JoinUrl != "" {
		fmt.Fprintf(w, "Join URL:\t%s\n", e.JoinUrl)
	}
	if len(e.Attendees) > 0 {
		fmt.Fprintln(w, "Attendees:")
		for _, a := range e.Attendees {
			name := a.Name
			if name == "" {
				name = a.DisplayName
			}
			if name == "" {
				name = a.Email
			}
			status := a.Response
			if status == "" {
				status = a.ResponseStatus
			}
			if status == "" {
				status = "needsAction"
			}
			fmt.Fprintf(w, "  - %s\t(%s)\n", name, status)
		}
	}
}

func printCalendarsTable(w *tabwriter.Writer, calendars []api.Calendar) {
	fmt.Fprintln(w, "ID\tNAME\tPROVIDER\tTIMEZONE\tPRIMARY\tOWNER")
	fmt.Fprintln(w, "──\t────\t────────\t────────\t───────\t─────")
	for _, c := range calendars {
		primary := ""
		if c.IsPrimary {
			primary = "yes"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", c.ID, c.Name, c.Provider, c.Timezone, primary, c.OwnerEmail)
	}
}

func printFreeBusyTable(w *tabwriter.Writer, resp *api.FreeBusyResponse) {
	for _, cal := range resp.Calendars {
		fmt.Fprintf(w, "Calendar: %s (ID: %d)\n", cal.CalendarName, cal.CalendarID)
		fmt.Fprintln(w, "  START\tEND\tDURATION")
		fmt.Fprintln(w, "  ─────\t───\t────────")
		for _, b := range cal.Busy {
			fmt.Fprintf(w, "  %s\t%s\t%dm\n",
				FormatLocalTime(b.StartUtc),
				FormatLocalTime(b.EndUtc),
				b.DurationMinutes)
		}
		fmt.Fprintln(w)
	}
	if resp.AccessInfo != "" {
		fmt.Fprintf(w, "Access: %s\n", resp.AccessInfo)
	}
}

func printEventsPlain(events []api.Event) {
	for _, e := range events {
		localStart := GetLocalStart(e.StartLocal, e.StartUtc)
		title := e.Title
		if title == "" {
			title = e.Summary
		}
		fmt.Printf("%s\t%s\t%s\t%dm\t%s\t%s\n",
			e.ID,
			safeDate(localStart),
			safeTime(localStart),
			e.DurationMinutes,
			title,
			e.Status,
		)
	}
}

func printEventPlain(e api.Event) {
	title := e.Title
	if title == "" {
		title = e.Summary
	}
	fmt.Printf("ID: %s\n", e.ID)
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Start: %s\n", GetLocalStart(e.StartLocal, e.StartUtc))
	fmt.Printf("End: %s\n", GetLocalEnd(e.EndLocal, e.EndUtc))
	fmt.Printf("Duration: %d minutes\n", e.DurationMinutes)
	fmt.Printf("Status: %s\n", e.Status)
	if e.Location != "" {
		fmt.Printf("Location: %s\n", e.Location)
	}
}

func printCalendarsPlain(calendars []api.Calendar) {
	for _, c := range calendars {
		primary := "false"
		if c.IsPrimary {
			primary = "true"
		}
		fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\n", c.ID, c.Name, c.Provider, c.Timezone, primary, c.OwnerEmail)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// ==================== EMAIL FORMATTERS ====================

func printEmailsTable(w *tabwriter.Writer, emails []api.Email, totalCount int, hasMore bool) {
	fmt.Fprintln(w, "ID\tDATE\tFROM\tSUBJECT\tREAD\tATTACH")
	fmt.Fprintln(w, "──\t────\t────\t───────\t────\t──────")

	for _, e := range emails {
		from := ""
		if e.From != nil {
			if e.From.Name != "" {
				from = e.From.Name
			} else {
				from = e.From.Email
			}
		}

		readStatus := ColorGreen("yes")
		if !e.IsRead {
			readStatus = ColorYellow("no")
		}

		attach := ""
		if e.HasAttachments {
			attach = "yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(e.ID, 24),
			safeDate(FormatLocalTime(e.ReceivedAt)),
			truncate(from, 24),
			truncate(e.Subject, 40),
			readStatus,
			attach,
		)
	}

	if totalCount > 0 || len(emails) > 0 {
		shown := len(emails)
		if hasMore {
			fmt.Fprintf(w, "\nShowing %d emails (more available, use --all to fetch all)\n", shown)
		} else if totalCount > 0 {
			fmt.Fprintf(w, "\nShowing %d of %d emails\n", shown, totalCount)
		}
	}
}

func printEmailDetail(w *tabwriter.Writer, e api.Email) {
	fmt.Fprintf(w, "ID:\t%s\n", e.ID)
	if e.ThreadID != "" {
		fmt.Fprintf(w, "Thread:\t%s\n", e.ThreadID)
	}
	fmt.Fprintf(w, "Subject:\t%s\n", e.Subject)

	if e.From != nil {
		fmt.Fprintf(w, "From:\t%s\n", formatParticipant(*e.From))
	}

	if len(e.To) > 0 {
		fmt.Fprintf(w, "To:\t%s\n", formatParticipants(e.To))
	}

	if len(e.CC) > 0 {
		fmt.Fprintf(w, "CC:\t%s\n", formatParticipants(e.CC))
	}

	if !e.SentAt.IsZero() {
		fmt.Fprintf(w, "Sent:\t%s\n", FormatLocalTime(e.SentAt))
	}
	if !e.ReceivedAt.IsZero() {
		fmt.Fprintf(w, "Received:\t%s\n", FormatLocalTime(e.ReceivedAt))
	}

	fmt.Fprintf(w, "Read:\t%v\n", e.IsRead)

	if len(e.Labels) > 0 {
		fmt.Fprintf(w, "Labels:\t%s\n", strings.Join(e.Labels, ", "))
	}

	if e.Importance != "" && e.Importance != "normal" {
		fmt.Fprintf(w, "Importance:\t%s\n", e.Importance)
	}

	fmt.Fprintf(w, "Provider:\t%s\n", e.Provider)

	if e.HasAttachments && len(e.Attachments) > 0 {
		fmt.Fprintln(w, "Attachments:")
		for _, att := range e.Attachments {
			sizeStr := formatBytes(att.Size)
			if att.ContentType != "" {
				fmt.Fprintf(w, "  - %s\t(%s, %s)\n", att.Name, att.ContentType, sizeStr)
			} else {
				fmt.Fprintf(w, "  - %s\t(%s)\n", att.Name, sizeStr)
			}
		}
	}

	if e.Body != "" {
		fmt.Fprintf(w, "\n%s\n", e.Body)
	} else if e.BodyPreview != "" {
		fmt.Fprintf(w, "\n%s\n", e.BodyPreview)
	}
}

func printThreadTable(w *tabwriter.Writer, t *api.ThreadResponse) {
	fmt.Fprintf(w, "Thread ID:\t%s\n", t.ID)
	fmt.Fprintf(w, "Subject:\t%s\n", t.Subject)
	fmt.Fprintf(w, "Messages:\t%d\n", t.MessageCount)
	if !t.LastMessageAt.IsZero() {
		fmt.Fprintf(w, "Last Message:\t%s\n", FormatLocalTime(t.LastMessageAt))
	}
	fmt.Fprintf(w, "Provider:\t%s\n", t.Provider)

	if len(t.Participants) > 0 {
		fmt.Fprintf(w, "Participants:\t%s\n", formatParticipants(t.Participants))
	}

	if t.AccessInfo != "" {
		fmt.Fprintf(w, "Access:\t%s\n", t.AccessInfo)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "ID\tFROM\tSENT\tREAD")
	fmt.Fprintln(w, "──\t────\t────\t────")

	for _, msg := range t.Messages {
		from := ""
		if msg.From != nil {
			if msg.From.Name != "" {
				from = msg.From.Name
			} else {
				from = msg.From.Email
			}
		}

		readStatus := ColorGreen("yes")
		if !msg.IsRead {
			readStatus = ColorYellow("no")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			truncate(msg.ID, 24),
			truncate(from, 24),
			FormatLocalTime(msg.SentAt),
			readStatus,
		)
	}
}

func printEmailsPlain(emails []api.Email) {
	for _, e := range emails {
		from := ""
		if e.From != nil {
			from = e.From.Email
		}
		fmt.Printf("%s\t%s\t%s\t%s\t%v\t%v\n",
			e.ID,
			safeDate(FormatLocalTime(e.ReceivedAt)),
			from,
			e.Subject,
			e.IsRead,
			e.HasAttachments,
		)
	}
}

func printEmailPlain(e api.Email) {
	fmt.Printf("ID: %s\n", e.ID)
	fmt.Printf("Subject: %s\n", e.Subject)
	if e.From != nil {
		fmt.Printf("From: %s\n", e.From.Email)
	}
	if !e.ReceivedAt.IsZero() {
		fmt.Printf("Received: %s\n", FormatLocalTime(e.ReceivedAt))
	}
	fmt.Printf("Read: %v\n", e.IsRead)
	if e.Body != "" {
		fmt.Printf("\n%s\n", e.Body)
	} else if e.BodyPreview != "" {
		fmt.Printf("\n%s\n", e.BodyPreview)
	}
}

func printThreadPlain(t *api.ThreadResponse) {
	fmt.Printf("Thread: %s\n", t.ID)
	fmt.Printf("Subject: %s\n", t.Subject)
	fmt.Printf("Messages: %d\n", t.MessageCount)
	for _, msg := range t.Messages {
		from := ""
		if msg.From != nil {
			from = msg.From.Email
		}
		fmt.Printf("%s\t%s\t%s\t%v\n", msg.ID, from, FormatLocalTime(msg.SentAt), msg.IsRead)
	}
}

func formatParticipant(p api.Participant) string {
	if p.Name != "" {
		return fmt.Sprintf("%s <%s>", p.Name, p.Email)
	}
	return p.Email
}

func formatParticipants(ps []api.Participant) string {
	parts := make([]string, len(ps))
	for i, p := range ps {
		parts[i] = formatParticipant(p)
	}
	return strings.Join(parts, ", ")
}

// ==================== DRIVE FORMATTERS ====================

func friendlyMimeType(mimeType string, isFolder bool) string {
	if isFolder {
		return "Folder"
	}
	switch mimeType {
	case "application/vnd.google-apps.document":
		return "Doc"
	case "application/vnd.google-apps.spreadsheet":
		return "Sheet"
	case "application/vnd.google-apps.presentation":
		return "Slide"
	case "application/vnd.google-apps.drawing":
		return "Drawing"
	case "application/vnd.google-apps.form":
		return "Form"
	case "application/pdf":
		return "PDF"
	case "application/zip":
		return "ZIP"
	}
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "Image"
	case strings.HasPrefix(mimeType, "video/"):
		return "Video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "Audio"
	case strings.HasPrefix(mimeType, "text/"):
		return "Text"
	default:
		if mimeType == "" {
			return "File"
		}
		// Return last segment of mime type
		parts := strings.Split(mimeType, "/")
		return parts[len(parts)-1]
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func driveFileOwner(f api.DriveFile) string {
	if len(f.Owners) == 0 {
		return ""
	}
	return f.Owners[0].Email
}

func driveFileSize(f api.DriveFile) string {
	if f.Size == nil || f.IsFolder {
		return "—"
	}
	return formatBytes(*f.Size)
}

func driveFileModified(f api.DriveFile) string {
	if f.ModifiedTime == nil {
		return ""
	}
	// ModifiedTime is ISO 8601; parse and format as short date
	t := *f.ModifiedTime
	if len(t) >= 10 {
		return t[:10]
	}
	return t
}

func printDriveFilesTable(w *tabwriter.Writer, files []api.DriveFile, hasMore bool) {
	fmt.Fprintln(w, "ID\tTYPE\tNAME\tSIZE\tMODIFIED\tOWNER")
	fmt.Fprintln(w, "──\t────\t────\t────\t────────\t─────")
	for _, f := range files {
		mimeType := derefStr(f.MimeType)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(f.ID, 22),
			friendlyMimeType(mimeType, f.IsFolder),
			truncate(derefStr(f.Name), 35),
			driveFileSize(f),
			driveFileModified(f),
			truncate(driveFileOwner(f), 30),
		)
	}
	if len(files) > 0 && hasMore {
		fmt.Fprintf(w, "\nShowing %d files (more available, use --all to fetch all)\n", len(files))
	}
}

func printDriveFileDetail(w *tabwriter.Writer, f api.DriveFile) {
	fmt.Fprintf(w, "ID:\t%s\n", f.ID)
	fmt.Fprintf(w, "Name:\t%s\n", derefStr(f.Name))
	fmt.Fprintf(w, "Type:\t%s\n", friendlyMimeType(derefStr(f.MimeType), f.IsFolder))
	fmt.Fprintf(w, "MIME:\t%s\n", derefStr(f.MimeType))
	fmt.Fprintf(w, "Size:\t%s\n", driveFileSize(f))
	if f.CreatedTime != nil {
		fmt.Fprintf(w, "Created:\t%s\n", *f.CreatedTime)
	}
	if f.ModifiedTime != nil {
		fmt.Fprintf(w, "Modified:\t%s\n", *f.ModifiedTime)
	}
	if len(f.Owners) > 0 {
		emails := make([]string, len(f.Owners))
		for i, o := range f.Owners {
			emails[i] = o.Email
		}
		fmt.Fprintf(w, "Owners:\t%s\n", strings.Join(emails, ", "))
	}
	if f.ParentFolderName != nil || f.ParentFolderID != nil {
		parent := derefStr(f.ParentFolderName)
		if parent == "" {
			parent = derefStr(f.ParentFolderID)
		}
		fmt.Fprintf(w, "Parent:\t%s\n", parent)
	}
	if f.WebViewLink != nil {
		fmt.Fprintf(w, "Web Link:\t%s\n", *f.WebViewLink)
	}
	if f.DownloadLink != nil {
		fmt.Fprintf(w, "Download:\t%s\n", *f.DownloadLink)
	}
	if f.Description != nil && *f.Description != "" {
		fmt.Fprintf(w, "Description:\t%s\n", truncate(*f.Description, 80))
	}
	fmt.Fprintf(w, "Provider:\t%s\n", f.Provider)
}

func printDrivePermissionsTable(w *tabwriter.Writer, perms []api.DrivePermission) {
	fmt.Fprintln(w, "TYPE\tROLE\tEMAIL / DOMAIN\tNAME")
	fmt.Fprintln(w, "────\t────\t──────────────\t────")
	for _, p := range perms {
		contact := derefStr(p.EmailAddress)
		if contact == "" {
			contact = derefStr(p.Domain)
		}
		if contact == "" {
			contact = "anyone"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Type, p.Role, contact, derefStr(p.DisplayName))
	}
}

func printDriveFileLinksTable(w *tabwriter.Writer, v *api.DriveFileLinkResponse) {
	if !v.Success {
		fmt.Fprintf(w, "Error:\t%s\n", derefStr(v.ErrorMessage))
		return
	}
	if v.FileName != nil {
		fmt.Fprintf(w, "File:\t%s\n", *v.FileName)
	}
	if v.MimeType != nil {
		fmt.Fprintf(w, "Type:\t%s\n", friendlyMimeType(*v.MimeType, false))
	}
	if v.WebViewLink != nil {
		fmt.Fprintf(w, "View:\t%s\n", *v.WebViewLink)
	}
	if v.DownloadUrl != nil {
		fmt.Fprintf(w, "Download:\t%s\n", *v.DownloadUrl)
	}
	if len(v.ExportLinks) > 0 {
		fmt.Fprintln(w, "\nEXPORT FORMAT\tURL")
		fmt.Fprintln(w, "─────────────\t───")
		for format, link := range v.ExportLinks {
			fmt.Fprintf(w, "%s\t%s\n", format, link)
		}
	}
}

func printDriveOperationResult(v *api.DriveOperationResult) {
	if v.Success {
		if v.FileID != nil && *v.FileID != "" {
			fmt.Printf("✓ Done  (id: %s)\n", *v.FileID)
		} else {
			fmt.Println("✓ Done")
		}
	} else {
		msg := derefStr(v.ErrorMessage)
		if msg == "" {
			msg = derefStr(v.ErrorCode)
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
}

func printDriveAccessWarningsTable(w *tabwriter.Writer, accessInfo *string, warnings []string) {
	if accessInfo != nil && *accessInfo != "" {
		fmt.Fprintf(w, "\nAccess: %s\n", *accessInfo)
	}
	for _, warn := range warnings {
		fmt.Fprintf(w, ColorYellow("Warning: %s\n"), warn)
	}
}

func printDriveAccessWarnings(accessInfo *string, warnings []string) {
	if accessInfo != nil && *accessInfo != "" {
		fmt.Printf("\nAccess: %s\n", *accessInfo)
	}
	for _, warn := range warnings {
		fmt.Printf(ColorYellow("Warning: %s\n"), warn)
	}
}

func printDriveFilesPlain(files []api.DriveFile) {
	for _, f := range files {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
			f.ID,
			friendlyMimeType(derefStr(f.MimeType), f.IsFolder),
			derefStr(f.Name),
			driveFileSize(f),
			driveFileModified(f),
			driveFileOwner(f),
		)
	}
}

func printDriveFilePlain(f api.DriveFile) {
	fmt.Printf("ID: %s\n", f.ID)
	fmt.Printf("Name: %s\n", derefStr(f.Name))
	fmt.Printf("Type: %s\n", friendlyMimeType(derefStr(f.MimeType), f.IsFolder))
	fmt.Printf("Size: %s\n", driveFileSize(f))
	if f.ModifiedTime != nil {
		fmt.Printf("Modified: %s\n", *f.ModifiedTime)
	}
	fmt.Printf("Owner: %s\n", driveFileOwner(f))
}

// ==================== SHEETS FORMATTERS ====================

func printSheetMetadataTable(w *tabwriter.Writer, v *api.SheetMetadataResponse) {
	title := derefStr(v.Title)
	if title == "" {
		title = v.SpreadsheetID
	}
	fmt.Fprintf(w, "Spreadsheet:\t%s\n", title)
	fmt.Fprintf(w, "ID:\t%s\n", v.SpreadsheetID)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "SHEET\tROWS\tCOLS")
	fmt.Fprintln(w, "─────\t────\t────")
	for _, s := range v.Sheets {
		fmt.Fprintf(w, "%s\t%d\t%d\n", s.Title, s.RowCount, s.ColumnCount)
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printSheetValuesTable(w *tabwriter.Writer, v *api.SheetValuesResponse) {
	if len(v.Values) == 0 {
		fmt.Fprintln(w, "(empty range)")
		return
	}
	// Determine max columns
	maxCols := 0
	for _, row := range v.Values {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Print header row with separators
	for i, cell := range v.Values[0] {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprintf(w, "%v", cell)
	}
	fmt.Fprintln(w)
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, "────")
	}
	fmt.Fprintln(w)

	// Data rows
	for _, row := range v.Values[1:] {
		for i := 0; i < maxCols; i++ {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			if i < len(row) {
				fmt.Fprintf(w, "%v", row[i])
			}
		}
		fmt.Fprintln(w)
	}
	if v.Range != "" {
		fmt.Fprintf(w, "\nRange: %s\n", v.Range)
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printSheetValuesPlain(v *api.SheetValuesResponse) {
	for _, row := range v.Values {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = fmt.Sprintf("%v", cell)
		}
		fmt.Println(strings.Join(cells, "\t"))
	}
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
