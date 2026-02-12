package api

import "time"

// Meta contains response metadata
type Meta struct {
	Count       int       `json:"count,omitempty"`
	Offset      int       `json:"offset,omitempty"`
	HasMore     bool      `json:"hasMore,omitempty"`
	TotalCount  int       `json:"totalCount,omitempty"`
	Truncated   bool      `json:"truncated,omitempty"`
	ExecutionMs int       `json:"execution_ms,omitempty"`
	From        time.Time `json:"from,omitempty"`
	To          time.Time `json:"to,omitempty"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
}

// EventsResponse is the response type for calendar events
type EventsResponse struct {
	RequestID                string  `json:"request_id,omitempty"`
	Events                   []Event `json:"events"`
	Meta                     *Meta   `json:"meta,omitempty"`
	AccessInfo               string  `json:"accessInfo,omitempty"`
	CurrentUserCalendarEmail string  `json:"currentUserCalendarEmail,omitempty"`
}

// SingleEventResponse is the response type for a single event GET
type SingleEventResponse struct {
	Event                    Event  `json:"event"`
	AccessInfo               string `json:"accessInfo,omitempty"`
	CurrentUserCalendarEmail string `json:"currentUserCalendarEmail,omitempty"`
}

// CalendarsResponse is the response type for calendars
type CalendarsResponse struct {
	Data       []Calendar `json:"data"`
	AccessInfo string     `json:"accessInfo,omitempty"`
}

// AuthStatusResponse is the response for auth status endpoint
type AuthStatusResponse struct {
	Email        string    `json:"email"`
	OperatorName string    `json:"operatorName"`
	KeyID        int       `json:"keyId"`
	KeyTitle     string    `json:"keyTitle,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Event represents a calendar event
type Event struct {
	ID               string     `json:"id"`
	CalendarID       int64      `json:"calendarId,omitempty"`
	Title            string     `json:"title"`
	Summary          string     `json:"summary,omitempty"` // Alias for backwards compat
	Description      string     `json:"description,omitempty"`
	Location         string     `json:"location,omitempty"`
	StartUtc         time.Time  `json:"startUtc"`
	EndUtc           time.Time  `json:"endUtc"`
	StartLocal       string     `json:"startLocal,omitempty"`
	EndLocal         string     `json:"endLocal,omitempty"`
	DurationMinutes  int        `json:"durationMinutes,omitempty"`
	Status           string     `json:"status"`
	AllDay           bool       `json:"allDay"`
	IsAllDay         bool       `json:"isAllDay,omitempty"` // Alias for backwards compat
	Attendees        []Attendee `json:"attendees,omitempty"`
	Organizer        string     `json:"organizer,omitempty"`
	JoinUrl          string     `json:"joinUrl,omitempty"`
	Labels           []string   `json:"labels,omitempty"`
	IsRecurringEvent bool       `json:"isRecurringEvent,omitempty"`
}

// Attendee represents an event attendee
type Attendee struct {
	Email          string `json:"email"`
	Name           string `json:"name,omitempty"`
	DisplayName    string `json:"displayName,omitempty"` // Alias
	Response       string `json:"response,omitempty"`
	ResponseStatus string `json:"responseStatus,omitempty"` // Alias
}

// Calendar represents a calendar
type Calendar struct {
	ID              int64     `json:"id"`
	ExternalID      string    `json:"externalId,omitempty"`
	Name            string    `json:"name"`
	Provider        string    `json:"provider"`
	Timezone        string    `json:"timezone,omitempty"`
	IsPrimary       bool      `json:"isPrimary"`
	IsOperatorOwner bool      `json:"isOperatorOwner,omitempty"`
	OwnerEmail      string    `json:"ownerEmail,omitempty"`
	LastSyncedAt    time.Time `json:"lastSyncedAt,omitempty"`
}

// EventParams holds parameters for event queries
type EventParams struct {
	From             time.Time
	To               time.Time
	CalendarID       int64
	Limit            int
	Offset           int
	Query            string // keyword search (q parameter)
	Attendees        string // comma-separated attendee emails
	IncludeCancelled bool
}

// CreateEventRequest represents a request to create an event
type CreateEventRequest struct {
	CalendarID  int64     `json:"calendarId"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	IsAllDay    bool      `json:"isAllDay,omitempty"`
	Attendees   []string  `json:"attendees,omitempty"`
	Recurrence  []string  `json:"recurrence,omitempty"`
}

// UpdateEventRequest represents a request to update an event (PATCH)
type UpdateEventRequest struct {
	Summary           string     `json:"summary,omitempty"`
	Description       string     `json:"description,omitempty"`
	Location          string     `json:"location,omitempty"`
	From              *time.Time `json:"from,omitempty"`
	To                *time.Time `json:"to,omitempty"`
	IsAllDay          *bool      `json:"isAllDay,omitempty"`
	AddAttendees      []string   `json:"addAttendees,omitempty"`
	RemoveAttendees   []string   `json:"removeAttendees,omitempty"`
	SendNotifications *bool      `json:"sendNotifications,omitempty"`
}

// EventsByContactParams holds parameters for events by-contact queries
type EventsByContactParams struct {
	Email  string // Partial email matching (case-insensitive)
	Name   string // Partial name/display name matching (case-insensitive)
	Limit  int
	Offset int
}

// FreeBusyResponse is the response type for free/busy queries
type FreeBusyResponse struct {
	Calendars  []FreeBusyCalendar `json:"calendars"`
	AccessInfo string             `json:"accessInfo,omitempty"`
}

// FreeBusyCalendar represents free/busy info for a single calendar
type FreeBusyCalendar struct {
	CalendarID   int64        `json:"calendarId"`
	CalendarName string       `json:"calendarName"`
	Busy         []BusyPeriod `json:"busy"`
}

// BusyPeriod represents a single busy time block
type BusyPeriod struct {
	StartUtc        time.Time `json:"startUtc"`
	EndUtc          time.Time `json:"endUtc"`
	DurationMinutes int       `json:"durationMinutes"`
}

// FreeBusyParams holds parameters for free/busy queries
type FreeBusyParams struct {
	From      time.Time
	To        time.Time
	Calendars string // comma-separated calendar IDs
}

// DeleteEventResponse is the response from deleting an event
type DeleteEventResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ==================== EMAIL TYPES ====================

// EmailsResponse is the response type for email list/search operations
type EmailsResponse struct {
	Emails        []Email `json:"emails"`
	TotalCount    int     `json:"totalCount,omitempty"`
	HasMore       bool    `json:"hasMore,omitempty"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	AccessInfo    string  `json:"accessInfo,omitempty"`
}

// SingleEmailResponse wraps a single email with access info
type SingleEmailResponse struct {
	Email      Email  `json:"email"`
	AccessInfo string `json:"accessInfo,omitempty"`
}

// Email represents an email message
type Email struct {
	ID             string        `json:"id"`
	ThreadID       string        `json:"threadId,omitempty"`
	Subject        string        `json:"subject,omitempty"`
	From           *Participant  `json:"from,omitempty"`
	To             []Participant `json:"to,omitempty"`
	CC             []Participant `json:"cc,omitempty"`
	BCC            []Participant `json:"bcc,omitempty"`
	BodyPreview    string        `json:"bodyPreview,omitempty"`
	Body           string        `json:"body,omitempty"`
	BodyType       string        `json:"bodyType,omitempty"`
	SentAt         time.Time     `json:"sentAt,omitempty"`
	ReceivedAt     time.Time     `json:"receivedAt,omitempty"`
	IsRead         bool          `json:"isRead"`
	HasAttachments bool          `json:"hasAttachments"`
	Attachments    []Attachment  `json:"attachments,omitempty"`
	Labels         []string      `json:"labels,omitempty"`
	Importance     string        `json:"importance,omitempty"`
	Provider       string        `json:"provider"`
}

// Participant represents an email participant (sender/recipient)
type Participant struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Attachment represents an email attachment
type Attachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType,omitempty"`
	Size        int64  `json:"size"`
	IsInline    bool   `json:"isInline"`
}

