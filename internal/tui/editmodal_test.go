package tui

import (
	"testing"
)

func TestEditModalNumberClampsAtConfirmation(t *testing.T) {
	m := NewNumberModal("Workers", "Edit Workers", 4, 1, 32)

	// Free typing builds a buffer well beyond the maximum...
	m, _ = m.updateNumber(key("9"))
	m, _ = m.updateNumber(key("9"))

	// ...but getValue (called at confirmation) clamps it to [min, max].
	if v, ok := m.getValue().(int); !ok || v != 32 {
		t.Fatalf("number should clamp to max 32 at confirmation, got %v", m.getValue())
	}
}

func TestEditModalMultiSelectReturnsSelectedInOrder(t *testing.T) {
	m := NewMultiSelectModal("Sources", "Edit Sources",
		[]string{"a", "b", "c"}, []string{"b", "a"})

	got, ok := m.getValue().([]string)
	if !ok {
		t.Fatalf("multiselect getValue should return []string, got %T", m.getValue())
	}
	want := []string{"b", "a"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected selected-in-order %v, got %v", want, got)
		}
	}
}

func TestEditModalConfirmCarriesFieldLabel(t *testing.T) {
	m := NewTextModal("Language", "Edit Language", "fr", false)
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("expected a command on enter")
	}
	result, ok := cmd().(EditModalResult)
	if !ok {
		t.Fatalf("enter should emit EditModalResult, got %T", cmd())
	}
	if !result.Confirmed || result.Field != "Language" {
		t.Fatalf("result should confirm field 'Language', got %#v", result)
	}
}
