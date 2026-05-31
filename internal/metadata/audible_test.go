package metadata

import "testing"

const audibleFixture = `{
  "products": [
    {
      "asin": "B017V4IM1G",
      "title": "Dune",
      "authors": [{"name": "Frank Herbert"}],
      "narrators": [{"name": "Scott Brick"}, {"name": "Orlagh Cassidy"}],
      "series": [{"title": "Dune", "sequence": "1"}],
      "publisher_name": "Macmillan Audio",
      "release_date": "2007-08-07",
      "language": "english",
      "merchandising_summary": "<p>Set on the desert planet <b>Arrakis</b>.</p>",
      "product_images": {"500": "https://img/500.jpg", "1024": "https://img/1024.jpg"}
    }
  ]
}`

func TestParseAudibleResponse(t *testing.T) {
	books, err := parseAudibleResponse([]byte(audibleFixture))
	if err != nil {
		t.Fatalf("parseAudibleResponse error: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	b := books[0]

	if b.Title != "Dune" {
		t.Errorf("title = %q", b.Title)
	}
	if len(b.Authors) != 1 || b.Authors[0] != "Frank Herbert" {
		t.Errorf("authors = %v", b.Authors)
	}
	if len(b.Narrators) != 2 || b.Narrators[0] != "Scott Brick" {
		t.Errorf("narrators = %v (audiobook-specific field must be populated)", b.Narrators)
	}
	if b.Series != "Dune" || b.SeriesPosition != "1" {
		t.Errorf("series = %q #%q", b.Series, b.SeriesPosition)
	}
	if b.PublishYear != "2007" {
		t.Errorf("publish year = %q", b.PublishYear)
	}
	if b.ASIN != "B017V4IM1G" {
		t.Errorf("asin = %q", b.ASIN)
	}
	if b.CoverURL != "https://img/1024.jpg" {
		t.Errorf("cover should be the largest image, got %q", b.CoverURL)
	}
	if b.Summary != "Set on the desert planet Arrakis." {
		t.Errorf("summary HTML should be stripped, got %q", b.Summary)
	}
	if b.Source != "audible" {
		t.Errorf("source = %q", b.Source)
	}
}
