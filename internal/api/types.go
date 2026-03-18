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

// ==================== DRIVE TYPES ====================

// DriveUser represents a file owner or collaborator
type DriveUser struct {
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName,omitempty"`
	Role        *string `json:"role,omitempty"`
}

// DriveFile represents a file or folder in Google Drive
type DriveFile struct {
	ID               string            `json:"id"`
	Name             *string           `json:"name,omitempty"`
	MimeType         *string           `json:"mimeType,omitempty"`
	Size             *int64            `json:"size,omitempty"`
	CreatedTime      *string           `json:"createdTime,omitempty"`
	ModifiedTime     *string           `json:"modifiedTime,omitempty"`
	Owners           []DriveUser       `json:"owners,omitempty"`
	SharedWith       []DriveUser       `json:"sharedWith,omitempty"`
	Description      *string           `json:"description,omitempty"`
	WebViewLink      *string           `json:"webViewLink,omitempty"`
	DownloadLink     *string           `json:"downloadLink,omitempty"`
	ParentFolderID   *string           `json:"parentFolderId,omitempty"`
	ParentFolderName *string           `json:"parentFolderName,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	IsFolder         bool              `json:"isFolder"`
	Provider         string            `json:"provider"`
}

// DriveFilesResponse is the response for drive file list/search
type DriveFilesResponse struct {
	Files         []DriveFile `json:"files"`
	NextPageToken *string     `json:"nextPageToken,omitempty"`
	HasMore       bool        `json:"hasMore,omitempty"`
	AccessInfo    *string     `json:"accessInfo,omitempty"`
	AuthWarnings  []string    `json:"authWarnings,omitempty"`
}

// SingleDriveFileResponse wraps a single file with access info
type SingleDriveFileResponse struct {
	File       *DriveFile `json:"file"`
	AccessInfo *string    `json:"accessInfo,omitempty"`
}

// DrivePermission represents a single sharing permission (ACL entry)
type DrivePermission struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Role         string  `json:"role"`
	EmailAddress *string `json:"emailAddress,omitempty"`
	Domain       *string `json:"domain,omitempty"`
	DisplayName  *string `json:"displayName,omitempty"`
}

// DrivePermissionsResponse is the response for file permissions
type DrivePermissionsResponse struct {
	Permissions []DrivePermission `json:"permissions"`
	AccessInfo  *string           `json:"accessInfo,omitempty"`
}

// DriveFileLinkResponse is the response for file download/export links
type DriveFileLinkResponse struct {
	Success               bool              `json:"success"`
	WebViewLink           *string           `json:"webViewLink,omitempty"`
	DownloadUrl           *string           `json:"downloadUrl,omitempty"`
	ExportLinks           map[string]string `json:"exportLinks,omitempty"`
	FileName              *string           `json:"fileName,omitempty"`
	MimeType              *string           `json:"mimeType,omitempty"`
	Size                  *int64            `json:"size,omitempty"`
	IsGoogleWorkspaceFile bool              `json:"isGoogleWorkspaceFile"`
	ErrorMessage          *string           `json:"errorMessage,omitempty"`
	ErrorCode             *string           `json:"errorCode,omitempty"`
}

// DriveOperationResult is the response for mutating drive operations
type DriveOperationResult struct {
	Success      bool    `json:"success"`
	FileID       *string `json:"fileId,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	ErrorCode    *string `json:"errorCode,omitempty"`
}

// DriveListParams holds parameters for drive file list/search queries
type DriveListParams struct {
	Q              string
	FolderID       string
	MimeType       string
	Name           string
	TrashedOnly    bool
	SharedWithMe   bool
	ModifiedAfter  string
	ModifiedBefore string
	Limit          int
	PageToken      string
	OrderBy        string
}

// CreateFolderRequest represents a request to create a new folder
type CreateFolderRequest struct {
	Name           string  `json:"name"`
	ParentFolderID *string `json:"parentFolderId,omitempty"`
	Description    *string `json:"description,omitempty"`
}

// RenameFileRequest represents a request to rename a file or folder
type RenameFileRequest struct {
	NewName string `json:"newName"`
}

// MoveFileRequest represents a request to move a file to another folder
type MoveFileRequest struct {
	DestinationFolderID string `json:"destinationFolderId"`
}

// ShareFileRequest represents a request to share a file
type ShareFileRequest struct {
	Type             string  `json:"type"`
	Role             string  `json:"role"`
	EmailAddress     *string `json:"emailAddress,omitempty"`
	Domain           *string `json:"domain,omitempty"`
	SendNotification *bool   `json:"sendNotification,omitempty"`
	Message          *string `json:"message,omitempty"`
}

// ==================== DOCS TYPES ====================

// DocContentResponse is the response for reading a Google Doc
type DocContentResponse struct {
	PlainText         *string     `json:"plainText,omitempty"`
	StructuredContent interface{} `json:"structuredContent,omitempty"`
	Title             *string     `json:"title,omitempty"`
	AccessInfo        *string     `json:"accessInfo,omitempty"`
}

// DocEditOperation represents a single text editing operation on a Google Doc
type DocEditOperation struct {
	Type      string  `json:"type"`
	Text      *string `json:"text,omitempty"`
	Index     *int    `json:"index,omitempty"`
	Find      *string `json:"find,omitempty"`
	Replace   *string `json:"replace,omitempty"`
	MatchCase *bool   `json:"matchCase,omitempty"`
}

// EditDocRequest represents a request to edit a Google Doc
type EditDocRequest struct {
	Operations []DocEditOperation `json:"operations"`
}

// ==================== SHEETS TYPES ====================

// SheetTabInfo represents metadata for a single sheet tab
type SheetTabInfo struct {
	SheetID     int    `json:"sheetId"`
	Title       string `json:"title"`
	RowCount    int    `json:"rowCount"`
	ColumnCount int    `json:"columnCount"`
}

// SheetMetadataResponse is the response for spreadsheet metadata
type SheetMetadataResponse struct {
	SpreadsheetID string         `json:"spreadsheetId"`
	Title         *string        `json:"title,omitempty"`
	Sheets        []SheetTabInfo `json:"sheets"`
	AccessInfo    *string        `json:"accessInfo,omitempty"`
}

// SheetValuesResponse is the response for reading sheet cell values
type SheetValuesResponse struct {
	Range      string          `json:"range"`
	Values     [][]interface{} `json:"values"`
	AccessInfo *string         `json:"accessInfo,omitempty"`
}

// WriteSheetValuesRequest represents a request to write cell values
type WriteSheetValuesRequest struct {
	Range            string          `json:"range"`
	Values           [][]interface{} `json:"values"`
	ValueInputOption string          `json:"valueInputOption,omitempty"`
}

// AppendSheetRowsRequest represents a request to append rows to a sheet
type AppendSheetRowsRequest struct {
	Range            string          `json:"range"`
	Values           [][]interface{} `json:"values"`
	ValueInputOption string          `json:"valueInputOption,omitempty"`
}
