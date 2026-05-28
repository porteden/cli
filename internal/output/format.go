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
	case *api.SheetBulkContentResponse:
		return CompactSheetBulkContentResponse(v, compactOpts)
	case *api.TaskItemsResponse:
		return CompactTaskItemsResponse(v, compactOpts)
	case *api.TaskItemResponse:
		return CompactTaskItemResponse(v, compactOpts)
	case *api.TaskSearchResponse:
		return CompactTaskSearchResponse(v, compactOpts)
	case *api.TaskBlockListResponse:
		return CompactTaskBlockListResponse(v, compactOpts)
	case *api.TaskItemResult:
		return CompactTaskItemResult(v, compactOpts)
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
		if v.AccessInfo != "" {
			fmt.Printf("Access: %s\n", v.AccessInfo)
		}
		for _, warn := range v.AuthWarnings {
			fmt.Printf("Warning: %s\n", warn)
		}
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
	case *api.SheetBulkContentResponse:
		printSheetBulkContentPlain(v)
	case *api.DriveFileContentResponse:
		printDriveFileContentPlain(v)
	// Slides
	case *api.SlidesMetadataResponse:
		printSlidesMetadataPlain(v)
	case *api.SlidesContentResponse:
		printSlidesContentPlain(v)
	// Tasks
	case *api.TaskProvidersResponse:
		printTaskProvidersPlain(*v)
	case *api.TaskBoardsResponse:
		printTaskBoardsPlain(v)
	case *api.TaskBoardResponse:
		printTaskBoardDetailPlain(v)
	case *api.TaskItemsResponse:
		printTaskItemsPlain(v)
	case *api.TaskItemResponse:
		if v.Item != nil {
			printTaskItemDetailPlain(*v.Item)
		}
		if footer := taskProviderFooter(v.Provider); footer != "" {
			fmt.Print("\n", footer)
		}
		printDriveAccessWarnings(v.AccessInfo, nil)
	case *api.TaskCommentsResponse:
		printTaskCommentsPlain(v)
	case *api.TaskSearchResponse:
		printTaskSearchPlain(v)
	case *api.TaskBlockListResponse:
		printTaskBlockListPlain(v)
	case *api.TaskItemResult:
		printTaskItemResult(v)
	case *api.TaskOperationResult:
		printTaskOperationResult(v)
	case *api.TaskCommentResult:
		printTaskCommentResult(v)
	case *api.AppendBlocksResponse:
		printAppendBlocksResponse(v)
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
		printEmailsTable(w, v.Emails, v.HasMore)
		if v.AccessInfo != "" {
			fmt.Fprintf(w, "\nAccess: %s\n", v.AccessInfo)
		}
		for _, warn := range v.AuthWarnings {
			fmt.Fprintf(w, ColorYellow("Warning: %s\n"), warn)
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
	case *api.SheetBulkContentResponse:
		printSheetBulkContentTable(w, v)
	case *api.DriveFileContentResponse:
		printDriveFileContentTable(w, v)
	// Slides
	case *api.SlidesMetadataResponse:
		printSlidesMetadataTable(w, v)
	case *api.SlidesContentResponse:
		printSlidesContentTable(w, v)
	// Tasks
	case *api.TaskProvidersResponse:
		printTaskProvidersTable(w, *v)
	case *api.TaskBoardsResponse:
		printTaskBoardsTable(w, v)
	case *api.TaskBoardResponse:
		printTaskBoardDetailTable(w, v)
	case *api.TaskItemsResponse:
		printTaskItemsTable(w, v)
	case *api.TaskItemResponse:
		if v.Item != nil {
			printTaskItemDetailTable(w, *v.Item)
		}
		if footer := taskProviderFooter(v.Provider); footer != "" {
			fmt.Fprint(w, "\n", footer)
		}
		printDriveAccessWarningsTable(w, v.AccessInfo, nil)
	case *api.TaskCommentsResponse:
		printTaskCommentsTable(w, v)
	case *api.TaskSearchResponse:
		printTaskSearchTable(w, v)
	case *api.TaskBlockListResponse:
		printTaskBlockListTable(w, v)
	case *api.TaskItemResult:
		w.Flush()
		printTaskItemResult(v)
	case *api.TaskOperationResult:
		w.Flush()
		printTaskOperationResult(v)
	case *api.TaskCommentResult:
		w.Flush()
		printTaskCommentResult(v)
	case *api.AppendBlocksResponse:
		w.Flush()
		printAppendBlocksResponse(v)
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

func printEmailsTable(w *tabwriter.Writer, emails []api.Email, hasMore bool) {
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
			safeDate(FormatLocalTimePtr(e.ReceivedAt)),
			truncate(from, 24),
			truncate(e.Subject, 40),
			readStatus,
			attach,
		)
	}

	// The API does not expose a post-firewall total — drive iteration by hasMore.
	if len(emails) > 0 {
		if hasMore {
			fmt.Fprintf(w, "\nShowing %d emails (more available, use --all to fetch all)\n", len(emails))
		} else {
			fmt.Fprintf(w, "\nShowing %d emails\n", len(emails))
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

	if e.SentAt != nil {
		fmt.Fprintf(w, "Sent:\t%s\n", FormatLocalTime(*e.SentAt))
	}
	if e.ReceivedAt != nil {
		fmt.Fprintf(w, "Received:\t%s\n", FormatLocalTime(*e.ReceivedAt))
	}

	fmt.Fprintf(w, "Read:\t%v\n", e.IsRead)
	fmt.Fprintf(w, "Direction:\t%s\n", emailDirection(e.IsOutbound))

	if len(e.Labels) > 0 {
		fmt.Fprintf(w, "Labels:\t%s\n", strings.Join(e.Labels, ", "))
	}

	if e.Importance != "" && e.Importance != "normal" {
		fmt.Fprintf(w, "Importance:\t%s\n", e.Importance)
	}

	fmt.Fprintf(w, "Provider:\t%s\n", e.Provider)
	if e.EmailAccountOwner != "" {
		fmt.Fprintf(w, "Account:\t%s\n", e.EmailAccountOwner)
	}

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
	if t.LastMessageAt != nil {
		fmt.Fprintf(w, "Last Message:\t%s\n", FormatLocalTime(*t.LastMessageAt))
	}
	fmt.Fprintf(w, "Provider:\t%s\n", t.Provider)

	if len(t.Participants) > 0 {
		fmt.Fprintf(w, "Participants:\t%s\n", formatParticipants(t.Participants))
	}

	if t.AccessInfo != "" {
		fmt.Fprintf(w, "Access:\t%s\n", t.AccessInfo)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "ID\tDIR\tFROM\tSENT\tREAD")
	fmt.Fprintln(w, "──\t───\t────\t────\t────")

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

		dir := "in"
		if msg.IsOutbound {
			dir = "out"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			truncate(msg.ID, 24),
			dir,
			truncate(from, 24),
			FormatLocalTimePtr(msg.SentAt),
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
			safeDate(FormatLocalTimePtr(e.ReceivedAt)),
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
	if e.ReceivedAt != nil {
		fmt.Printf("Received: %s\n", FormatLocalTime(*e.ReceivedAt))
	}
	fmt.Printf("Read: %v\n", e.IsRead)
	fmt.Printf("Direction: %s\n", emailDirection(e.IsOutbound))
	if e.EmailAccountOwner != "" {
		fmt.Printf("Account: %s\n", e.EmailAccountOwner)
	}
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
		fmt.Printf("%s\t%s\t%s\t%v\n", msg.ID, from, FormatLocalTimePtr(msg.SentAt), msg.IsRead)
	}
}

func emailDirection(isOutbound bool) string {
	if isOutbound {
		return "outbound"
	}
	return "inbound"
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

// ==================== DRIVE FILE CONTENT FORMATTERS ====================

// driveContentReadableHint converts a server reason code into a one-line steer
// for the human reader. Empty when the file was read successfully.
func driveContentReadableHint(v *api.DriveFileContentResponse) string {
	if v.Readable {
		if v.Truncated {
			return "Content truncated at the 10 MB cap — use webContentLink for the full file."
		}
		return ""
	}
	reason := derefStr(v.Reason)
	switch reason {
	case "USE_SHEETS_ENDPOINT":
		return "This file is a spreadsheet. Use: porteden sheets content <fileId>"
	case "USE_SLIDES_ENDPOINT":
		return "This file is a presentation. Use: porteden slides read <fileId>"
	case "BINARY_CONTENT":
		return "Binary file — open via webViewLink."
	case "TOO_LARGE":
		return "File exceeds the 10 MB cap — fetch via webContentLink."
	case "EXPORT_FAILED":
		return "Workspace export failed — try `porteden drive download <fileId>` for exportLinks."
	}
	if reason != "" {
		return "Not readable: " + reason
	}
	return "Not readable."
}

func printDriveFileContentTable(w *tabwriter.Writer, v *api.DriveFileContentResponse) {
	w.Flush()
	if v.Readable && v.Content != nil {
		fmt.Print(*v.Content)
		if !strings.HasSuffix(*v.Content, "\n") {
			fmt.Println()
		}
	}
	if hint := driveContentReadableHint(v); hint != "" {
		fmt.Fprintln(os.Stderr, hint)
	}
	if v.WebViewLink != nil && *v.WebViewLink != "" {
		fmt.Fprintf(os.Stderr, "View:     %s\n", *v.WebViewLink)
	}
	if v.WebContentLink != nil && *v.WebContentLink != "" {
		fmt.Fprintf(os.Stderr, "Download: %s\n", *v.WebContentLink)
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func printDriveFileContentPlain(v *api.DriveFileContentResponse) {
	if v.Readable && v.Content != nil {
		fmt.Print(*v.Content)
		if !strings.HasSuffix(*v.Content, "\n") {
			fmt.Println()
		}
	}
	if hint := driveContentReadableHint(v); hint != "" {
		fmt.Fprintln(os.Stderr, hint)
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

// ==================== SHEETS BULK CONTENT FORMATTERS ====================

func printSheetBulkContentTable(w *tabwriter.Writer, v *api.SheetBulkContentResponse) {
	title := derefStr(v.Title)
	if title == "" {
		title = v.SpreadsheetID
	}
	fmt.Fprintf(w, "Spreadsheet:\t%s\n", title)
	fmt.Fprintf(w, "ID:\t%s\n", v.SpreadsheetID)
	for _, tab := range v.Sheets {
		fmt.Fprintln(w)
		marker := ""
		if tab.Clipped {
			marker = ColorYellow(" (clipped)")
		}
		fmt.Fprintf(w, "── %s%s ── (%s)\n", tab.Title, marker, tab.Range)
		printValuesGrid(w, tab.Values)
		if tab.Clipped && tab.FullRange != nil {
			fmt.Fprintf(w, "Full range: %s — fetch with `porteden sheets read --range '%s'`\n", *tab.FullRange, *tab.FullRange)
		}
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printSheetBulkContentPlain(v *api.SheetBulkContentResponse) {
	for _, tab := range v.Sheets {
		fmt.Printf("# %s\t%s", tab.Title, tab.Range)
		if tab.Clipped {
			fmt.Print("\tclipped")
		}
		fmt.Println()
		for _, row := range tab.Values {
			cells := make([]string, len(row))
			for i, c := range row {
				cells[i] = fmt.Sprintf("%v", c)
			}
			fmt.Println(strings.Join(cells, "\t"))
		}
		if tab.Clipped && tab.FullRange != nil {
			fmt.Printf("# fullRange\t%s\n", *tab.FullRange)
		}
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

// printValuesGrid prints a 2D values grid with a header underline.
// Shared by /sheets/content per-tab rendering.
func printValuesGrid(w *tabwriter.Writer, values [][]interface{}) {
	if len(values) == 0 {
		fmt.Fprintln(w, "(empty)")
		return
	}
	maxCols := 0
	for _, row := range values {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	for i, cell := range values[0] {
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
	for _, row := range values[1:] {
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
}

// ==================== SLIDES FORMATTERS ====================

func printSlidesMetadataTable(w *tabwriter.Writer, v *api.SlidesMetadataResponse) {
	title := derefStr(v.Title)
	if title == "" {
		title = v.PresentationID
	}
	fmt.Fprintf(w, "Presentation:\t%s\n", title)
	fmt.Fprintf(w, "ID:\t%s\n", v.PresentationID)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "INDEX\tTITLE")
	fmt.Fprintln(w, "─────\t─────")
	for _, s := range v.Slides {
		fmt.Fprintf(w, "%d\t%s\n", s.Index, derefStr(s.Title))
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printSlidesMetadataPlain(v *api.SlidesMetadataResponse) {
	for _, s := range v.Slides {
		fmt.Printf("%d\t%s\n", s.Index, derefStr(s.Title))
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func printSlidesContentTable(w *tabwriter.Writer, v *api.SlidesContentResponse) {
	w.Flush()
	if v.PlainText != nil {
		fmt.Print(*v.PlainText)
		if !strings.HasSuffix(*v.PlainText, "\n") {
			fmt.Println()
		}
	} else if v.StructuredContent != nil {
		printJSON(v.StructuredContent)
	}
	if v.AccessInfo != nil && *v.AccessInfo != "" {
		fmt.Fprintf(os.Stderr, "\nAccess: %s\n", *v.AccessInfo)
	}
}

func printSlidesContentPlain(v *api.SlidesContentResponse) {
	if v.PlainText != nil {
		fmt.Print(*v.PlainText)
		if !strings.HasSuffix(*v.PlainText, "\n") {
			fmt.Println()
		}
	} else if v.StructuredContent != nil {
		printJSON(v.StructuredContent)
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

// ==================== TASKS FORMATTERS ====================

func taskProviderFooter(provider *string) string {
	if provider == nil || *provider == "" {
		return ""
	}
	return fmt.Sprintf("Provider: %s\n", *provider)
}

func printTaskProvidersTable(w *tabwriter.Writer, v api.TaskProvidersResponse) {
	fmt.Fprintln(w, "ID\tCODE\tNAME")
	fmt.Fprintln(w, "──\t────\t────")
	for _, p := range v {
		fmt.Fprintf(w, "%d\t%s\t%s\n", p.TaskProviderID, p.ProviderCode, p.ProviderDisplayName)
	}
}

func printTaskProvidersPlain(v api.TaskProvidersResponse) {
	for _, p := range v {
		fmt.Printf("%d\t%s\t%s\n", p.TaskProviderID, p.ProviderCode, p.ProviderDisplayName)
	}
}

func printTaskBoardsTable(w *tabwriter.Writer, v *api.TaskBoardsResponse) {
	fmt.Fprintln(w, "ID\tNAME\tGROUPS\tCOLS\tWORKSPACE")
	fmt.Fprintln(w, "──\t────\t──────\t────\t─────────")
	for _, b := range v.Boards {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
			truncate(b.ID, 24),
			truncate(derefStr(b.Name), 35),
			len(b.Groups),
			len(b.Columns),
			truncate(derefStr(b.WorkspaceName), 20),
		)
	}
	if hasMoreBoards(v) {
		fmt.Fprintln(w, "\nMore results available — use --all to fetch all or pass --cursor/--page.")
	}
	if footer := taskProviderFooter(v.Provider); footer != "" {
		fmt.Fprint(w, "\n", footer)
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printTaskBoardsPlain(v *api.TaskBoardsResponse) {
	for _, b := range v.Boards {
		fmt.Printf("%s\t%s\t%d\t%d\t%s\n",
			b.ID, derefStr(b.Name), len(b.Groups), len(b.Columns), derefStr(b.WorkspaceName))
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func hasMoreBoards(v *api.TaskBoardsResponse) bool {
	return (v.NextCursor != nil && *v.NextCursor != "") || (v.NextPage != nil && *v.NextPage > 0)
}

func printTaskBoardDetailTable(w *tabwriter.Writer, v *api.TaskBoardResponse) {
	if v.Board == nil {
		fmt.Fprintln(w, "(no board)")
		return
	}
	b := *v.Board
	fmt.Fprintf(w, "ID:\t%s\n", b.ID)
	fmt.Fprintf(w, "Name:\t%s\n", derefStr(b.Name))
	if b.Description != nil && *b.Description != "" {
		fmt.Fprintf(w, "Description:\t%s\n", truncate(*b.Description, 120))
	}
	if b.WorkspaceName != nil {
		fmt.Fprintf(w, "Workspace:\t%s\n", *b.WorkspaceName)
	}
	if b.FolderName != nil {
		fmt.Fprintf(w, "Folder:\t%s\n", *b.FolderName)
	}
	if len(b.Groups) > 0 {
		fmt.Fprintln(w, "\nGROUP ID\tTITLE\tCOLOR")
		fmt.Fprintln(w, "────────\t─────\t─────")
		for _, g := range b.Groups {
			fmt.Fprintf(w, "%s\t%s\t%s\n", truncate(g.ID, 30), g.Title, derefStr(g.Color))
		}
	}
	if len(b.Columns) > 0 {
		fmt.Fprintln(w, "\nCOL ID\tTITLE\tTYPE")
		fmt.Fprintln(w, "──────\t─────\t────")
		for _, c := range b.Columns {
			fmt.Fprintf(w, "%s\t%s\t%s\n", truncate(c.ID, 30), c.Title, c.Type)
		}
	}
	if footer := taskProviderFooter(v.Provider); footer != "" {
		fmt.Fprint(w, "\n", footer)
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printTaskBoardDetailPlain(v *api.TaskBoardResponse) {
	if v.Board == nil {
		return
	}
	b := *v.Board
	fmt.Printf("ID: %s\n", b.ID)
	fmt.Printf("Name: %s\n", derefStr(b.Name))
	if b.WorkspaceName != nil {
		fmt.Printf("Workspace: %s\n", *b.WorkspaceName)
	}
	for _, g := range b.Groups {
		fmt.Printf("group\t%s\t%s\n", g.ID, g.Title)
	}
	for _, c := range b.Columns {
		fmt.Printf("col\t%s\t%s\t%s\n", c.ID, c.Title, c.Type)
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func taskItemDue(item api.TaskItemDto) string {
	if item.DueDate == nil {
		return ""
	}
	// dueDate is ISO 8601; show the date portion when present.
	s := *item.DueDate
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func printTaskItemsTable(w *tabwriter.Writer, v *api.TaskItemsResponse) {
	fmt.Fprintln(w, "ID\tSTATUS\tNAME\tASSIGNEES\tDUE\tPRIORITY")
	fmt.Fprintln(w, "──\t──────\t────\t─────────\t───\t────────")
	for _, item := range v.Items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(item.ID, 24),
			derefStr(item.Status),
			truncate(derefStr(item.Name), 40),
			truncate(strings.Join(item.Assignees, ", "), 24),
			taskItemDue(item),
			derefStr(item.Priority),
		)
	}
	if len(v.Items) > 0 && hasMoreItems(v) {
		fmt.Fprintln(w, "\nMore results available — use --all to fetch all or pass --cursor/--page.")
	}
	if v.TotalCount != nil {
		fmt.Fprintf(w, "\nTotal: %d (pre-rule-filter)\n", *v.TotalCount)
	}
	if footer := taskProviderFooter(v.Provider); footer != "" {
		fmt.Fprint(w, "\n", footer)
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printTaskItemsPlain(v *api.TaskItemsResponse) {
	for _, item := range v.Items {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			derefStr(item.Status),
			derefStr(item.Name),
			strings.Join(item.Assignees, ","),
			taskItemDue(item),
			derefStr(item.Priority),
		)
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func hasMoreItems(v *api.TaskItemsResponse) bool {
	return (v.NextCursor != nil && *v.NextCursor != "") || (v.NextPage != nil && *v.NextPage > 0)
}

func printTaskItemDetailTable(w *tabwriter.Writer, item api.TaskItemDto) {
	fmt.Fprintf(w, "ID:\t%s\n", item.ID)
	fmt.Fprintf(w, "Name:\t%s\n", derefStr(item.Name))
	if item.GroupName != nil {
		fmt.Fprintf(w, "Group:\t%s\n", *item.GroupName)
	} else if item.GroupID != nil {
		fmt.Fprintf(w, "Group:\t%s\n", *item.GroupID)
	}
	if item.Status != nil {
		fmt.Fprintf(w, "Status:\t%s\n", *item.Status)
	}
	if item.Priority != nil {
		fmt.Fprintf(w, "Priority:\t%s\n", *item.Priority)
	}
	if item.DueDate != nil {
		fmt.Fprintf(w, "Due:\t%s\n", *item.DueDate)
	}
	if len(item.Assignees) > 0 {
		fmt.Fprintf(w, "Assignees:\t%s\n", strings.Join(item.Assignees, ", "))
	}
	if len(item.Labels) > 0 {
		fmt.Fprintf(w, "Labels:\t%s\n", strings.Join(item.Labels, ", "))
	}
	if item.Description != nil && *item.Description != "" {
		fmt.Fprintf(w, "Description:\t%s\n", *item.Description)
	}
	if len(item.ColumnValues) > 0 {
		fmt.Fprintln(w, "\nCOLUMN\tTYPE\tVALUE")
		fmt.Fprintln(w, "──────\t────\t─────")
		for _, cv := range item.ColumnValues {
			fmt.Fprintf(w, "%s\t%s\t%s\n", cv.ColumnTitle, cv.Type, truncate(derefStr(cv.Text), 60))
		}
	}
	if len(item.SubItems) > 0 {
		fmt.Fprintf(w, "\nSubItems:\t%d\n", len(item.SubItems))
	}
}

func printTaskItemDetailPlain(item api.TaskItemDto) {
	fmt.Printf("ID: %s\n", item.ID)
	fmt.Printf("Name: %s\n", derefStr(item.Name))
	if item.Status != nil {
		fmt.Printf("Status: %s\n", *item.Status)
	}
	if len(item.Assignees) > 0 {
		fmt.Printf("Assignees: %s\n", strings.Join(item.Assignees, ", "))
	}
	if item.DueDate != nil {
		fmt.Printf("Due: %s\n", *item.DueDate)
	}
	if item.Description != nil && *item.Description != "" {
		fmt.Printf("\n%s\n", *item.Description)
	}
}

func printTaskCommentsTable(w *tabwriter.Writer, v *api.TaskCommentsResponse) {
	fmt.Fprintln(w, "ID\tAUTHOR\tCREATED\tBODY")
	fmt.Fprintln(w, "──\t──────\t───────\t────")
	for _, c := range v.Comments {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			truncate(c.ID, 20),
			truncate(derefStr(c.AuthorName), 24),
			derefStr(c.CreatedAt),
			truncate(derefStr(c.Body), 60),
		)
	}
	if footer := taskProviderFooter(v.Provider); footer != "" {
		fmt.Fprint(w, "\n", footer)
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printTaskCommentsPlain(v *api.TaskCommentsResponse) {
	for _, c := range v.Comments {
		fmt.Printf("%s\t%s\t%s\t%s\n",
			c.ID, derefStr(c.AuthorName), derefStr(c.CreatedAt), derefStr(c.Body))
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func printTaskSearchTable(w *tabwriter.Writer, v *api.TaskSearchResponse) {
	fmt.Fprintf(w, "Query:\t%s\n", v.Query)
	fmt.Fprintf(w, "Boards searched:\t%d", v.BoardsSearched)
	if v.BoardsFailed > 0 {
		fmt.Fprintf(w, "  (failed: %d)", v.BoardsFailed)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Results:\t%d\n\n", v.TotalResults)

	fmt.Fprintln(w, "BOARD\tITEM_ID\tSTATUS\tNAME")
	fmt.Fprintln(w, "─────\t───────\t──────\t────")
	for _, r := range v.Results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			truncate(derefStr(r.BoardName), 24),
			truncate(r.Item.ID, 24),
			derefStr(r.Item.Status),
			truncate(derefStr(r.Item.Name), 40),
		)
	}
	if footer := taskProviderFooter(v.Provider); footer != "" {
		fmt.Fprint(w, "\n", footer)
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printTaskSearchPlain(v *api.TaskSearchResponse) {
	for _, r := range v.Results {
		fmt.Printf("%s\t%s\t%s\t%s\n",
			derefStr(r.BoardName), r.Item.ID, derefStr(r.Item.Status), derefStr(r.Item.Name))
	}
	if v.BoardsFailed > 0 {
		fmt.Fprintf(os.Stderr, "Warning: %d boards failed upstream — results are partial.\n", v.BoardsFailed)
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func printTaskBlockListTable(w *tabwriter.Writer, v *api.TaskBlockListResponse) {
	fmt.Fprintln(w, "TYPE\tCHILDREN\tTEXT")
	fmt.Fprintln(w, "────\t────────\t────")
	for _, b := range v.Blocks {
		children := ""
		if b.HasChildren {
			children = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", b.Type, children, truncate(derefStr(b.Text), 80))
	}
	if v.HasMore {
		fmt.Fprintln(w, "\nMore blocks available — use --all to fetch all or pass --cursor.")
	}
	printDriveAccessWarningsTable(w, v.AccessInfo, nil)
}

func printTaskBlockListPlain(v *api.TaskBlockListResponse) {
	for _, b := range v.Blocks {
		fmt.Printf("%s\t%v\t%s\n", b.Type, b.HasChildren, derefStr(b.Text))
	}
	printDriveAccessWarnings(v.AccessInfo, nil)
}

func printTaskItemResult(v *api.TaskItemResult) {
	if v.Success {
		if v.ItemID != nil && *v.ItemID != "" {
			fmt.Printf("OK  (item: %s)\n", *v.ItemID)
		} else {
			fmt.Println("OK")
		}
		if len(v.RejectedFields) > 0 {
			fmt.Fprintf(os.Stderr, ColorYellow("Warning: rejected fields: %s\n"), strings.Join(v.RejectedFields, ", "))
		}
		return
	}
	msg := derefStr(v.ErrorMessage)
	if msg == "" {
		msg = derefStr(v.ErrorCode)
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}

func printTaskOperationResult(v *api.TaskOperationResult) {
	if v.Success {
		fmt.Println("OK")
		return
	}
	msg := derefStr(v.ErrorMessage)
	if msg == "" {
		msg = derefStr(v.ErrorCode)
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}

func printTaskCommentResult(v *api.TaskCommentResult) {
	if v.Success {
		if v.CommentID != nil && *v.CommentID != "" {
			fmt.Printf("OK  (comment: %s)\n", *v.CommentID)
		} else {
			fmt.Println("OK")
		}
		return
	}
	msg := derefStr(v.ErrorMessage)
	if msg == "" {
		msg = derefStr(v.ErrorCode)
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}

func printAppendBlocksResponse(v *api.AppendBlocksResponse) {
	if v.Success {
		fmt.Printf("OK  (%d blocks appended)\n", v.BlocksAppended)
		return
	}
	msg := derefStr(v.Error)
	if msg == "" {
		msg = derefStr(v.ErrorCode)
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}
