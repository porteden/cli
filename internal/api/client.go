package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/porteden/cli/internal/apierr"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	baseURL := "https://cliv1b.porteden.com"
	if envURL := os.Getenv("PE_API_URL"); envURL != "" {
		baseURL = envURL
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: NewHTTPClient(apiKey),
	}
}

// WithBaseURL sets a custom base URL (useful for testing)
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

func (c *Client) Get(path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := c.doWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, apierr.ParseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) Post(path string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := c.doWithRetry(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, apierr.ParseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) Patch(path string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := c.doWithRetry(ctx, "PATCH", path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, apierr.ParseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) Delete(path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := c.doWithRetry(ctx, "DELETE", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, apierr.ParseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}

// GetAuthStatus returns the current authentication status
func (c *Client) GetAuthStatus() (*AuthStatusResponse, error) {
	body, err := c.Get("/api/auth/token/status")
	if err != nil {
		return nil, err
	}

	var status AuthStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &status, nil
}

// Logout revokes the current API key
func (c *Client) Logout() error {
	_, err := c.Post("/api/auth/token/logout", nil)
	return err
}

// GetCalendars returns all calendars
func (c *Client) GetCalendars() (*CalendarsResponse, error) {
	body, err := c.Get("/api/access/calendar/calendars")
	if err != nil {
		return nil, err
	}

	var response CalendarsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetEvents returns events based on parameters
func (c *Client) GetEvents(params EventParams) (*EventsResponse, error) {
	v := url.Values{}
	if !params.From.IsZero() {
		v.Set("from", params.From.Format(time.RFC3339))
	}
	if !params.To.IsZero() {
		v.Set("to", params.To.Format(time.RFC3339))
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.CalendarID > 0 {
		v.Set("calendarId", strconv.FormatInt(params.CalendarID, 10))
	}
	if params.Offset > 0 {
		v.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.IncludeCancelled {
		v.Set("includeCancelled", "true")
	}
	if params.Query != "" {
		v.Set("q", params.Query)
	}
	if params.Attendees != "" {
		v.Set("attendees", params.Attendees)
	}

	body, err := c.Get("/api/access/calendar/events?" + v.Encode())
	if err != nil {
		return nil, err
	}

	var response EventsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetEvent returns a single event by ID
func (c *Client) GetEvent(eventID string) (*SingleEventResponse, error) {
	path := "/api/access/calendar/events/" + url.PathEscape(eventID)
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SingleEventResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateEvent creates a new event
func (c *Client) CreateEvent(req CreateEventRequest) (*Event, error) {
	body, err := c.Post("/api/access/calendar/events", req)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &event, nil
}

// UpdateEvent updates an existing event (partial update)
func (c *Client) UpdateEvent(eventID string, req UpdateEventRequest) (*Event, error) {
	path := "/api/access/calendar/events/" + url.PathEscape(eventID)
	body, err := c.Patch(path, req)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &event, nil
}

// DeleteEvent deletes a calendar event
func (c *Client) DeleteEvent(eventID string, notifyAttendees bool) (*DeleteEventResponse, error) {
	v := url.Values{}
	v.Set("notifyAttendees", strconv.FormatBool(notifyAttendees))

	path := "/api/access/calendar/events/" + url.PathEscape(eventID) + "?" + v.Encode()
	body, err := c.Delete(path)
	if err != nil {
		return nil, err
	}

	var response DeleteEventResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// RespondToEvent responds to an event invitation (accept/decline/tentative,
// with an optional comment and organizer-notification toggle)
func (c *Client) RespondToEvent(eventID string, req EventRespondRequest) (*Event, error) {
	path := "/api/access/calendar/events/" + url.PathEscape(eventID) + "/respond"
	body, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &event, nil
}

// GetFreeBusy returns free/busy information for calendars
func (c *Client) GetFreeBusy(params FreeBusyParams) (*FreeBusyResponse, error) {
	v := url.Values{}
	v.Set("from", params.From.Format(time.RFC3339))
	v.Set("to", params.To.Format(time.RFC3339))
	if params.Calendars != "" {
		v.Set("calendars", params.Calendars)
	}

	body, err := c.Get("/api/access/calendar/freebusy?" + v.Encode())
	if err != nil {
		return nil, err
	}

	var response FreeBusyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetEventsByContact returns events with a specific contact
// Requires at least one of: email or name
// email and name parameters support partial matching (case-insensitive)
func (c *Client) GetEventsByContact(params EventsByContactParams) (*EventsResponse, error) {
	v := url.Values{}
	if params.Email != "" {
		v.Set("email", params.Email)
	}
	if params.Name != "" {
		v.Set("name", params.Name)
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		v.Set("offset", strconv.Itoa(params.Offset))
	}

	body, err := c.Get("/api/access/calendar/events/by-contact?" + v.Encode())
	if err != nil {
		return nil, err
	}

	var response EventsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// ==================== EMAIL METHODS ====================

// GetEmails returns emails based on search parameters
func (c *Client) GetEmails(params EmailParams) (*EmailsResponse, error) {
	v := url.Values{}
	if params.Query != "" {
		v.Set("q", params.Query)
	}
	if params.From != "" {
		v.Set("from", params.From)
	}
	if params.To != "" {
		v.Set("to", params.To)
	}
	if params.Subject != "" {
		v.Set("subject", params.Subject)
	}
	if params.Label != "" {
		v.Set("label", params.Label)
	}
	if params.Unread != nil {
		v.Set("unread", strconv.FormatBool(*params.Unread))
	}
	if params.HasAttachment != nil {
		v.Set("hasAttachment", strconv.FormatBool(*params.HasAttachment))
	}
	if !params.After.IsZero() {
		v.Set("after", params.After.Format(time.RFC3339))
	}
	if !params.Before.IsZero() {
		v.Set("before", params.Before.Format(time.RFC3339))
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.IncludeBody {
		v.Set("includeBody", "true")
	}
	if params.PageToken != "" {
		v.Set("pageToken", params.PageToken)
	}

	body, err := c.Get("/api/access/email/messages?" + v.Encode())
	if err != nil {
		return nil, err
	}

	var response EmailsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAllEmails fetches all emails by auto-paginating through results.
// Each page already refills toward `limit` from subsequent pages when the
// firewall trims the first batch; we iterate until HasMore is false or the
// safety cap fires.
func (c *Client) GetAllEmails(params EmailParams) (*EmailsResponse, error) {
	var allEmails []Email
	var accessInfo string
	var authWarnings []string
	const maxPages = 100

	for page := 0; page < maxPages; page++ {
		resp, err := c.GetEmails(params)
		if err != nil {
			return nil, err
		}

		allEmails = append(allEmails, resp.Emails...)
		accessInfo = resp.AccessInfo
		authWarnings = resp.AuthWarnings

		if !resp.HasMore || resp.NextPageToken == "" {
			return &EmailsResponse{
				Emails:       allEmails,
				AccessInfo:   accessInfo,
				AuthWarnings: authWarnings,
			}, nil
		}

		params.PageToken = resp.NextPageToken
	}

	// Safety: return what we have after hitting page limit
	return &EmailsResponse{
		Emails:       allEmails,
		HasMore:      true,
		AccessInfo:   accessInfo,
		AuthWarnings: authWarnings,
	}, nil
}

// Email/thread IDs are prefixed (`google:…`, `m365:…`); url.PathEscape
// turns the `:` into `%3A` as the API requires when embedding in the path.
const emailBase = "/api/access/email"

// GetEmail returns a single email by ID
func (c *Client) GetEmail(emailID string, includeBody bool) (*SingleEmailResponse, error) {
	path := emailBase + "/messages/" + url.PathEscape(emailID)
	if !includeBody {
		path += "?includeBody=false"
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SingleEmailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetThread returns all messages in a thread by ID
func (c *Client) GetThread(threadID string) (*ThreadResponse, error) {
	path := emailBase + "/threads/" + url.PathEscape(threadID)
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	// The API may wrap the thread in a "thread" key with accessInfo at the top level
	var wrapper struct {
		Thread     ThreadResponse `json:"thread"`
		AccessInfo string         `json:"accessInfo,omitempty"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	wrapper.Thread.AccessInfo = wrapper.AccessInfo
	return &wrapper.Thread, nil
}

// SendEmail sends a new email
func (c *Client) SendEmail(req SendEmailRequest) (*EmailActionResponse, error) {
	body, err := c.Post(emailBase+"/messages/send", req)
	if err != nil {
		return nil, err
	}

	var response EmailActionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// ReplyToEmail replies to an existing email
func (c *Client) ReplyToEmail(emailID string, req ReplyEmailRequest) (*EmailActionResponse, error) {
	path := emailBase + "/messages/" + url.PathEscape(emailID) + "/reply"
	body, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var response EmailActionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// ForwardEmail forwards an email to specified recipients
func (c *Client) ForwardEmail(emailID string, req ForwardEmailRequest) (*EmailActionResponse, error) {
	path := emailBase + "/messages/" + url.PathEscape(emailID) + "/forward"
	body, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var response EmailActionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteEmail deletes (trashes) an email
func (c *Client) DeleteEmail(emailID string) error {
	path := emailBase + "/messages/" + url.PathEscape(emailID)
	_, err := c.Delete(path)
	return err
}

// ModifyEmail modifies email properties (read status, labels)
func (c *Client) ModifyEmail(emailID string, req ModifyEmailRequest) error {
	path := emailBase + "/messages/" + url.PathEscape(emailID)
	_, err := c.Patch(path, req)
	return err
}

// Put sends a PUT request with JSON body
func (c *Client) Put(path string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := c.doWithRetry(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, apierr.ParseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}

// PostRaw sends a POST request with a raw byte body and specified Content-Type
func (c *Client) PostRaw(path string, body []byte, contentType string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, apierr.ParseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}

// ==================== DRIVE METHODS ====================

const driveBase = "/api/access/drive"

// GetDriveFiles returns drive files matching the given parameters
func (c *Client) GetDriveFiles(params DriveListParams) (*DriveFilesResponse, error) {
	v := url.Values{}
	if params.Q != "" {
		v.Set("q", params.Q)
	}
	if params.FolderID != "" {
		v.Set("folderId", params.FolderID)
	}
	if params.MimeType != "" {
		v.Set("mimeType", params.MimeType)
	}
	if params.Name != "" {
		v.Set("name", params.Name)
	}
	if params.TrashedOnly {
		v.Set("trashedOnly", "true")
	}
	if params.SharedWithMe {
		v.Set("sharedWithMe", "true")
	}
	if params.ModifiedAfter != "" {
		v.Set("modifiedAfter", params.ModifiedAfter)
	}
	if params.ModifiedBefore != "" {
		v.Set("modifiedBefore", params.ModifiedBefore)
	}
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.PageToken != "" {
		v.Set("pageToken", params.PageToken)
	}
	if params.OrderBy != "" {
		v.Set("orderBy", params.OrderBy)
	}

	path := driveBase + "/files"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response DriveFilesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAllDriveFiles fetches all drive files by auto-paginating (safety cap: 50 pages)
func (c *Client) GetAllDriveFiles(params DriveListParams) (*DriveFilesResponse, error) {
	var allFiles []DriveFile
	var accessInfo *string
	var authWarnings []string
	const maxPages = 50

	for page := 0; page < maxPages; page++ {
		resp, err := c.GetDriveFiles(params)
		if err != nil {
			return nil, err
		}

		allFiles = append(allFiles, resp.Files...)
		accessInfo = resp.AccessInfo
		authWarnings = resp.AuthWarnings

		if !resp.HasMore || resp.NextPageToken == nil || *resp.NextPageToken == "" {
			return &DriveFilesResponse{
				Files:        allFiles,
				HasMore:      false,
				AccessInfo:   accessInfo,
				AuthWarnings: authWarnings,
			}, nil
		}

		params.PageToken = *resp.NextPageToken
	}

	// Safety cap reached
	return &DriveFilesResponse{
		Files:        allFiles,
		HasMore:      true,
		AccessInfo:   accessInfo,
		AuthWarnings: authWarnings,
	}, nil
}

// GetDriveFile returns metadata for a single drive file
func (c *Client) GetDriveFile(fileID string) (*SingleDriveFileResponse, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID)
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SingleDriveFileResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetDriveFileContent returns the textual content of a file. For Google
// Workspace types (Sheets, Slides) the response steers to the dedicated
// endpoint via Readable=false + Reason; the HTTP status is always 200.
func (c *Client) GetDriveFileContent(fileID string) (*DriveFileContentResponse, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID) + "/content"
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response DriveFileContentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateDriveFile creates a file with inline UTF-8 text content (POST /files).
// For Workspace target MIME types, Drive auto-imports the content; otherwise
// the file is stored as-is. Use UploadDriveFile for binary content.
func (c *Client) CreateDriveFile(req CreateDriveFileWithContentRequest) (*DriveOperationResult, error) {
	respBody, err := c.Post(driveBase+"/files", req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetDriveFileLinks returns view/download/export links for a file
func (c *Client) GetDriveFileLinks(fileID string) (*DriveFileLinkResponse, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID) + "/download"
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response DriveFileLinkResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetDrivePermissions returns the sharing permissions for a file
func (c *Client) GetDrivePermissions(fileID string) (*DrivePermissionsResponse, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID) + "/permissions"
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response DrivePermissionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UploadDriveFile uploads a file to Google Drive. Pass an empty body to create a Google Workspace file.
func (c *Client) UploadDriveFile(fileName, mimeType, folderID, description string, body []byte) (*DriveOperationResult, error) {
	v := url.Values{}
	v.Set("fileName", fileName)
	if mimeType != "" {
		v.Set("mimeType", mimeType)
	}
	if folderID != "" {
		v.Set("folderId", folderID)
	}
	if description != "" {
		v.Set("description", description)
	}

	path := driveBase + "/files/upload?" + v.Encode()
	contentType := "application/octet-stream"
	if mimeType != "" {
		contentType = mimeType
	}

	respBody, err := c.PostRaw(path, body, contentType)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// CreateDriveFolder creates a new folder in Google Drive
func (c *Client) CreateDriveFolder(req CreateFolderRequest) (*DriveOperationResult, error) {
	respBody, err := c.Post(driveBase+"/folders", req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// RenameDriveFile renames a file or folder
func (c *Client) RenameDriveFile(fileID string, req RenameFileRequest) (*DriveOperationResult, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID) + "/rename"
	respBody, err := c.Patch(path, req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// MoveDriveFile moves a file to a different folder
func (c *Client) MoveDriveFile(fileID string, req MoveFileRequest) (*DriveOperationResult, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID) + "/move"
	respBody, err := c.Patch(path, req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// DeleteDriveFile moves a file to trash (204 No Content on success)
func (c *Client) DeleteDriveFile(fileID string) error {
	path := driveBase + "/files/" + url.PathEscape(fileID)
	_, err := c.Delete(path)
	return err
}

// CopyDriveFile duplicates a Drive file, returning the new file's id.
func (c *Client) CopyDriveFile(fileID string, req CopyFileRequest) (*DriveOperationResult, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID) + "/copy"
	respBody, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ShareDriveFile shares a file with a user, group, domain, or anyone
func (c *Client) ShareDriveFile(fileID string, req ShareFileRequest) (*DriveOperationResult, error) {
	path := driveBase + "/files/" + url.PathEscape(fileID) + "/share"
	respBody, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ==================== DOCS METHODS ====================

// GetDocContent returns the content of a Google Doc
func (c *Client) GetDocContent(fileID, format string) (*DocContentResponse, error) {
	v := url.Values{}
	if format != "" && format != "text" {
		v.Set("format", format)
	}

	path := driveBase + "/docs/" + url.PathEscape(fileID) + "/content"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response DocContentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// EditDoc applies text editing operations to a Google Doc
func (c *Client) EditDoc(fileID string, req EditDocRequest) (*DriveOperationResult, error) {
	path := driveBase + "/docs/" + url.PathEscape(fileID) + "/edit"
	respBody, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ==================== SHEETS METHODS ====================

// GetSheetMetadata returns spreadsheet title and sheet tab info
func (c *Client) GetSheetMetadata(fileID string) (*SheetMetadataResponse, error) {
	path := driveBase + "/sheets/" + url.PathEscape(fileID)
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SheetMetadataResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// ReadSheetValues reads cell values from a range in a spreadsheet
func (c *Client) ReadSheetValues(fileID, rangeStr string) (*SheetValuesResponse, error) {
	v := url.Values{}
	v.Set("range", rangeStr)

	path := driveBase + "/sheets/" + url.PathEscape(fileID) + "/values?" + v.Encode()
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SheetValuesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// WriteSheetValues writes cell values to a range in a spreadsheet (overwrites)
func (c *Client) WriteSheetValues(fileID string, req WriteSheetValuesRequest) (*DriveOperationResult, error) {
	path := driveBase + "/sheets/" + url.PathEscape(fileID) + "/values"
	respBody, err := c.Put(path, req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// AppendSheetRows appends rows after the last row with data in the specified range
func (c *Client) AppendSheetRows(fileID string, req AppendSheetRowsRequest) (*DriveOperationResult, error) {
	path := driveBase + "/sheets/" + url.PathEscape(fileID) + "/values:append"
	respBody, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result DriveOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ReadSheetTab reads the entire used range of one tab, identified by numeric
// sheetID (preferred when set) or title.
func (c *Client) ReadSheetTab(fileID, title string, sheetID *int) (*SheetValuesResponse, error) {
	v := url.Values{}
	if sheetID != nil {
		v.Set("sheetId", strconv.Itoa(*sheetID))
	} else {
		v.Set("title", title)
	}

	path := driveBase + "/sheets/" + url.PathEscape(fileID) + "/tabs/values?" + v.Encode()
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SheetValuesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// AddSheetTab adds a worksheet tab and returns its assigned sheetId.
func (c *Client) AddSheetTab(fileID string, req AddSheetTabRequest) (*AddSheetTabResponse, error) {
	path := driveBase + "/sheets/" + url.PathEscape(fileID) + "/tabs"
	respBody, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result AddSheetTabResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// DeleteSheetTab deletes one tab, identified by numeric sheetID (preferred when
// set) or title. The endpoint returns 204 with no body.
func (c *Client) DeleteSheetTab(fileID string, sheetID *int, title string) error {
	v := url.Values{}
	if sheetID != nil {
		v.Set("sheetId", strconv.Itoa(*sheetID))
	} else {
		v.Set("title", title)
	}

	path := driveBase + "/sheets/" + url.PathEscape(fileID) + "/tabs?" + v.Encode()
	_, err := c.Delete(path)
	return err
}

// GetSheetBulkContent reads every tab of a spreadsheet in one upstream call.
// Pass nil/empty ranges to use the metadata-driven default (capped per tab by
// maxRowsPerSheet; 0 means use the server default of 200). Tabs with more
// data than was returned are annotated with Clipped=true and a FullRange
// usable directly against /values.
func (c *Client) GetSheetBulkContent(fileID string, ranges []string, maxRowsPerSheet int) (*SheetBulkContentResponse, error) {
	v := url.Values{}
	if len(ranges) > 0 {
		v.Set("ranges", strings.Join(ranges, ","))
	}
	if maxRowsPerSheet > 0 {
		v.Set("maxRowsPerSheet", strconv.Itoa(maxRowsPerSheet))
	}

	path := driveBase + "/sheets/" + url.PathEscape(fileID) + "/content"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SheetBulkContentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// ==================== SLIDES METHODS ====================

// GetSlidesMetadata returns deck title and per-slide index+title.
func (c *Client) GetSlidesMetadata(fileID string) (*SlidesMetadataResponse, error) {
	path := driveBase + "/slides/" + url.PathEscape(fileID)
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SlidesMetadataResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetSlidesContent returns deck content. format="" defaults to text.
// "structured" returns raw Slides API JSON.
func (c *Client) GetSlidesContent(fileID, format string) (*SlidesContentResponse, error) {
	v := url.Values{}
	if format != "" && format != "text" {
		v.Set("format", format)
	}

	path := driveBase + "/slides/" + url.PathEscape(fileID) + "/content"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var response SlidesContentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAllEvents fetches all events by auto-paginating through results.
// Bounded by a page cap and a zero-progress guard: the firewall filters
// pages server-side, so a page can legitimately come back with Count==0
// while HasMore stays true — without the guard, offset would never advance
// and the loop would hammer the API (and the user's monthly quota) forever.
func (c *Client) GetAllEvents(params EventParams) (*EventsResponse, error) {
	var allEvents []Event
	offset := 0
	var accessInfo string
	var calEmail string
	var requestID string
	finalMeta := &Meta{}
	const maxPages = 100

	for page := 0; page < maxPages; page++ {
		params.Offset = offset
		resp, err := c.GetEvents(params)
		if err != nil {
			return nil, err
		}

		allEvents = append(allEvents, resp.Events...)
		accessInfo = resp.AccessInfo
		calEmail = resp.CurrentUserCalendarEmail
		requestID = resp.RequestID
		if resp.Meta != nil {
			finalMeta.From = resp.Meta.From
			finalMeta.To = resp.Meta.To
			finalMeta.Timestamp = resp.Meta.Timestamp
		}

		// Stop on last page, or on a HasMore=true page that made no forward
		// progress (Count<=0 would leave offset stuck and loop forever).
		if resp.Meta == nil || !resp.Meta.HasMore || resp.Meta.Count <= 0 {
			break
		}

		offset += resp.Meta.Count
	}

	finalMeta.Count = len(allEvents)
	finalMeta.TotalCount = len(allEvents)
	return &EventsResponse{
		RequestID:                requestID,
		Events:                   allEvents,
		Meta:                     finalMeta,
		AccessInfo:               accessInfo,
		CurrentUserCalendarEmail: calEmail,
	}, nil
}

// ==================== TASKS METHODS ====================

const tasksBase = "/api/access/tasks"

// taskPaginationCap mirrors the drive safety cap on auto-paginated --all calls.
const taskPaginationCap = 50

// addProvider sets ?provider= when non-empty; tolerating empty lets the
// backend auto-resolve single-provider accounts.
func addProvider(v url.Values, provider string) {
	if provider != "" {
		v.Set("provider", provider)
	}
}

// GetTaskProviders lists task providers connected to the account.
func (c *Client) GetTaskProviders() (TaskProvidersResponse, error) {
	body, err := c.Get(tasksBase + "/providers")
	if err != nil {
		return nil, err
	}

	var resp TaskProvidersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return resp, nil
}

// GetTaskBoards returns one page of boards.
func (c *Client) GetTaskBoards(params TaskBoardListParams) (*TaskBoardsResponse, error) {
	v := url.Values{}
	addProvider(v, params.Provider)
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		v.Set("cursor", params.Cursor)
	}
	if params.Page > 0 {
		v.Set("page", strconv.Itoa(params.Page))
	}

	path := tasksBase + "/boards"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var resp TaskBoardsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// GetAllTaskBoards auto-paginates boards up to taskPaginationCap pages.
// Advances by cursor when present, otherwise by page; returns a partial
// result with NextCursor/NextPage still set when the cap is hit.
func (c *Client) GetAllTaskBoards(params TaskBoardListParams) (*TaskBoardsResponse, error) {
	var all []TaskBoardDto
	var provider, accessInfo *string

	for page := 0; page < taskPaginationCap; page++ {
		resp, err := c.GetTaskBoards(params)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Boards...)
		provider = resp.Provider
		accessInfo = resp.AccessInfo

		if !advanceTaskCursor(&params.Cursor, &params.Page, resp.NextCursor, resp.NextPage) {
			return &TaskBoardsResponse{
				Provider:   provider,
				Boards:     all,
				AccessInfo: accessInfo,
			}, nil
		}
	}

	// Cap hit — preserve the last cursor/page so the caller can resume.
	final := &TaskBoardsResponse{
		Provider:   provider,
		Boards:     all,
		AccessInfo: accessInfo,
	}
	if params.Cursor != "" {
		c := params.Cursor
		final.NextCursor = &c
	}
	if params.Page > 0 {
		p := params.Page
		final.NextPage = &p
	}
	return final, nil
}

// GetTaskBoard returns one board's metadata (groups + columns).
func (c *Client) GetTaskBoard(boardID, provider string) (*TaskBoardResponse, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/boards/" + url.PathEscape(boardID)
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var resp TaskBoardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// GetBoardItems returns one page of items on a board.
func (c *Client) GetBoardItems(params TaskItemListParams) (*TaskItemsResponse, error) {
	v := url.Values{}
	addProvider(v, params.Provider)
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		v.Set("cursor", params.Cursor)
	}
	if params.Page > 0 {
		v.Set("page", strconv.Itoa(params.Page))
	}
	if params.GroupID != "" {
		v.Set("groupId", params.GroupID)
	}
	if params.Query != "" {
		v.Set("query", params.Query)
	}
	if params.Status != "" {
		v.Set("status", params.Status)
	}

	path := tasksBase + "/boards/" + url.PathEscape(params.BoardID) + "/items"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var resp TaskItemsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// GetAllBoardItems auto-paginates items up to taskPaginationCap pages.
func (c *Client) GetAllBoardItems(params TaskItemListParams) (*TaskItemsResponse, error) {
	var all []TaskItemDto
	var provider, accessInfo *string
	var totalCount *int

	for page := 0; page < taskPaginationCap; page++ {
		resp, err := c.GetBoardItems(params)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		provider = resp.Provider
		accessInfo = resp.AccessInfo
		if resp.TotalCount != nil {
			totalCount = resp.TotalCount
		}

		if !advanceTaskCursor(&params.Cursor, &params.Page, resp.NextCursor, resp.NextPage) {
			return &TaskItemsResponse{
				Provider:   provider,
				Items:      all,
				TotalCount: totalCount,
				AccessInfo: accessInfo,
			}, nil
		}
	}

	final := &TaskItemsResponse{
		Provider:   provider,
		Items:      all,
		TotalCount: totalCount,
		AccessInfo: accessInfo,
	}
	if params.Cursor != "" {
		c := params.Cursor
		final.NextCursor = &c
	}
	if params.Page > 0 {
		p := params.Page
		final.NextPage = &p
	}
	return final, nil
}

// GetTaskItem returns a single item.
func (c *Client) GetTaskItem(itemID, provider string) (*TaskItemResponse, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/items/" + url.PathEscape(itemID)
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var resp TaskItemResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// CreateTaskItem creates a new item on a board.
func (c *Client) CreateTaskItem(boardID, provider string, req CreateTaskItemRequest) (*TaskItemResult, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/boards/" + url.PathEscape(boardID) + "/items"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	respBody, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var result TaskItemResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// UpdateTaskItem patches an existing item.
func (c *Client) UpdateTaskItem(itemID, provider string, req UpdateTaskItemRequest) (*TaskItemResult, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/items/" + url.PathEscape(itemID)
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	respBody, err := c.Patch(path, req)
	if err != nil {
		return nil, err
	}

	var result TaskItemResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// DeleteTaskItem deletes (or archives, for Notion) an item.
func (c *Client) DeleteTaskItem(itemID, provider string) (*TaskOperationResult, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/items/" + url.PathEscape(itemID)
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	respBody, err := c.Delete(path)
	if err != nil {
		return nil, err
	}

	// The endpoint may legitimately return an empty body on success.
	if len(respBody) == 0 {
		return &TaskOperationResult{Success: true}, nil
	}

	var result TaskOperationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// GetItemComments lists comments on an item.
func (c *Client) GetItemComments(itemID, provider string) (*TaskCommentsResponse, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/items/" + url.PathEscape(itemID) + "/comments"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var resp TaskCommentsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// AddItemComment posts a new comment on an item.
func (c *Client) AddItemComment(itemID, provider, body string) (*TaskCommentResult, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/items/" + url.PathEscape(itemID) + "/comments"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	respBody, err := c.Post(path, AddTaskCommentRequest{Body: body})
	if err != nil {
		return nil, err
	}

	var result TaskCommentResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// SearchTasks searches items across all in-scope boards (or the supplied subset).
// Server-side single-shot — the API doesn't paginate this endpoint; bump Limit
// (clamped to 1–200) to widen the result set.
func (c *Client) SearchTasks(params TaskSearchParams) (*TaskSearchResponse, error) {
	v := url.Values{}
	addProvider(v, params.Provider)
	v.Set("query", params.Query)
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if len(params.BoardIDs) > 0 {
		v.Set("boardIds", strings.Join(params.BoardIDs, ","))
	}

	path := tasksBase + "/items/search?" + v.Encode()
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var resp TaskSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// GetItemBlocks returns one page of Notion page-body blocks.
func (c *Client) GetItemBlocks(params TaskBlockListParams) (*TaskBlockListResponse, error) {
	v := url.Values{}
	addProvider(v, params.Provider)
	if params.Limit > 0 {
		v.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		v.Set("cursor", params.Cursor)
	}

	path := tasksBase + "/items/" + url.PathEscape(params.ItemID) + "/blocks"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var resp TaskBlockListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &resp, nil
}

// GetAllItemBlocks auto-paginates blocks up to taskPaginationCap pages.
func (c *Client) GetAllItemBlocks(params TaskBlockListParams) (*TaskBlockListResponse, error) {
	var all []TaskBlockDto
	var accessInfo *string

	for page := 0; page < taskPaginationCap; page++ {
		resp, err := c.GetItemBlocks(params)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Blocks...)
		accessInfo = resp.AccessInfo

		if !resp.HasMore || resp.NextCursor == nil || *resp.NextCursor == "" {
			return &TaskBlockListResponse{
				Blocks:     all,
				HasMore:    false,
				AccessInfo: accessInfo,
			}, nil
		}
		params.Cursor = *resp.NextCursor
	}

	cursor := params.Cursor
	return &TaskBlockListResponse{
		Blocks:     all,
		HasMore:    true,
		NextCursor: &cursor,
		AccessInfo: accessInfo,
	}, nil
}

// AppendItemBlocks appends blocks to a Notion page body.
func (c *Client) AppendItemBlocks(itemID, provider string, blocks []AppendBlockInput) (*AppendBlocksResponse, error) {
	v := url.Values{}
	addProvider(v, provider)

	path := tasksBase + "/items/" + url.PathEscape(itemID) + "/blocks"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	respBody, err := c.Post(path, AppendBlocksRequest{Blocks: blocks})
	if err != nil {
		return nil, err
	}

	var result AppendBlocksResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// advanceTaskCursor consumes the response's NextCursor/NextPage hints into the
// params and reports whether the caller should fetch another page. Cursor wins
// over page when both are non-nil (some providers report both). The unused
// field is cleared so a prior iteration's value can't leak into the next
// request when a provider switches modes (or when the caller seeded both
// --cursor and --page).
func advanceTaskCursor(cursor *string, page *int, nextCursor *string, nextPage *int) bool {
	if nextCursor != nil && *nextCursor != "" {
		*cursor = *nextCursor
		*page = 0
		return true
	}
	if nextPage != nil && *nextPage > 0 {
		*page = *nextPage
		*cursor = ""
		return true
	}
	return false
}
