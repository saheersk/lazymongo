package documents

import (
	"testing"
)

// ── distributeWidths ──────────────────────────────────────────────────────────

func TestDistributeWidths_EmptyCols_ReturnsNil(t *testing.T) {
	got := distributeWidths(nil, 100)
	if got != nil {
		t.Errorf("expected nil for empty cols, got %v", got)
	}

	got = distributeWidths([]string{}, 100)
	if got != nil {
		t.Errorf("expected nil for empty slice, got %v", got)
	}
}

func TestDistributeWidths_OneCol_GetsTotalWidth(t *testing.T) {
	cols := []string{"_id"}
	totalW := 80

	widths := distributeWidths(cols, totalW)

	if len(widths) != 1 {
		t.Fatalf("expected 1 width, got %d", len(widths))
	}
	// With 1 column, widths[0] == idW (24) regardless of totalW, because
	// remaining > 0 branch only executes when len(cols) > 1.
	const idW = 24
	if widths[0] != idW {
		t.Errorf("widths[0] = %d; want %d", widths[0], idW)
	}
}

func TestDistributeWidths_TwoCols_IDGets24_OtherGetsRemainder(t *testing.T) {
	cols := []string{"_id", "name"}
	totalW := 100

	widths := distributeWidths(cols, totalW)

	if len(widths) != 2 {
		t.Fatalf("expected 2 widths, got %d", len(widths))
	}

	const idW = 24
	if widths[0] != idW {
		t.Errorf("widths[0] (_id) = %d; want %d", widths[0], idW)
	}

	// remaining = totalW - idW - len(cols) = 100 - 24 - 2 = 74
	// each = 74 / 1 = 74
	expectedOther := (totalW - idW - len(cols)) / (len(cols) - 1)
	if widths[1] != expectedOther {
		t.Errorf("widths[1] = %d; want %d", widths[1], expectedOther)
	}
}

func TestDistributeWidths_ThreeCols_IDGets24_OthersSplitEvenly(t *testing.T) {
	cols := []string{"_id", "name", "age"}
	totalW := 100

	widths := distributeWidths(cols, totalW)

	if len(widths) != 3 {
		t.Fatalf("expected 3 widths, got %d", len(widths))
	}

	const idW = 24
	if widths[0] != idW {
		t.Errorf("widths[0] (_id) = %d; want %d", widths[0], idW)
	}

	// remaining = 100 - 24 - 3 = 73; each = 73/2 = 36
	remaining := totalW - idW - len(cols)
	each := remaining / (len(cols) - 1)
	for i := 1; i < len(widths); i++ {
		if widths[i] != each {
			t.Errorf("widths[%d] = %d; want %d", i, widths[i], each)
		}
	}
}

func TestDistributeWidths_FiveCols_LengthMatchesCols(t *testing.T) {
	cols := []string{"_id", "a", "b", "c", "d"}
	totalW := 200

	widths := distributeWidths(cols, totalW)

	if len(widths) != len(cols) {
		t.Errorf("len(widths) = %d; want %d", len(widths), len(cols))
	}
}

func TestDistributeWidths_VeryNarrow_IDStillGets24(t *testing.T) {
	// Total width less than idW — remaining goes negative.
	// The function should still set widths[0] = 24 without panicking.
	cols := []string{"_id", "name"}
	totalW := 10

	widths := distributeWidths(cols, totalW)

	if len(widths) != 2 {
		t.Fatalf("expected 2 widths, got %d", len(widths))
	}
	const idW = 24
	if widths[0] != idW {
		t.Errorf("widths[0] = %d; want %d even on narrow totalW", widths[0], idW)
	}
	// When remaining <= 0 the branch is skipped, widths[1] stays zero.
	if widths[1] != 0 {
		t.Errorf("widths[1] = %d; want 0 on narrow totalW", widths[1])
	}
}

func TestDistributeWidths_IDAlwaysFirst(t *testing.T) {
	// Regardless of column names, widths[0] always corresponds to _id column
	// and gets 24.
	cols := []string{"_id", "x", "y", "z"}
	widths := distributeWidths(cols, 150)

	const idW = 24
	if widths[0] != idW {
		t.Errorf("widths[0] = %d; want %d", widths[0], idW)
	}
}

func TestDistributeWidths_ReturnsCorrectSliceLength(t *testing.T) {
	for n := 1; n <= 6; n++ {
		cols := make([]string, n)
		for i := range cols {
			cols[i] = "col"
		}
		widths := distributeWidths(cols, 200)
		if len(widths) != n {
			t.Errorf("n=%d: len(widths) = %d; want %d", n, len(widths), n)
		}
	}
}
