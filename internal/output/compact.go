package output

import (
	"strings"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/debug"
)

// CompactOptions configures compact mode transformations
type CompactOptions struct {
	MaxDescriptionLength int  // default: 100
	FilterAttendees      bool // default: true
	MaxAttendees         int  // default: 10 (0 = unlimited)
}

// DefaultCompactOptions returns the default compact mode settings
func DefaultCompactOptions() CompactOptions {
	return CompactOptions{
		MaxDescriptionLength: 100,
		FilterAttendees:      true,
		MaxAttendees:         10,
	}
}

// CompactEventsResponse applies compact transformations to an events response
func CompactEventsResponse(resp *api.EventsResponse, opts CompactOptions) *api.EventsResponse {
	if resp == nil {
		return nil
	}

	// Create a copy to avoid mutating the original
	compacted := &api.EventsResponse{
		RequestID:                "", // Omit request_id in compact mode
		Events:                   make([]api.Event, len(resp.Events)),
		Meta:                     resp.Meta,
		AccessInfo:               resp.AccessInfo,
		CurrentUserCalendarEmail: resp.CurrentUserCalendarEmail,
	}

	for i, event := range resp.Events {
		compacted.Events[i] = compactEvent(event, opts)
	}

	return compacted
}

// CompactEvent applies compact transformations to a single event
func CompactEvent(event *api.Event, opts CompactOptions) *api.Event {
	if event == nil {
		return nil
	}
	compacted := compactEvent(*event, opts)
	return &compacted
}

func compactEvent(event api.Event, opts CompactOptions) api.Event {
	// Truncate description if too long (rune-safe: this feeds JSON output)
	if opts.MaxDescriptionLength > 0 {
		event.Description = truncateRunesSafe(event.Description, opts.MaxDescriptionLength)
	}

	// Filter invalid attendees
	if opts.FilterAttendees && len(event.Attendees) > 0 {
		event.Attendees = filterAttendees(event.Attendees)
	}

	// Limit number of attendees
	if opts.MaxAttendees > 0 && len(event.Attendees) > opts.MaxAttendees {
		overflow := len(event.Attendees) - opts.MaxAttendees
		if debug.Verbose {
			debug.Log("Compact mode limited attendees: showing %d of %d (+%d more)",
				opts.MaxAttendees, len(event.Attendees), overflow)
		}
		event.Attendees = event.Attendees[:opts.MaxAttendees]
	}

	return event
}

// filterAttendees removes attendees that don't have valid email addresses
// (e.g., numeric IDs that sometimes appear in API responses)
func filterAttendees(attendees []api.Attendee) []api.Attendee {
	var filtered []api.Attendee
	var removed []string

	for _, a := range attendees {
		if isValidEmail(a.Email) {
			filtered = append(filtered, a)
		} else {
			removed = append(removed, a.Email)
		}
	}

	// Log filtered attendees in verbose mode for debugging
	if debug.Verbose && len(removed) > 0 {
		debug.Log("Compact mode filtered %d invalid attendee(s): %v", len(removed), removed)
	}

	return filtered
}

// isValidEmail performs a basic check for email-like strings
// Returns false for numeric-only strings or strings without @
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	// Must contain @ to be considered an email
	if !strings.Contains(email, "@") {
		return false
	}
	// Must have something before and after @
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	return true
}

// CompactEmailsResponse applies compact transformations to an emails response
func CompactEmailsResponse(resp *api.EmailsResponse, opts CompactOptions) *api.EmailsResponse {
	if resp == nil {
		return nil
	}

	compacted := &api.EmailsResponse{
		Emails:        make([]api.Email, len(resp.Emails)),
		HasMore:       resp.HasMore,
		NextPageToken: resp.NextPageToken,
		AccessInfo:    resp.AccessInfo,
		AuthWarnings:  resp.AuthWarnings,
	}

	for i, email := range resp.Emails {
		compacted.Emails[i] = compactEmailMsg(email, opts)
	}

	return compacted
}

// CompactEmail applies compact transformations to a single email
func CompactEmail(email *api.Email, opts CompactOptions) *api.Email {
	if email == nil {
		return nil
	}
	compacted := compactEmailMsg(*email, opts)
	return &compacted
}

