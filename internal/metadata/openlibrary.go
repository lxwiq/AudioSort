package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"audiosort/pkg/models"
)

// OpenLibrary implements MetadataSource for Open Library API
type OpenLibrary struct {
	client *http.Client
}

// NewOpenLibrary creates a new Open Library metadata source
func NewOpenLibrary(client *http.Client) *OpenLibrary {
	return &OpenLibrary{client: client}
}

// Name returns the source identifier
func (o *OpenLibrary) Name() string {
	return "openlibrary"
}

// Priority returns the source priority (2 = fallback)
func (o *OpenLibrary) Priority() int {
	return 2
}

// Supports checks if the source supports the given language
func (o *OpenLibrary) Supports(lang string) bool {
	return true // Open Library supports all languages
}

// Search queries Open Library API for metadata
func (o *OpenLibrary) Search(ctx context.Context, query string) ([]models.BookMetadata, error) {
	apiURL := fmt.Sprintf(
		"https://openlibrary.org/search.json?q=%s&limit=5",
		url.QueryEscape(query),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Docs []struct {
			Title            string   `json:"title"`
			AuthorName       []string `json:"author_name"`
			Publisher        []string `json:"publisher"`
			FirstPublishYear int      `json:"first_publish_year"`
			Language         []string `json:"language"`
			Subject          []string `json:"subject"`
			CoverI           int      `json:"cover_i"`
			ISBN             []string `json:"isbn"`
		} `json:"docs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var books []models.BookMetadata
	for _, doc := range result.Docs {
		book := models.BookMetadata{
			Title:   doc.Title,
			Authors: doc.AuthorName,
			Source:  "openlibrary",
		}

		if len(doc.Publisher) > 0 {
			book.Publisher = doc.Publisher[0]
		}
		if doc.FirstPublishYear > 0 {
			book.PublishYear = fmt.Sprintf("%d", doc.FirstPublishYear)
		}
		if len(doc.Language) > 0 {
			book.Language = doc.Language[0]
		}
		if len(doc.Subject) > 0 {
			limit := 5
			if len(doc.Subject) < limit {
				limit = len(doc.Subject)
			}
			book.Genres = doc.Subject[:limit]
		}
		if doc.CoverI > 0 {
			book.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", doc.CoverI)
		}
		if len(doc.ISBN) > 0 {
			book.ISBN = doc.ISBN[0]
		}

		books = append(books, book)
	}

	return books, nil
}
