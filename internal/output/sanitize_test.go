package output

import "testing"

func TestTruncateRuneSafe(t *testing.T) {
	// "héllo wörld" is multi-byte; a byte slice at an odd offset would split a
	// rune. truncate must cut on rune boundaries and never emit invalid UTF-8.
	got := truncate("héllo wörld", 8)
	if !isValidUTF8(got) {
		t.Fatalf("truncate produced invalid UTF-8: %q", got)
	}
	if want := "héllo..."; got != want {
		t.Fatalf("truncate = %q, want %q", got, want)
	}

	// Pure multi-byte string truncated mid-way stays valid.
	if g := truncate("日本語テキスト", 5); !isValidUTF8(g) {
		t.Fatalf("truncate produced invalid UTF-8 for CJK: %q", g)
	}

	// Short input passes through unchanged (minus control stripping).
	if g := truncate("abc", 10); g != "abc" {
		t.Fatalf("truncate short = %q, want abc", g)
	}
}

func TestTruncateStripsControl(t *testing.T) {
	// A tab or newline in a cell would break table alignment / inject a row;
	// an ESC would emit a terminal sequence. All become spaces.
	got := truncate("a\tb\nc\x1b[31md", 100)
	if want := "a b c [31md"; got != want {
		t.Fatalf("truncate control = %q, want %q", got, want)
	}
}

func TestSanitizeTextKeepsNewlinesDropsEscape(t *testing.T) {
	// An OSC terminal sequence (ESC ] 0 ; title BEL): the active control bytes
	// (ESC, BEL) are dropped, leaving only inert printable residue. Newlines
	// and tabs are preserved as legitimate layout.
	got := sanitizeText("line1\nline2\twith\x1b]0;title\x07tab")
	for _, r := range got {
		if r == 0x1b || r == 0x07 {
			t.Fatalf("sanitizeText left a control byte: %q", got)
		}
	}
	if want := "line1\nline2\twith]0;titletab"; got != want {
		t.Fatalf("sanitizeText = %q, want %q", got, want)
	}
}

func TestFormatCell(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{float64(12500000), "12500000"}, // no scientific notation
		{float64(19820.5), "19820.5"},
		{nil, ""},        // not "<nil>"
		{"text", "text"}, // strings pass through
		{true, "true"},
	}
	for _, c := range cases {
		if got := formatCell(c.in); got != c.want {
			t.Errorf("formatCell(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			// Distinguish a real replacement char in input (none here) from a
			// decode failure: our test inputs contain no literal U+FFFD.
			return false
		}
	}
	return true
}
