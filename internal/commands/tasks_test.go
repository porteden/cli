package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFieldsFlag(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantErr string // substring to match in error; empty = expect success
		want    map[string]string
	}{
		{
			name:  "empty input returns nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "single key=value",
			input: []string{"status=Done"},
			want:  map[string]string{"status": "Done"},
		},
		{
			name:  "multiple distinct keys",
			input: []string{"status=Done", "priority=High"},
			want:  map[string]string{"status": "Done", "priority": "High"},
		},
		{
			name:  "value containing equals signs preserved verbatim after first split",
			input: []string{"raw={\"k\":\"a=b\"}"},
			want:  map[string]string{"raw": `{"k":"a=b"}`},
		},
		{
			name:  "empty value allowed (caller may want to clear a column)",
			input: []string{"status="},
			want:  map[string]string{"status": ""},
		},
		{
			name:  "leading/trailing whitespace in key is trimmed",
			input: []string{"  status  =Done"},
			want:  map[string]string{"status": "Done"},
		},
		{
			name:    "missing = is rejected",
			input:   []string{"status"},
			wantErr: "expected key=value",
		},
		{
			name:    "empty key is rejected",
			input:   []string{"=Done"},
			wantErr: "empty key",
		},
		{
			name:    "whitespace-only key is rejected",
			input:   []string{"   =Done"},
			wantErr: "empty key",
		},
		{
			name:    "duplicate keys are rejected",
			input:   []string{"status=Done", "status=Pending"},
			wantErr: "duplicate key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFieldsFlag(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len mismatch: got %d (%v), want %d (%v)", len(got), got, len(tt.want), tt.want)
			}
			for k, v := range tt.want {
				gv, ok := got[k]
				if !ok {
					t.Errorf("missing key %q", k)
					continue
				}
				if gv != v {
					t.Errorf("key %q: got %q, want %q", k, gv, v)
				}
			}
		})
	}
}

func TestParseBlocksInput(t *testing.T) {
	dir := t.TempDir()

	validJSON := `[{"type":"heading_2","text":"From CLI"},{"type":"paragraph","text":"hi"}]`
	emptyJSON := `[]`
	malformedJSON := `[{"type":"heading_2","text":"unclosed`

	validFile := filepath.Join(dir, "valid.json")
	if err := os.WriteFile(validFile, []byte(validJSON), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	emptyFile := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(emptyFile, []byte(emptyJSON), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	malformedFile := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(malformedFile, []byte(malformedJSON), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	missingFile := filepath.Join(dir, "does-not-exist.json")

	tests := []struct {
		name      string
		inline    string
		file      string
		wantErr   string
		wantCount int
	}{
		{
			name:    "neither inline nor file is rejected",
			wantErr: "provide --blocks",
		},
		{
			name:    "both inline and file rejected",
			inline:  validJSON,
			file:    validFile,
			wantErr: "mutually exclusive",
		},
		{
			name:      "inline valid JSON",
			inline:    validJSON,
			wantCount: 2,
		},
		{
			name:    "inline malformed JSON",
			inline:  malformedJSON,
			wantErr: "invalid blocks JSON",
		},
		{
			name:    "inline empty array",
			inline:  emptyJSON,
			wantErr: "empty",
		},
		{
			name:      "file valid JSON",
			file:      validFile,
			wantCount: 2,
		},
		{
			name:    "file empty array",
			file:    emptyFile,
			wantErr: "empty",
		},
		{
			name:    "file malformed JSON",
			file:    malformedFile,
			wantErr: "invalid blocks JSON",
		},
		{
			name:    "file missing",
			file:    missingFile,
			wantErr: "cannot read --blocks-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBlocksInput(tt.inline, tt.file)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("got %d blocks, want %d", len(got), tt.wantCount)
			}
		})
	}
}