func compactEmailMsg(email api.Email, opts CompactOptions) api.Email {
	if opts.MaxDescriptionLength > 0 {
		email.BodyPreview = truncateRunesSafe(email.BodyPreview, opts.MaxDescriptionLength)
		email.Body = truncateRunesSafe(email.Body, opts.MaxDescriptionLength*2)
	}

	// Strip attachment details in compact mode (keep HasAttachments flag)
	email.Attachments = nil

	// Limit labels
	if len(email.Labels) > 3 {
		email.Labels = email.Labels[:3]
	}

	return email
}

// CompactDriveFilesResponse applies compact transformations to a drive files response
func CompactDriveFilesResponse(resp *api.DriveFilesResponse, opts CompactOptions) *api.DriveFilesResponse {
	if resp == nil {
		return nil
	}

	compacted := &api.DriveFilesResponse{
		Files:         make([]api.DriveFile, len(resp.Files)),
		NextPageToken: resp.NextPageToken,
		HasMore:       resp.HasMore,
		AccessInfo:    resp.AccessInfo,
		AuthWarnings:  resp.AuthWarnings,
	}

	for i, f := range resp.Files {
		cf := f
		// Truncate name (rune-safe)
		if cf.Name != nil {
			s := truncateRunesSafe(*cf.Name, 40)
			cf.Name = &s
		}
		// First owner only
		if len(cf.Owners) > 1 {
			cf.Owners = cf.Owners[:1]
		}
		// Strip noisy fields
		cf.SharedWith = nil
		cf.Labels = nil
		cf.Description = nil
		cf.DownloadLink = nil
		compacted.Files[i] = cf
	}

	return compacted
}

// CompactSheetBulkContentResponse trims per-tab values to a reasonable head
// in compact mode so an LLM agent doesn't drown in untouched ranges. The
// Clipped/FullRange hints from the server are preserved so the agent still
// knows where to drill in.
func CompactSheetBulkContentResponse(resp *api.SheetBulkContentResponse, opts CompactOptions) *api.SheetBulkContentResponse {
	if resp == nil {
		return nil
	}

	const maxRowsPerTab = 25
	tabs := make([]api.SheetBulkContentTab, len(resp.Sheets))
	for i, t := range resp.Sheets {
		ct := t
		if len(ct.Values) > maxRowsPerTab {
			ct.Values = ct.Values[:maxRowsPerTab]
			ct.Clipped = true
			// If the server didn't already provide a drill-in range (i.e. the
			// tab fit upstream but we just trimmed it for the agent), seed
			// FullRange from the returned range so the agent has a concrete
			// re-read target. Without this, "clipped" with no FullRange would
			// be a dead end.
			if ct.FullRange == nil {
				r := ct.Range
				ct.FullRange = &r
			}
		}
		tabs[i] = ct
	}

	return &api.SheetBulkContentResponse{
		SpreadsheetID: resp.SpreadsheetID,
		Title:         resp.Title,
		Sheets:        tabs,
		AccessInfo:    resp.AccessInfo,
	}
}

// ==================== TASKS COMPACT ====================

const (
	taskMaxAssignees    = 10
	taskMaxLabels       = 10
	taskMaxColumnValues = 10
)

// compactTaskItem trims a single item in place (value receiver — Go pass-by-value).
// Structural fields (ID, GroupID, GroupName) are never touched — they're the
// caller's only way to address the item once other fields are masked.
func compactTaskItem(item api.TaskItemDto, opts CompactOptions) api.TaskItemDto {
	if item.Description != nil && opts.MaxDescriptionLength > 0 {
		s := truncateRunesSafe(*item.Description, opts.MaxDescriptionLength)
		item.Description = &s
	}
	if len(item.Assignees) > taskMaxAssignees {
		item.Assignees = item.Assignees[:taskMaxAssignees]
	}
	if len(item.Labels) > taskMaxLabels {
		item.Labels = item.Labels[:taskMaxLabels]
	}
	if len(item.ColumnValues) > taskMaxColumnValues {
		item.ColumnValues = item.ColumnValues[:taskMaxColumnValues]
	}
	// Drop embedded comments — they're available via `tasks comments <itemId>`.
	item.Comments = nil

	// Recurse one level into SubItems; deeper hierarchies are rare and
	// dropping them keeps payloads bounded.
	if len(item.SubItems) > 0 {
		subs := make([]api.TaskItemDto, len(item.SubItems))
		for i, sub := range item.SubItems {
			sub.SubItems = nil
			subs[i] = compactTaskItem(sub, opts)
		}
		item.SubItems = subs
	}
	return item
}

