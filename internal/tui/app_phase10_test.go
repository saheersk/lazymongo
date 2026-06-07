package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/saheersk/lazymongo/internal/tui/panels/statusbar"
	"github.com/saheersk/lazymongo/internal/tui/style"
)

func newTestStatusbar(th *style.Theme, uri string) statusbar.Model {
	sb := statusbar.New(th, uri)
	return sb.SetWidth(120)
}

// ── renderWatch ───────────────────────────────────────────────────────────────

func TestRenderWatch_EmptyState_ShowsWaiting(t *testing.T) {
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "users", nil, 0, nil)
	if !strings.Contains(out, "waiting for changes") {
		t.Errorf("empty watch should show 'waiting for changes', got:\n%s", out)
	}
}

func TestRenderWatch_LiveIndicator(t *testing.T) {
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "users", nil, 0, nil)
	if !strings.Contains(out, "LIVE") {
		t.Errorf("live watch should show 'LIVE', got:\n%s", out)
	}
	if !strings.Contains(out, "WATCHING") {
		t.Errorf("should show 'WATCHING', got:\n%s", out)
	}
}

func TestRenderWatch_ShowsDBAndCol(t *testing.T) {
	out := renderWatch(testBase, 120, 40, testTheme(), "testdb", "orders", nil, 0, nil)
	if !strings.Contains(out, "testdb") {
		t.Errorf("should show db name 'testdb', got:\n%s", out)
	}
	if !strings.Contains(out, "orders") {
		t.Errorf("should show collection name 'orders', got:\n%s", out)
	}
}

func TestRenderWatch_Error_ShowsStopped(t *testing.T) {
	watchErr := errors.New("stream closed unexpectedly")
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "users", nil, 0, watchErr)
	if !strings.Contains(out, "STOPPED") {
		t.Errorf("error state should show 'STOPPED', got:\n%s", out)
	}
	if !strings.Contains(out, "stream closed unexpectedly") {
		t.Errorf("should show the error message, got:\n%s", out)
	}
}

func TestRenderWatch_Insert_ShowsGreenBadge(t *testing.T) {
	events := []watchEventMsg{
		{op: "insert", docID: "abc123", doc: map[string]interface{}{"name": "Alice"}, ts: time.Now()},
	}
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "users", events, 0, nil)
	if !strings.Contains(out, "INSERT") {
		t.Errorf("should show INSERT badge, got:\n%s", out)
	}
	if !strings.Contains(out, "abc123") {
		t.Errorf("should show doc ID, got:\n%s", out)
	}
}

func TestRenderWatch_Update_ShowsUpdatedFields(t *testing.T) {
	events := []watchEventMsg{
		{op: "update", docID: "abc123", updatedFields: map[string]interface{}{"email": "new@example.com"}, ts: time.Now()},
	}
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "users", events, 0, nil)
	if !strings.Contains(out, "UPDATE") {
		t.Errorf("should show UPDATE badge, got:\n%s", out)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("should show 'updated' field summary, got:\n%s", out)
	}
}

func TestRenderWatch_Delete_ShowsDeleteBadge(t *testing.T) {
	events := []watchEventMsg{
		{op: "delete", docID: "deadbeef", ts: time.Now()},
	}
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "users", events, 0, nil)
	if !strings.Contains(out, "DELETE") {
		t.Errorf("should show DELETE badge, got:\n%s", out)
	}
}

func TestRenderWatch_MultipleEvents_ShowsCount(t *testing.T) {
	events := []watchEventMsg{
		{op: "insert", docID: "1", ts: time.Now()},
		{op: "delete", docID: "2", ts: time.Now()},
		{op: "update", docID: "3", ts: time.Now()},
	}
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "col", events, 0, nil)
	if !strings.Contains(out, "3 events") {
		t.Errorf("should show '3 events', got:\n%s", out)
	}
}

func TestRenderWatch_Scroll_ShowsOffset(t *testing.T) {
	events := []watchEventMsg{
		{op: "insert", docID: "a", ts: time.Now()},
		{op: "insert", docID: "b", ts: time.Now()},
		{op: "insert", docID: "c", ts: time.Now()},
	}
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "col", events, 1, nil)
	if !strings.Contains(out, "scroll: 1") {
		t.Errorf("scrolled view should show scroll offset, got:\n%s", out)
	}
}

