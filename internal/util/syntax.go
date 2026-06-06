package util

import (
	"regexp"
	"strings"

	"github.com/saheersk/lazymongo/internal/tui/style"
)

var (
	reKeyValue  = regexp.MustCompile(`^(\s*)("(?:[^"\\]|\\.)*")(\s*:\s*)(.*)$`)
	reString    = regexp.MustCompile(`^".*"[,]?$`)
	reNumber    = regexp.MustCompile(`^-?\d+(\.\d+)?([eE][+-]?\d+)?[,]?$`)
	reBool      = regexp.MustCompile(`^(true|false)[,]?$`)
	reNull      = regexp.MustCompile(`^null[,]?$`)
	reObjectID  = regexp.MustCompile(`^\{"?\$oid"?\s*:\s*"[0-9a-f]{24}"\}[,]?$`)
	reDate      = regexp.MustCompile(`^\{"?\$date"?`)
	reBracket   = regexp.MustCompile(`^[{\[}\]][,]?$`)
)

// SyntaxHighlight applies lipgloss colours to a pretty-printed JSON string.
// It processes the input line-by-line so it never needs to fully parse the JSON.
func SyntaxHighlight(jsonStr string, th *style.Theme) string {
	lines := strings.Split(jsonStr, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, highlightLine(line, th))
	}
	return strings.Join(out, "\n")
}

func highlightLine(line string, th *style.Theme) string {
	if line == "" {
		return line
	}

	// key: value  →  colour key and value separately
	if m := reKeyValue.FindStringSubmatch(line); m != nil {
		indent := m[1]
		key := m[2]   // the quoted key
		colon := m[3] // ": "
		value := m[4] // everything after

		return indent +
			th.JSONKey.Render(key) +
			th.JSONBracket.Render(colon) +
			colorValue(strings.TrimSpace(value), th)
	}

	// array element or standalone bracket — no key
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	return indent + colorValue(trimmed, th)
}

// colorValue colours a JSON value fragment (may include a trailing comma).
func colorValue(v string, th *style.Theme) string {
	bare := strings.TrimRight(v, ",")
	trail := v[len(bare):]

	switch {
	case reObjectID.MatchString(v):
		return th.JSONOID.Render(bare) + th.JSONBracket.Render(trail)

	case reDate.MatchString(v):
		return th.JSONOID.Render(bare) + th.JSONBracket.Render(trail)

	case reString.MatchString(v):
		return th.JSONString.Render(bare) + th.JSONBracket.Render(trail)

	case reNumber.MatchString(v):
		return th.JSONNumber.Render(bare) + th.JSONBracket.Render(trail)

	case reBool.MatchString(v):
		return th.JSONBool.Render(bare) + th.JSONBracket.Render(trail)

	case reNull.MatchString(v):
		return th.JSONNull.Render(bare) + th.JSONBracket.Render(trail)

	case reBracket.MatchString(v):
		return th.JSONBracket.Render(v)

	default:
		return th.JSONBracket.Render(v)
	}
}