// ThreadResponse is the response type for GET /threads/{id}
type ThreadResponse struct {
	ID            string        `json:"id"`
	Subject       string        `json:"subject,omitempty"`
	Messages      []Email       `json:"messages"`
	MessageCount  int           `json:"messageCount"`
	Participants  []Participant `json:"participants,omitempty"`
	LastMessageAt time.Time     `json:"lastMessageAt,omitempty"`
	Provider      string        `json:"provider"`
	AccessInfo    string        `json:"accessInfo,omitempty"`
}

// EmailParams holds parameters for email list/search queries
type EmailParams struct {
	Query         string
	From          string
	To            string
	Subject       string
	Label         string
	Unread        *bool
	After         time.Time
	Before        time.Time
	HasAttachment *bool
	Limit         int
	IncludeBody   bool
	PageToken     string
}

// SendEmailRequest represents a request to send a new email
type SendEmailRequest struct {
	To           []Participant `json:"to"`
	CC           []Participant `json:"cc,omitempty"`
	BCC          []Participant `json:"bcc,omitempty"`
	Subject      string        `json:"subject"`
	Body         string        `json:"body"`
	BodyType     string        `json:"bodyType,omitempty"`
	Importance   string        `json:"importance,omitempty"`
	ConnectionID *int64        `json:"connectionId,omitempty"`
}

// ReplyEmailRequest represents a request to reply to an email
type ReplyEmailRequest struct {
	Body     string `json:"body"`
	BodyType string `json:"bodyType,omitempty"`
	ReplyAll bool   `json:"replyAll,omitempty"`
}

// ForwardEmailRequest represents a request to forward an email
type ForwardEmailRequest struct {
	To       []Participant `json:"to"`
	CC       []Participant `json:"cc,omitempty"`
	Body     string        `json:"body,omitempty"`
	BodyType string        `json:"bodyType,omitempty"`
}

// ModifyEmailRequest represents a request to modify email properties
type ModifyEmailRequest struct {
	MarkAsRead   *bool    `json:"markAsRead,omitempty"`
	AddLabels    []string `json:"addLabels,omitempty"`
	RemoveLabels []string `json:"removeLabels,omitempty"`
}

// EmailActionResponse is the response for send/reply/forward operations
type EmailActionResponse struct {
	Success      bool   `json:"success"`
	EmailID      string `json:"emailId,omitempty"`
	ThreadID     string `json:"threadId,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}