func TestRenderWatch_ShowsStopHint(t *testing.T) {
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "col", nil, 0, nil)
	if !strings.Contains(out, "stop") {
		t.Errorf("should show stop hint, got:\n%s", out)
	}
	if !strings.Contains(out, "scroll") {
		t.Errorf("should show scroll hint, got:\n%s", out)
	}
}

func TestRenderWatch_UnknownOp_ShowsEvent(t *testing.T) {
	events := []watchEventMsg{
		{op: "invalidate", docID: nil, ts: time.Now()},
	}
	out := renderWatch(testBase, 120, 40, testTheme(), "mydb", "col", events, 0, nil)
	if !strings.Contains(out, "INVALIDATE") {
		t.Errorf("should show operation name, got:\n%s", out)
	}
}

// ── renderConnPicker ──────────────────────────────────────────────────────────

func TestRenderConnPicker_ShowsTitle(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "local", URI: "mongodb://localhost:27017"},
	}
	out := renderConnPicker(testBase, 120, 40, testTheme(), profiles, 0, false)
	if !strings.Contains(out, "SWITCH CONNECTION") {
		t.Errorf("should show 'SWITCH CONNECTION' title, got:\n%s", out)
	}
}

func TestRenderConnPicker_ShowsProfileNames(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "local", URI: "mongodb://localhost:27017"},
		{Name: "staging", URI: "mongodb://staging:27017"},
	}
	out := renderConnPicker(testBase, 120, 40, testTheme(), profiles, 0, false)
	if !strings.Contains(out, "local") {
		t.Errorf("should show 'local' profile, got:\n%s", out)
	}
	if !strings.Contains(out, "staging") {
		t.Errorf("should show 'staging' profile, got:\n%s", out)
	}
}

func TestRenderConnPicker_SelectedProfileHasArrow(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "local", URI: "mongodb://localhost:27017"},
		{Name: "staging", URI: "mongodb://staging:27017"},
	}
	// Select first profile (idx=0)
	out := renderConnPicker(testBase, 120, 40, testTheme(), profiles, 0, false)
	if !strings.Contains(out, "▶") {
		t.Errorf("selected profile should have '▶' arrow, got:\n%s", out)
	}
}

func TestRenderConnPicker_SecondProfileSelected(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "local", URI: "mongodb://localhost:27017"},
		{Name: "staging", URI: "mongodb://staging:27017"},
	}
	// Select second profile (idx=1)
	out := renderConnPicker(testBase, 120, 40, testTheme(), profiles, 1, false)
	if !strings.Contains(out, "▶") {
		t.Errorf("selected profile should have '▶' arrow, got:\n%s", out)
	}
	// staging line should contain the arrow
	lines := strings.Split(out, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "▶") && strings.Contains(line, "staging") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("'staging' profile should have the arrow '▶', got:\n%s", out)
	}
}

func TestRenderConnPicker_Switching_ShowsConnectingMessage(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "production", URI: "mongodb://prod:27017"},
	}
	out := renderConnPicker(testBase, 120, 40, testTheme(), profiles, 0, true)
	if !strings.Contains(out, "connecting to production") {
		t.Errorf("switching state should say 'connecting to production', got:\n%s", out)
	}
}

func TestRenderConnPicker_ShowsNavigationHint(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "local", URI: "mongodb://localhost:27017"},
	}
	out := renderConnPicker(testBase, 120, 40, testTheme(), profiles, 0, false)
	if !strings.Contains(out, "navigate") {
		t.Errorf("should show navigation hint, got:\n%s", out)
	}
	if !strings.Contains(out, "connect") {
		t.Errorf("should show 'connect' hint, got:\n%s", out)
	}
	if !strings.Contains(out, "cancel") {
		t.Errorf("should show 'cancel' hint, got:\n%s", out)
	}
}

