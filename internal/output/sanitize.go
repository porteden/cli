package output

import (
	"fmt"
	"strconv"
	"strings"
)

// formatCell renders a spreadsheet cell value for table/plain output. JSON
// decodes every number to float64, and the default %v verb prints large values
// in scientific notation (12500000 -> "1.25e+07") and nil as "<nil>" — both
// wrong for an agent or human reading the data. Render floats in plain decimal
// and nil as empty.
func formatCell(v interface{}) string {
	switch n := v.(type) {
	case nil:
		return ""
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	case string:
		return n
	default:
		return fmt.Sprintf("%v", v)
	}
}

// stripControl replaces C0/C1 control characters — including ESC, CR, LF, and
// TAB — with a single space. Upstream API data (email subjects and sender
// names, event titles, file names, task fields) is attacker-influenced: a tab
// or newline shifts or injects rows in tabwriter/TSV output, and an ESC
// sequence would be interpreted by the terminal. Multi-byte UTF-8 runes are
// preserved unchanged.
func stripControl(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isControl(r) {
			return ' '
		}
		return r
	}, s)
}

// sanitizeText is like stripControl but keeps newlines and tabs, for
// multi-line free text (email/event bodies) where those are legitimate
// layout. Other control bytes — notably ESC — are still removed so the data
// cannot emit terminal escape sequences.
func sanitizeText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if isControl(r) {
			return -1 // drop
		}
		return r
	}, s)
}

func isControl(r rune) bool {
	return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f)
}

// truncateRunesSafe truncates s to at most max characters on a rune boundary,
// appending "..." when it had to cut. Unlike the table truncate() it does not
// strip control characters — compact output feeds JSON consumers that want the
// content preserved; it only guarantees the cut never splits a UTF-8 rune
// (a byte-slice truncation there surfaces as U+FFFD after JSON re-encoding).
func truncateRunesSafe(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
