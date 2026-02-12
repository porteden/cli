package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	v.Set("limit", strconv.Itoa(params.Limit))
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

// RespondToEvent responds to an event invitation
func (c *Client) RespondToEvent(eventID, status string) (*Event, error) {
	path := "/api/access/calendar/events/" + url.PathEscape(eventID) + "/respond"
	body, err := c.Post(path, map[string]string{"status": status})
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
	v.Set("limit", strconv.Itoa(params.Limit))
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

// GetAllEmails fetches all emails by auto-paginating through results
func (c *Client) GetAllEmails(params EmailParams) (*EmailsResponse, error) {
	var allEmails []Email
	var accessInfo string
	const maxPages = 100

	for page := 0; page < maxPages; page++ {
		resp, err := c.GetEmails(params)
		if err != nil {
			return nil, err
		}

		allEmails = append(allEmails, resp.Emails...)
		accessInfo = resp.AccessInfo

		if !resp.HasMore || resp.NextPageToken == "" {
			return &EmailsResponse{
				Emails:     allEmails,
				TotalCount: len(allEmails),
				AccessInfo: accessInfo,
			}, nil
		}

		params.PageToken = resp.NextPageToken
	}

	// Safety: return what we have after hitting page limit
	return &EmailsResponse{
		Emails:     allEmails,
		TotalCount: len(allEmails),
		HasMore:    true,
		AccessInfo: accessInfo,
	}, nil
}

// GetEmail returns a single email by ID
func (c *Client) GetEmail(emailID string, includeBody bool) (*SingleEmailResponse, error) {
	v := url.Values{}
	if !includeBody {
		v.Set("includeBody", "false")
	}

	path := "/api/access/email/messages/" + emailID
	if len(v) > 0 {
		path += "?" + v.Encode()
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
	path := "/api/access/email/threads/" + threadID
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	// The API may wrap the thread in a "thread" key
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
	body, err := c.Post("/api/access/email/messages/send", req)
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
	path := "/api/access/email/messages/" + emailID + "/reply"
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
	path := "/api/access/email/messages/" + emailID + "/forward"
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
	path := "/api/access/email/messages/" + emailID
	_, err := c.Delete(path)
	return err
}

// ModifyEmail modifies email properties (read status, labels)
func (c *Client) ModifyEmail(emailID string, req ModifyEmailRequest) error {
	path := "/api/access/email/messages/" + emailID
	_, err := c.Patch(path, req)
	return err
}

// GetAllEvents fetches all events by auto-paginating through results
func (c *Client) GetAllEvents(params EventParams) (*EventsResponse, error) {
	var allEvents []Event
	offset := 0
	var accessInfo string
	var calEmail string

	for {
		params.Offset = offset
		resp, err := c.GetEvents(params)
		if err != nil {
			return nil, err
		}

		allEvents = append(allEvents, resp.Events...)
		accessInfo = resp.AccessInfo
		calEmail = resp.CurrentUserCalendarEmail

		if resp.Meta == nil || !resp.Meta.HasMore {
			// Build final response with aggregated data
			finalMeta := &Meta{
				Count:      len(allEvents),
				TotalCount: len(allEvents),
			}
			if resp.Meta != nil {
				finalMeta.From = resp.Meta.From
				finalMeta.To = resp.Meta.To
				finalMeta.Timestamp = resp.Meta.Timestamp
			}
			return &EventsResponse{
				RequestID:                resp.RequestID,
				Events:                   allEvents,
				Meta:                     finalMeta,
				AccessInfo:               accessInfo,
				CurrentUserCalendarEmail: calEmail,
			}, nil
		}

		offset += resp.Meta.Count
	}
}
