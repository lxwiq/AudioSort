package tui

import (
	"testing"

	"audiosort/internal/config"
)

// TestConfigFieldLabelsMatchModalField guards the join key: every fieldSpec
// label must equal the field name carried back in EditModalResult.Field,
// otherwise applyModalResult silently no-ops.
func TestConfigFieldLabelsMatchModalField(t *testing.T) {
	for _, f := range configFields() {
		modal := f.modal(config.DefaultConfig())
		if modal.field != f.label {
			t.Fatalf("field %q builds a modal whose field is %q; they must match", f.label, modal.field)
		}
	}
}

func TestConfigEditsStayOnWorkingCopyUntilSave(t *testing.T) {
	shared := config.DefaultConfig()
	original := shared.ParallelWorkers

	m := NewConfigModel(shared)
	m.applyModalResult(EditModalResult{Confirmed: true, Field: "Workers", Value: 9})

	if m.working.ParallelWorkers != 9 {
		t.Fatalf("working copy should be updated to 9, got %d", m.working.ParallelWorkers)
	}
	if shared.ParallelWorkers != original {
		t.Fatalf("shared config must not change before save, got %d", shared.ParallelWorkers)
	}
}

func TestConfigCommitPropagatesToSharedPointer(t *testing.T) {
	shared := config.DefaultConfig()
	m := NewConfigModel(shared)
	m.working.ParallelWorkers = 7
	m.working.Sources = []string{"googlebooks"}

	// commit propagates in memory without touching disk.
	m.commit()

	if shared.ParallelWorkers != 7 {
		t.Fatalf("save should propagate workers to the shared pointer, got %d", shared.ParallelWorkers)
	}
	if len(shared.Sources) != 1 || shared.Sources[0] != "googlebooks" {
		t.Fatalf("save should propagate sources to the shared pointer, got %v", shared.Sources)
	}
	// The shared slice must not alias the working slice.
	m.working.Sources[0] = "mutated"
	if shared.Sources[0] == "mutated" {
		t.Fatalf("saved sources must be a copy, not alias the working slice")
	}
}
