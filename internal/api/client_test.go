package api

import (
	"os"
	"testing"
	"time"
)

// getTestClient returns a client configured for integration testing.
// Skips the test if PE_API_KEY is not set.
func getTestClient(t *testing.T) *Client {
	apiKey := os.Getenv("PE_API_KEY")
	if apiKey == "" {
		t.Skip("PE_API_KEY not set, skipping integration test")
	}
	return NewClient(apiKey)
}

func TestAuthStatus(t *testing.T) {
	client := getTestClient(t)

	status, err := client.GetAuthStatus()
	if err != nil {
		t.Fatalf("GetAuthStatus failed: %v", err)
	}

	if status.Email == "" {
		t.Error("Expected non-empty email")
	}
	if status.OperatorName == "" {
		t.Error("Expected non-empty operator name")
	}
	if status.KeyID == 0 {
		t.Error("Expected non-zero key ID")
	}

	t.Logf("Authenticated as: %s (Operator: %s, KeyID: %d)", status.Email, status.OperatorName, status.KeyID)
}

func TestGetCalendars(t *testing.T) {
	client := getTestClient(t)

	resp, err := client.GetCalendars()
	if err != nil {
		t.Fatalf("GetCalendars failed: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Expected non-nil data")
	}

	t.Logf("Found %d calendar(s)", len(resp.Data))

	for _, cal := range resp.Data {
		if cal.ID == 0 {
			t.Error("Expected non-zero calendar ID")
		}
		if cal.Name == "" {
			t.Error("Expected non-empty calendar name")
		}
		t.Logf("  - Calendar %d: %s (%s, primary=%v, tz=%s)", cal.ID, cal.Name, cal.Provider, cal.IsPrimary, cal.Timezone)
	}
}

func TestGetEventsToday(t *testing.T) {
	client := getTestClient(t)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	params := EventParams{
		From:  startOfDay,
		To:    endOfDay,
		Limit: 50,
	}

	resp, err := client.GetEvents(params)
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	t.Logf("Found %d event(s) today", len(resp.Events))
	if resp.AccessInfo != "" {
		t.Logf("Access info: %s", resp.AccessInfo)
	}

	for _, event := range resp.Events {
		t.Logf("  - %s (%s)", event.Title, event.StartUtc)
	}
}

func TestGetEventsWeek(t *testing.T) {
	client := getTestClient(t)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfWeek := startOfDay.Add(7 * 24 * time.Hour)

	params := EventParams{
		From:  startOfDay,
		To:    endOfWeek,
		Limit: 50,
	}

	resp, err := client.GetEvents(params)
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	t.Logf("Found %d event(s) this week", len(resp.Events))
}

func TestGetEventsDateRange(t *testing.T) {
	client := getTestClient(t)

	// Test a specific date range (current month)
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	params := EventParams{
		From:  startOfMonth,
		To:    endOfMonth,
		Limit: 100,
	}

	resp, err := client.GetEvents(params)
	if err != nil {
		t.Fatalf("GetEvents with date range failed: %v", err)
	}

	t.Logf("Found %d event(s) this month", len(resp.Events))

	if resp.Meta != nil {
		t.Logf("Meta: count=%d, hasMore=%v", resp.Meta.Count, resp.Meta.HasMore)
	}
}

func TestSearchViaEvents(t *testing.T) {
	client := getTestClient(t)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfMonth := startOfDay.AddDate(0, 1, 0)

	params := EventParams{
		From:  startOfDay,
		To:    endOfMonth,
		Limit: 50,
		Query: "meeting",
	}

	resp, err := client.GetEvents(params)
	if err != nil {
		t.Fatalf("GetEvents with query failed: %v", err)
	}

	t.Logf("Found %d event(s) matching 'meeting'", len(resp.Events))
}

func TestGetEventsByContact(t *testing.T) {
	client := getTestClient(t)

	params := EventsByContactParams{
		Email: "test@example.com",
		Limit: 50,
	}

	resp, err := client.GetEventsByContact(params)
	if err != nil {
		t.Fatalf("GetEventsByContact failed: %v", err)
	}

	t.Logf("Found %d event(s) with contact 'test@example.com'", len(resp.Events))
}

func TestRespondToEvent_NotFound(t *testing.T) {
	client := getTestClient(t)

	// Test with a non-existent event ID - should return an error
	_, err := client.RespondToEvent("999999", "accepted")
	if err == nil {
		t.Fatal("Expected error for non-existent event, got nil")
	}

	t.Logf("Got expected error for non-existent event: %v", err)
}

func TestGetEvent_NotFound(t *testing.T) {
	client := getTestClient(t)

	// Test with a non-existent event ID - should return an error
	_, err := client.GetEvent("999999")
	if err == nil {
		t.Fatal("Expected error for non-existent event, got nil")
	}

	t.Logf("Got expected error for non-existent event: %v", err)
}

func TestUpdateEvent_NotFound(t *testing.T) {
	client := getTestClient(t)

	req := UpdateEventRequest{
		Summary: "Test Update",
	}

	_, err := client.UpdateEvent("999999", req)
	if err == nil {
		t.Fatal("Expected error for non-existent event, got nil")
	}

	t.Logf("Got expected error for non-existent event: %v", err)
}

func TestDeleteEvent_NotFound(t *testing.T) {
	client := getTestClient(t)

	_, err := client.DeleteEvent("999999", true)
	if err == nil {
		t.Fatal("Expected error for non-existent event, got nil")
	}

	t.Logf("Got expected error for non-existent event: %v", err)
}

func TestGetFreeBusy(t *testing.T) {
	client := getTestClient(t)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfWeek := startOfDay.Add(7 * 24 * time.Hour)

	params := FreeBusyParams{
		From: startOfDay,
		To:   endOfWeek,
	}

	resp, err := client.GetFreeBusy(params)
	if err != nil {
		t.Fatalf("GetFreeBusy failed: %v", err)
	}

	t.Logf("Got free/busy info for %d calendar(s)", len(resp.Calendars))
	if resp.AccessInfo != "" {
		t.Logf("Access info: %s", resp.AccessInfo)
	}
}

func TestGetAllEvents(t *testing.T) {
	client := getTestClient(t)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfWeek := startOfDay.Add(7 * 24 * time.Hour)

	params := EventParams{
		From:  startOfDay,
		To:    endOfWeek,
		Limit: 10, // Small limit to test pagination
	}

	resp, err := client.GetAllEvents(params)
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}

	t.Logf("GetAllEvents returned %d event(s)", len(resp.Events))
}
