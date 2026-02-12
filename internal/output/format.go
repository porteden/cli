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
