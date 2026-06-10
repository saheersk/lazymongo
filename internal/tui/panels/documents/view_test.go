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
	// Single column always gets the full width.
	if widths[0] != totalW {
		t.Errorf("widths[0] = %d; want %d (totalW)", widths[0], totalW)
	}
}

func TestDistributeWidths_TwoCols_IDGetsCompact(t *testing.T) {
	cols := []string{"_id", "name"}
	totalW := 100

	widths := distributeWidths(cols, totalW)

	if len(widths) != 2 {
		t.Fatalf("expected 2 widths, got %d", len(widths))
	}

	const idColW = 10
	if widths[0] != idColW {
		t.Errorf("widths[0] (_id) = %d; want %d", widths[0], idColW)
	}

	// each = (100 - 10 - 1) / 1 = 89
	expected := (totalW - idColW - 1) / 1
	if widths[1] != expected {
		t.Errorf("widths[1] = %d; want %d", widths[1], expected)
	}
}

func TestDistributeWidths_ThreeCols_IDGetsCompact_OthersSplitEvenly(t *testing.T) {
	cols := []string{"_id", "name", "age"}
	totalW := 100

	widths := distributeWidths(cols, totalW)

	if len(widths) != 3 {
		t.Fatalf("expected 3 widths, got %d", len(widths))
	}

	const idColW = 10
	if widths[0] != idColW {
		t.Errorf("widths[0] (_id) = %d; want %d", widths[0], idColW)
	}

	// nExtra=2: each = (100 - 10 - 2) / 2 = 44
	each := (totalW - idColW - 2) / 2
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

func TestDistributeWidths_VeryNarrow_IDGetsTotalWidth(t *testing.T) {
	// When the panel is too narrow for even a minimal extra column,
	// _id gets the full totalW and the extra column is hidden (0).
	cols := []string{"_id", "name"}
	totalW := 10

	widths := distributeWidths(cols, totalW)

	if len(widths) != 2 {
		t.Fatalf("expected 2 widths, got %d", len(widths))
	}
	if widths[0] != totalW {
		t.Errorf("widths[0] = %d; want %d (totalW) when too narrow", widths[0], totalW)
	}
	if widths[1] != 0 {
		t.Errorf("widths[1] = %d; want 0 when too narrow", widths[1])
	}
}

func TestDistributeWidths_IDAlwaysFirst(t *testing.T) {
	// widths[0] always corresponds to the _id column and gets idColW.
	cols := []string{"_id", "x", "y", "z"}
	widths := distributeWidths(cols, 150)

	const idColW = 10
	if widths[0] != idColW {
		t.Errorf("widths[0] = %d; want %d", widths[0], idColW)
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

func TestDistributeWidths_TooManyColsHideExtra(t *testing.T) {
	// With a narrow panel and many columns, only the first few should be visible.
	cols := []string{"_id", "a", "b", "c", "d", "e", "f"}
	totalW := 32 // narrow: idColW(10) + sep + col(10) + sep + col(10) = 32

	widths := distributeWidths(cols, totalW)

	if len(widths) != len(cols) {
		t.Fatalf("len(widths) = %d; want %d", len(widths), len(cols))
	}
	// widths[0] must be 10 (idColW)
	if widths[0] != 10 {
		t.Errorf("widths[0] = %d; want 10", widths[0])
	}
	// At least one extra column should be visible (non-zero).
	anyVisible := false
	for _, w := range widths[1:] {
		if w > 0 {
			anyVisible = true
		}
	}
	if !anyVisible {
		t.Errorf("expected at least one non-_id column to be visible")
	}
	// Columns beyond what fits should be hidden (width 0).
	// With totalW=32: nExtra=2 fits (10+1+10+1+10=32), so widths[3..6] should be 0.
	for i := 3; i < len(widths); i++ {
		if widths[i] != 0 {
			t.Errorf("widths[%d] = %d; want 0 (hidden column)", i, widths[i])
		}
	}
}