func TestRenderConnPicker_EmptyList_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderConnPicker panicked on empty list: %v", r)
		}
	}()
	_ = renderConnPicker(testBase, 120, 40, testTheme(), nil, 0, false)
}

func TestRenderConnPicker_LongURI_IsTruncated(t *testing.T) {
	longURI := "mongodb://user:password@very-long-hostname.example.com:27017/?authSource=admin&replicaSet=rs0"
	profiles := []ConnectionProfile{
		{Name: "production", URI: longURI},
	}
	out := renderConnPicker(testBase, 120, 40, testTheme(), profiles, 0, false)
	// The full URI should not appear verbatim (it would overflow the box)
	if strings.Contains(out, longURI) {
		// Check box width — if it fits, that's also fine
		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if len(line) > 80 {
				t.Errorf("line too long (%d chars) in conn picker output: %q", len(line), line)
			}
		}
	}
}

// ── statusbar health indicator ────────────────────────────────────────────────

func TestStatusbar_SetHealth_OKShowsLatency(t *testing.T) {
	th := testTheme()
	sb := newTestStatusbar(th, "mongodb://localhost:27017")
	sb = sb.SetHealth(true, 5)
	out := sb.View()
	if !strings.Contains(out, "5ms") {
		t.Errorf("healthy statusbar should show '5ms', got: %s", out)
	}
}

func TestStatusbar_SetHealth_Offline_ShowsHollowDiamond(t *testing.T) {
	th := testTheme()
	sb := newTestStatusbar(th, "mongodb://localhost:27017")
	sb = sb.SetHealth(false, 0)
	out := sb.View()
	if !strings.Contains(out, "◇") {
		t.Errorf("offline statusbar should show '◇', got: %s", out)
	}
}

func TestStatusbar_BeforeHealthSet_ShowsFilledDiamond(t *testing.T) {
	th := testTheme()
	sb := newTestStatusbar(th, "mongodb://localhost:27017")
	// No SetHealth call yet
	out := sb.View()
	if !strings.Contains(out, "◆") {
		t.Errorf("initial statusbar should show '◆', got: %s", out)
	}
}

func TestStatusbar_SetHealth_ZeroLatency_NoMsShown(t *testing.T) {
	th := testTheme()
	sb := newTestStatusbar(th, "mongodb://localhost:27017")
	sb = sb.SetHealth(true, 0)
	out := sb.View()
	if strings.Contains(out, "0ms") {
		t.Errorf("zero latency should not show '0ms', got: %s", out)
	}
}

func TestStatusbar_SetHealth_OfflineThenOnline(t *testing.T) {
	th := testTheme()
	sb := newTestStatusbar(th, "mongodb://localhost:27017")
	sb = sb.SetHealth(false, 0)
	out := sb.View()
	if !strings.Contains(out, "◇") {
		t.Fatalf("offline should show hollow diamond, got: %s", out)
	}
	sb = sb.SetHealth(true, 3)
	out = sb.View()
	if strings.Contains(out, "◇") {
		t.Errorf("after reconnect should not show hollow diamond, got: %s", out)
	}
	if !strings.Contains(out, "3ms") {
		t.Errorf("after reconnect should show latency, got: %s", out)
	}
}

// ── fmtAgo ───────────────────────────────────────────────────────────────────

func TestFmtAgo_JustNow(t *testing.T) {
	out := fmtAgo(time.Now())
	if out != "just now" {
		t.Errorf("expected 'just now', got %q", out)
	}
}

func TestFmtAgo_Seconds(t *testing.T) {
	out := fmtAgo(time.Now().Add(-10 * time.Second))
	if out != "10s ago" {
		t.Errorf("expected '10s ago', got %q", out)
	}
}

func TestFmtAgo_Minutes(t *testing.T) {
	out := fmtAgo(time.Now().Add(-90 * time.Second))
	if out != "1m ago" {
		t.Errorf("expected '1m ago', got %q", out)
	}
}

func TestFmtAgo_Hours(t *testing.T) {
	out := fmtAgo(time.Now().Add(-2 * time.Hour))
	if out != "2h ago" {
		t.Errorf("expected '2h ago', got %q", out)
	}
}
