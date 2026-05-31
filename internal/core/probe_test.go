package core

import (
	"testing"

	"audiosort/pkg/models"
)

func TestInferQueryPrefersProbeTags(t *testing.T) {
	book := &models.Audiobook{
		Path:  "/audiobooks/Some_Messy_Folder [128kbps] mp3",
		Probe: &models.Probe{Title: "Dune", Author: "Frank Herbert"},
	}
	if q := inferQuery(book); q != "Frank Herbert Dune" {
		t.Fatalf("expected tag-based query %q, got %q", "Frank Herbert Dune", q)
	}
}

func TestInferQueryFallsBackToFolderName(t *testing.T) {
	book := &models.Audiobook{Path: "/audiobooks/Dune"}
	if q := inferQuery(book); q != "Dune" {
		t.Fatalf("expected folder fallback %q, got %q", "Dune", q)
	}
}

func TestInferQueryProbeTitleOnly(t *testing.T) {
	book := &models.Audiobook{
		Path:  "/x/messy",
		Probe: &models.Probe{Title: "1984"},
	}
	if q := inferQuery(book); q != "1984" {
		t.Fatalf("expected %q, got %q", "1984", q)
	}
}
