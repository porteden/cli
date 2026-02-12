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
	// Truncate description if too long
	if opts.MaxDescriptionLength > 0 && len(event.Description) > opts.MaxDescriptionLength {
		event.Description = event.Description[:opts.MaxDescriptionLength-3] + "..."
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
		TotalCount:    resp.TotalCount,
		HasMore:       resp.HasMore,
		NextPageToken: resp.NextPageToken,
		AccessInfo:    resp.AccessInfo,
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
	if opts.MaxDescriptionLength > 0 && len(email.BodyPreview) > opts.MaxDescriptionLength {
		email.BodyPreview = email.BodyPreview[:opts.MaxDescriptionLength-3] + "..."
	}

	if opts.MaxDescriptionLength > 0 && len(email.Body) > opts.MaxDescriptionLength*2 {
		email.Body = email.Body[:opts.MaxDescriptionLength*2-3] + "..."
	}

	// Strip attachment details in compact mode (keep HasAttachments flag)
	email.Attachments = nil

	// Limit labels
	if len(email.Labels) > 3 {
		email.Labels = email.Labels[:3]
	}

	return email
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