// CompactTaskItem trims a single TaskItemDto. Returns the compacted item
// (or nil if input was nil).
func CompactTaskItem(item *api.TaskItemDto, opts CompactOptions) *api.TaskItemDto {
	if item == nil {
		return nil
	}
	compacted := compactTaskItem(*item, opts)
	return &compacted
}

// CompactTaskItemsResponse trims every item in a list response.
func CompactTaskItemsResponse(resp *api.TaskItemsResponse, opts CompactOptions) *api.TaskItemsResponse {
	if resp == nil {
		return nil
	}
	items := make([]api.TaskItemDto, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = compactTaskItem(item, opts)
	}
	return &api.TaskItemsResponse{
		Provider:   resp.Provider,
		Items:      items,
		NextCursor: resp.NextCursor,
		NextPage:   resp.NextPage,
		TotalCount: resp.TotalCount,
		AccessInfo: resp.AccessInfo,
	}
}

// CompactTaskItemResponse trims a single-item response.
func CompactTaskItemResponse(resp *api.TaskItemResponse, opts CompactOptions) *api.TaskItemResponse {
	if resp == nil {
		return nil
	}
	return &api.TaskItemResponse{
		Provider:   resp.Provider,
		Item:       CompactTaskItem(resp.Item, opts),
		AccessInfo: resp.AccessInfo,
	}
}

// CompactTaskSearchResponse trims each search result's embedded item.
func CompactTaskSearchResponse(resp *api.TaskSearchResponse, opts CompactOptions) *api.TaskSearchResponse {
	if resp == nil {
		return nil
	}
	results := make([]api.TaskSearchResultDto, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = api.TaskSearchResultDto{
			BoardID:   r.BoardID,
			BoardName: r.BoardName,
			Item:      compactTaskItem(r.Item, opts),
		}
	}
	return &api.TaskSearchResponse{
		Provider:       resp.Provider,
		Query:          resp.Query,
		Results:        results,
		TotalResults:   resp.TotalResults,
		BoardsSearched: resp.BoardsSearched,
		BoardsFailed:   resp.BoardsFailed,
		AccessInfo:     resp.AccessInfo,
	}
}

// CompactTaskItemResult trims the embedded Item on create/update responses.
// The error fields are left untouched.
func CompactTaskItemResult(resp *api.TaskItemResult, opts CompactOptions) *api.TaskItemResult {
	if resp == nil {
		return nil
	}
	return &api.TaskItemResult{
		Provider:       resp.Provider,
		Success:        resp.Success,
		ItemID:         resp.ItemID,
		Item:           CompactTaskItem(resp.Item, opts),
		ErrorCode:      resp.ErrorCode,
		ErrorMessage:   resp.ErrorMessage,
		RejectedFields: resp.RejectedFields,
	}
}

// CompactTaskBlockListResponse truncates each block's text. Metadata is small
// (typically a single language or checked field) and kept intact.
func CompactTaskBlockListResponse(resp *api.TaskBlockListResponse, opts CompactOptions) *api.TaskBlockListResponse {
	if resp == nil {
		return nil
	}
	blocks := make([]api.TaskBlockDto, len(resp.Blocks))
	for i, b := range resp.Blocks {
		if b.Text != nil && opts.MaxDescriptionLength > 0 {
			s := truncateRunesSafe(*b.Text, opts.MaxDescriptionLength)
			b.Text = &s
		}
		// Drop richText (HTML) in compact mode — the plain text suffices.
		b.RichText = nil
		// Drop nested children — block trees are walked via separate calls.
		b.Children = nil
		blocks[i] = b
	}
	return &api.TaskBlockListResponse{
		Blocks:     blocks,
		NextCursor: resp.NextCursor,
		HasMore:    resp.HasMore,
		AccessInfo: resp.AccessInfo,
	}
}

// CompactThreadResponse applies compact transformations to a thread response
func CompactThreadResponse(resp *api.ThreadResponse, opts CompactOptions) *api.ThreadResponse {
	if resp == nil {
		return nil
	}

	compacted := &api.ThreadResponse{
		ID:            resp.ID,
		Subject:       resp.Subject,
		Messages:      make([]api.Email, len(resp.Messages)),
		MessageCount:  resp.MessageCount,
		Participants:  resp.Participants,
		LastMessageAt: resp.LastMessageAt,
		Provider:      resp.Provider,
		AccessInfo:    resp.AccessInfo,
	}

	for i, msg := range resp.Messages {
		compacted.Messages[i] = compactEmailMsg(msg, opts)
	}

	return compacted
}
