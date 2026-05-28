package api

import "testing"

// TestAdvanceTaskCursor exercises the pagination invariant that matters in
// production: the unused field is cleared on advance so a prior iteration's
// value can't leak into the next request when a provider switches modes
// (cursor → page or page → cursor) or when the caller seeded both fields.
func TestAdvanceTaskCursor(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name        string
		cursor      string
		page        int
		nextCursor  *string
		nextPage    *int
		wantCursor  string
		wantPage    int
		wantAdvance bool
	}{
		{
			name:        "no next-of-either stops iteration",
			cursor:      "stale",
			page:        3,
			wantCursor:  "stale",
			wantPage:    3,
			wantAdvance: false,
		},
		{
			name:        "empty next cursor is treated as absent",
			cursor:      "stale",
			page:        3,
			nextCursor:  strPtr(""),
			wantCursor:  "stale",
			wantPage:    3,
			wantAdvance: false,
		},
		{
			name:        "zero next page is treated as absent",
			cursor:      "stale",
			page:        3,
			nextPage:    intPtr(0),
			wantCursor:  "stale",
			wantPage:    3,
			wantAdvance: false,
		},
		{
			name:        "next cursor advances and clears stale page",
			cursor:      "old",
			page:        2,
			nextCursor:  strPtr("new"),
			wantCursor:  "new",
			wantPage:    0,
			wantAdvance: true,
		},
		{
			name:        "next page advances and clears stale cursor",
			cursor:      "old",
			page:        2,
			nextPage:    intPtr(5),
			wantCursor:  "",
			wantPage:    5,
			wantAdvance: true,
		},
		{
			name:        "cursor wins when both next values are set",
			cursor:      "",
			page:        0,
			nextCursor:  strPtr("c"),
			nextPage:    intPtr(7),
			wantCursor:  "c",
			wantPage:    0,
			wantAdvance: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor, page := tt.cursor, tt.page
			got := advanceTaskCursor(&cursor, &page, tt.nextCursor, tt.nextPage)
			if got != tt.wantAdvance {
				t.Errorf("advance: got %v, want %v", got, tt.wantAdvance)
			}
			if cursor != tt.wantCursor {
				t.Errorf("cursor: got %q, want %q", cursor, tt.wantCursor)
			}
			if page != tt.wantPage {
				t.Errorf("page: got %d, want %d", page, tt.wantPage)
			}
		})
	}
}

// ── Integration tests (require PE_API_KEY + a connected task provider) ──

func TestGetTaskProviders(t *testing.T) {
	client := getTestClient(t)

	resp, err := client.GetTaskProviders()
	if err != nil {
		t.Fatalf("GetTaskProviders failed: %v", err)
	}

	t.Logf("Found %d connected task provider(s)", len(resp))
	for _, p := range resp {
		if p.ProviderCode == "" {
			t.Error("Expected non-empty providerCode")
		}
		t.Logf("  - %s (%s, id=%d)", p.ProviderDisplayName, p.ProviderCode, p.TaskProviderID)
	}
}

func TestGetTaskBoards(t *testing.T) {
	client := getTestClient(t)

	resp, err := client.GetTaskBoards(TaskBoardListParams{Limit: 10})
	if err != nil {
		t.Fatalf("GetTaskBoards failed: %v", err)
	}

	t.Logf("Found %d board(s)", len(resp.Boards))
	if resp.AccessInfo != nil {
		t.Logf("Access info: %s", *resp.AccessInfo)
	}
	for _, b := range resp.Boards {
		if b.ID == "" {
			t.Error("Expected non-empty board ID")
		}
		name := ""
		if b.Name != nil {
			name = *b.Name
		}
		t.Logf("  - %s: %s", b.ID, name)
	}
}

func TestGetTaskItem_NotFound(t *testing.T) {
	client := getTestClient(t)

	_, err := client.GetTaskItem("nonexistent-id-9999", "")
	if err == nil {
		t.Fatal("Expected error for non-existent item, got nil")
	}
	t.Logf("Got expected error: %v", err)
}

func TestSearchTasks(t *testing.T) {
	client := getTestClient(t)

	resp, err := client.SearchTasks(TaskSearchParams{Query: "test", Limit: 5})
	if err != nil {
		t.Fatalf("SearchTasks failed: %v", err)
	}

	t.Logf("Search returned %d result(s) across %d board(s) (%d failed)",
		resp.TotalResults, resp.BoardsSearched, resp.BoardsFailed)
}
