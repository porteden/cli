package api

import (
	"strings"
	"time"
)

// FlexTime is a time.Time that tolerates the backend's occasional
// zero-stamped naive datetimes (e.g. "0001-01-01T00:00:00" with no offset)
// on meta fields that don't have a query-bound value.
type FlexTime struct{ time.Time }

func (t *FlexTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, s); err == nil {
		t.Time = parsed
		return nil
	}
	if parsed, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		t.Time = parsed
		return nil
	}
	if parsed, err := time.Parse("2006-01-02", s); err == nil {
		t.Time = parsed
		return nil
	}
	return nil
}

// Meta contains response metadata
type Meta struct {
	Count       int      `json:"count,omitempty"`
	Offset      int      `json:"offset,omitempty"`
	HasMore     bool     `json:"hasMore,omitempty"`
	TotalCount  int      `json:"totalCount,omitempty"`
	Truncated   bool     `json:"truncated,omitempty"`
	ExecutionMs int      `json:"execution_ms,omitempty"`
	From        FlexTime `json:"from,omitempty"`
	To          FlexTime `json:"to,omitempty"`
	Timestamp   FlexTime `json:"timestamp,omitempty"`
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

// EmailsResponse is the response type for email list/search operations.
// Pagination: drive iteration via HasMore + NextPageToken — there is no totalCount
// (the firewall filters server-side and a pre-filter count would mislead).
type EmailsResponse struct {
	Emails        []Email  `json:"emails"`
	HasMore       bool     `json:"hasMoreEmailsInNextResultPage,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
	AccessInfo    string   `json:"accessInfo,omitempty"`
	AuthWarnings  []string `json:"authWarnings,omitempty"`
}

// SingleEmailResponse wraps a single email with access info
type SingleEmailResponse struct {
	Email      Email  `json:"email"`
	AccessInfo string `json:"accessInfo,omitempty"`
}

// Email represents an email message (EmailAccessDto on the wire).
// IsRead, IsOutbound, HasAttachments, Provider are structural — always present.
// EmailAccountOwner identifies which connected mailbox a result came from.
type Email struct {
	ID          string        `json:"id"`
	ThreadID    string        `json:"threadId,omitempty"`
	Subject     string        `json:"subject,omitempty"`
	From        *Participant  `json:"from,omitempty"`
	To          []Participant `json:"to,omitempty"`
	CC          []Participant `json:"cc,omitempty"`
	BCC         []Participant `json:"bcc,omitempty"`
	BodyPreview string        `json:"bodyPreview,omitempty"`
	Body        string        `json:"body,omitempty"`
	BodyType    string        `json:"bodyType,omitempty"`
	// Pointer types so a wire-omitted field (per spec contract) stays
	// omitted in our re-emission instead of decoding as Go zero-time and
	// re-emitting "0001-01-01T00:00:00Z" through -jc output.
	SentAt            *time.Time   `json:"sentAt,omitempty"`
	ReceivedAt        *time.Time   `json:"receivedAt,omitempty"`
	IsRead            bool         `json:"isRead"`
	IsOutbound        bool         `json:"isOutbound"`
	HasAttachments    bool         `json:"hasAttachments"`
	Attachments       []Attachment `json:"attachments,omitempty"`
	Labels            []string     `json:"labels,omitempty"`
	Importance        string       `json:"importance,omitempty"`
	Provider          string       `json:"provider"`
	EmailAccountOwner string       `json:"emailAccountOwner,omitempty"`
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
	LastMessageAt *time.Time    `json:"lastMessageAt,omitempty"`
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

// SendEmailRequest represents a request to send a new email.
// For multi-mailbox tokens, pin the sending mailbox via SendFrom (email address)
// or ConnectionID (integer). Omit both to use the first active connection.
type SendEmailRequest struct {
	To           []Participant `json:"to"`
	CC           []Participant `json:"cc,omitempty"`
	BCC          []Participant `json:"bcc,omitempty"`
	Subject      string        `json:"subject"`
	Body         string        `json:"body"`
	BodyType     string        `json:"bodyType,omitempty"`
	Importance   string        `json:"importance,omitempty"`
	ConnectionID *int64        `json:"connectionId,omitempty"`
	SendFrom     string        `json:"sendFrom,omitempty"`
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
// (SendEmailResult on the wire).
type EmailActionResponse struct {
	Success      bool   `json:"success"`
	EmailID      string `json:"emailId,omitempty"`
	ThreadID     string `json:"threadId,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
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
	SharedBy         *DriveUser        `json:"sharedBy,omitempty"`
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

// DriveFileContentResponse is the response for GET /files/{id}/content.
// Readability is conveyed via Readable+Reason; HTTP status is always 200.
// Reason values: BINARY_CONTENT, TOO_LARGE, EXPORT_FAILED, USE_SHEETS_ENDPOINT, USE_SLIDES_ENDPOINT.
type DriveFileContentResponse struct {
	FileID          *string `json:"fileId,omitempty"`
	Name            *string `json:"name,omitempty"`
	MimeType        *string `json:"mimeType,omitempty"`
	ContentMimeType *string `json:"contentMimeType,omitempty"`
	Content         *string `json:"content,omitempty"`
	ByteLength      *int64  `json:"byteLength,omitempty"`
	Readable        bool    `json:"readable"`
	Truncated       bool    `json:"truncated"`
	Reason          *string `json:"reason,omitempty"`
	WebViewLink     *string `json:"webViewLink,omitempty"`
	WebContentLink  *string `json:"webContentLink,omitempty"`
	AccessInfo      *string `json:"accessInfo,omitempty"`
}

// CreateDriveFileWithContentRequest represents a request to create a file
// with inline UTF-8 text content (POST /files). For Workspace target mime
// types (document/spreadsheet/presentation), Drive auto-imports the content.
type CreateDriveFileWithContentRequest struct {
	Name            string  `json:"name"`
	MimeType        string  `json:"mimeType"`
	Content         string  `json:"content"`
	ContentMimeType *string `json:"contentMimeType,omitempty"`
	FolderID        *string `json:"folderId,omitempty"`
	Description     *string `json:"description,omitempty"`
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

// SheetBulkContentTab represents one tab's content in a bulk read.
// Clipped is true when the tab has more data than was returned; in that
// case FullRange names the A1 range needed to read the rest via /values.
type SheetBulkContentTab struct {
	Title     string          `json:"title"`
	Range     string          `json:"range"`
	Values    [][]interface{} `json:"values"`
	Clipped   bool            `json:"clipped,omitempty"`
	FullRange *string         `json:"fullRange,omitempty"`
}

// SheetBulkContentResponse is the response for GET /sheets/{id}/content.
type SheetBulkContentResponse struct {
	SpreadsheetID string                `json:"spreadsheetId"`
	Title         *string               `json:"title,omitempty"`
	Sheets        []SheetBulkContentTab `json:"sheets"`
	AccessInfo    *string               `json:"accessInfo,omitempty"`
}

// ==================== SLIDES TYPES ====================

// SlideInfo represents one slide's index + title in the deck metadata.
type SlideInfo struct {
	Index int     `json:"index"`
	Title *string `json:"title,omitempty"`
}

// SlidesMetadataResponse is the response for GET /slides/{id}.
type SlidesMetadataResponse struct {
	PresentationID string      `json:"presentationId"`
	Title          *string     `json:"title,omitempty"`
	Slides         []SlideInfo `json:"slides"`
	AccessInfo     *string     `json:"accessInfo,omitempty"`
}

// SlidesContentResponse is the response for GET /slides/{id}/content.
// PlainText populated when format=text; StructuredContent populated when
// format=structured. Never both.
type SlidesContentResponse struct {
	PlainText         *string     `json:"plainText,omitempty"`
	StructuredContent interface{} `json:"structuredContent,omitempty"`
	Title             *string     `json:"title,omitempty"`
	SlideCount        int         `json:"slideCount"`
	AccessInfo        *string     `json:"accessInfo,omitempty"`
}

// ==================== TASKS TYPES ====================

// TaskConnectionInfoDto is one row in GET /tasks/providers.
type TaskConnectionInfoDto struct {
	TaskProviderID      int    `json:"taskProviderId"`
	ProviderCode        string `json:"providerCode"`
	ProviderDisplayName string `json:"providerDisplayName"`
}

// TaskProvidersResponse is GET /tasks/providers — a flat (un-wrapped) array.
type TaskProvidersResponse []TaskConnectionInfoDto

// TaskGroupDto is a group/section/status-option within a board.
type TaskGroupDto struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Color *string `json:"color,omitempty"`
}

// TaskColumnDto describes a column/property on a board.
type TaskColumnDto struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// TaskTagDto is a board-level tag (Monday).
type TaskTagDto struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

// TaskBoardDto is one board (Monday board / Notion database / Asana project / Jira project / Linear team).
type TaskBoardDto struct {
	ID            string          `json:"id"`
	Name          *string         `json:"name,omitempty"`
	Description   *string         `json:"description,omitempty"`
	State         *string         `json:"state,omitempty"`
	Groups        []TaskGroupDto  `json:"groups,omitempty"`
	Columns       []TaskColumnDto `json:"columns,omitempty"`
	WorkspaceID   *string         `json:"workspaceId,omitempty"`
	WorkspaceName *string         `json:"workspaceName,omitempty"`
	FolderID      *string         `json:"folderId,omitempty"`
	FolderName    *string         `json:"folderName,omitempty"`
	Tags          []TaskTagDto    `json:"tags,omitempty"`
}

// TaskColumnValueDto is one column's value on an item (provider-specific column metadata).
type TaskColumnValueDto struct {
	ColumnID    string  `json:"columnId"`
	ColumnTitle string  `json:"columnTitle"`
	Type        string  `json:"type"`
	Text        *string `json:"text,omitempty"`
	Value       *string `json:"value,omitempty"`
}

// TaskCommentDto is one comment on an item.
type TaskCommentDto struct {
	ID         string  `json:"id"`
	Body       *string `json:"body,omitempty"`
	AuthorName *string `json:"authorName,omitempty"`
	CreatedAt  *string `json:"createdAt,omitempty"`
}

// TaskItemDto is one item (task / issue / page) on a board.
// id, groupId, groupName are structural and always returned even when other
// fields are masked off by the token's visibleTaskFields.
type TaskItemDto struct {
	ID           string               `json:"id"`
	Name         *string              `json:"name,omitempty"`
	GroupID      *string              `json:"groupId,omitempty"`
	GroupName    *string              `json:"groupName,omitempty"`
	Status       *string              `json:"status,omitempty"`
	Assignees    []string             `json:"assignees,omitempty"`
	DueDate      *string              `json:"dueDate,omitempty"`
	Priority     *string              `json:"priority,omitempty"`
	Labels       []string             `json:"labels,omitempty"`
	Description  *string              `json:"description,omitempty"`
	Comments     []TaskCommentDto     `json:"comments,omitempty"`
	ColumnValues []TaskColumnValueDto `json:"columnValues,omitempty"`
	SubItems     []TaskItemDto        `json:"subItems,omitempty"`
}

// TaskBlockDto is one block of a Notion page body.
type TaskBlockDto struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Text        *string                `json:"text,omitempty"`
	RichText    *string                `json:"richText,omitempty"`
	HasChildren bool                   `json:"hasChildren"`
	Children    []TaskBlockDto         `json:"children,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TaskSearchResultDto is one hit in a cross-board search.
type TaskSearchResultDto struct {
	BoardID   string      `json:"boardId"`
	BoardName *string     `json:"boardName,omitempty"`
	Item      TaskItemDto `json:"item"`
}

// ── Response wrappers ──

// TaskBoardsResponse wraps a paginated board list.
type TaskBoardsResponse struct {
	Provider   *string        `json:"provider,omitempty"`
	Boards     []TaskBoardDto `json:"boards"`
	NextCursor *string        `json:"nextCursor,omitempty"`
	NextPage   *int           `json:"nextPage,omitempty"`
	AccessInfo *string        `json:"accessInfo,omitempty"`
}

// TaskBoardResponse wraps a single-board response.
type TaskBoardResponse struct {
	Provider   *string       `json:"provider,omitempty"`
	Board      *TaskBoardDto `json:"board,omitempty"`
	AccessInfo *string       `json:"accessInfo,omitempty"`
}

// TaskItemsResponse wraps a paginated item list. TotalCount is the upstream
// count BEFORE assignee-rule filtering — len(Items) may be smaller per page
// when access rules are active.
type TaskItemsResponse struct {
	Provider   *string       `json:"provider,omitempty"`
	Items      []TaskItemDto `json:"items"`
	NextCursor *string       `json:"nextCursor,omitempty"`
	NextPage   *int          `json:"nextPage,omitempty"`
	TotalCount *int          `json:"totalCount,omitempty"`
	AccessInfo *string       `json:"accessInfo,omitempty"`
}

// TaskItemResponse wraps a single-item response.
type TaskItemResponse struct {
	Provider   *string      `json:"provider,omitempty"`
	Item       *TaskItemDto `json:"item,omitempty"`
	AccessInfo *string      `json:"accessInfo,omitempty"`
}

// TaskCommentsResponse wraps an item's comment list.
type TaskCommentsResponse struct {
	Provider   *string          `json:"provider,omitempty"`
	ItemID     string           `json:"itemId"`
	Comments   []TaskCommentDto `json:"comments"`
	AccessInfo *string          `json:"accessInfo,omitempty"`
}

// TaskSearchResponse wraps a cross-board search. BoardsFailed > 0 means
// partial results — treat as a soft warning, not an error.
type TaskSearchResponse struct {
	Provider       *string               `json:"provider,omitempty"`
	Query          string                `json:"query"`
	Results        []TaskSearchResultDto `json:"results"`
	TotalResults   int                   `json:"totalResults"`
	BoardsSearched int                   `json:"boardsSearched"`
	BoardsFailed   int                   `json:"boardsFailed"`
	AccessInfo     *string               `json:"accessInfo,omitempty"`
}

// TaskBlockListResponse wraps a Notion page-body block list.
type TaskBlockListResponse struct {
	Blocks     []TaskBlockDto `json:"blocks"`
	NextCursor *string        `json:"nextCursor,omitempty"`
	HasMore    bool           `json:"hasMore"`
	AccessInfo *string        `json:"accessInfo,omitempty"`
}

// ── Param structs ──

// TaskBoardListParams parameterises GET /tasks/boards.
type TaskBoardListParams struct {
	Provider string
	Limit    int
	Cursor   string
	Page     int
}

// TaskItemListParams parameterises GET /tasks/boards/{id}/items.
type TaskItemListParams struct {
	Provider string
	BoardID  string
	Limit    int
	Cursor   string
	Page     int
	GroupID  string
	Query    string
	Status   string
}

// TaskSearchParams parameterises GET /tasks/items/search.
type TaskSearchParams struct {
	Provider string
	Query    string
	Limit    int
	BoardIDs []string
}

// TaskBlockListParams parameterises GET /tasks/items/{id}/blocks.
type TaskBlockListParams struct {
	Provider string
	ItemID   string
	Limit    int
	Cursor   string
}

// ── Request bodies ──

// CreateTaskItemRequest is the POST /tasks/boards/{id}/items body.
type CreateTaskItemRequest struct {
	Name    string            `json:"name"`
	GroupID *string           `json:"groupId,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// UpdateTaskItemRequest is the PATCH /tasks/items/{id} body. Only the keys
// in Fields are updated; omitted keys are left untouched. Empty map -> 400.
type UpdateTaskItemRequest struct {
	Fields map[string]string `json:"fields"`
}

// AddTaskCommentRequest is the POST /tasks/items/{id}/comments body.
type AddTaskCommentRequest struct {
	Body string `json:"body"`
}

// AppendBlockInput is one entry of an AppendBlocksRequest.
type AppendBlockInput struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AppendBlocksRequest is the POST /tasks/items/{id}/blocks body (Notion).
type AppendBlocksRequest struct {
	Blocks []AppendBlockInput `json:"blocks"`
}

// ── Mutation results ──

// TaskItemResult is returned by POST/PATCH on items. RejectedFields lists
// keys stripped by the token's writability mask (HTTP 200 still — only
// "every field rejected" becomes 403 NO_WRITABLE_FIELDS).
type TaskItemResult struct {
	Provider       *string      `json:"provider,omitempty"`
	Success        bool         `json:"success"`
	ItemID         *string      `json:"itemId,omitempty"`
	Item           *TaskItemDto `json:"item,omitempty"`
	ErrorCode      *string      `json:"errorCode,omitempty"`
	ErrorMessage   *string      `json:"errorMessage,omitempty"`
	RejectedFields []string     `json:"rejectedFields,omitempty"`
}

// TaskOperationResult is returned by DELETE /tasks/items/{id}.
type TaskOperationResult struct {
	Provider     *string `json:"provider,omitempty"`
	Success      bool    `json:"success"`
	ErrorCode    *string `json:"errorCode,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// TaskCommentResult is returned by POST /tasks/items/{id}/comments.
type TaskCommentResult struct {
	Provider     *string `json:"provider,omitempty"`
	Success      bool    `json:"success"`
	CommentID    *string `json:"commentId,omitempty"`
	ErrorCode    *string `json:"errorCode,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// AppendBlocksResponse is returned by POST /tasks/items/{id}/blocks.
type AppendBlocksResponse struct {
	Success        bool    `json:"success"`
	BlocksAppended int     `json:"blocksAppended"`
	Error          *string `json:"error,omitempty"`
	ErrorCode      *string `json:"errorCode,omitempty"`
}
