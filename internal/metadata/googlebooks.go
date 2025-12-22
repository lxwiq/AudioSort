package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"audiosort/pkg/models"
)

// GoogleBooks implements MetadataSource for Google Books API
type GoogleBooks struct {
	client *http.Client
}

// NewGoogleBooks creates a new Google Books metadata source
func NewGoogleBooks(client *http.Client) *GoogleBooks {
	return &GoogleBooks{client: client}
}

// Name returns the source identifier
func (g *GoogleBooks) Name() string {
	return "googlebooks"
}

// Priority returns the source priority (1 = highest)
func (g *GoogleBooks) Priority() int {
	return 1
}

// Supports checks if the source supports the given language
func (g *GoogleBooks) Supports(lang string) bool {
	return true // Google Books supports all languages
}

// Search queries Google Books API for metadata
func (g *GoogleBooks) Search(ctx context.Context, query string) ([]models.BookMetadata, error) {
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=5",
		url.QueryEscape(query),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			VolumeInfo struct {
				Title         string   `json:"title"`
				Authors       []string `json:"authors"`
				Publisher     string   `json:"publisher"`
				PublishedDate string   `json:"publishedDate"`
				Description   string   `json:"description"`
				Language      string   `json:"language"`
				Categories    []string `json:"categories"`
				ImageLinks    struct {
					Thumbnail string `json:"thumbnail"`
					Large     string `json:"large"`
				} `json:"imageLinks"`
				IndustryIdentifiers []struct {
					Type       string `json:"type"`
					Identifier string `json:"identifier"`
				} `json:"industryIdentifiers"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var books []models.BookMetadata
	for _, item := range result.Items {
		v := item.VolumeInfo
		book := models.BookMetadata{
			Title:       v.Title,
			Authors:     v.Authors,
			Publisher:   v.Publisher,
			PublishYear: extractYear(v.PublishedDate),
			Summary:     v.Description,
			Language:    v.Language,
			Genres:      v.Categories,
			Source:      "googlebooks",
		}

		if v.ImageLinks.Large != "" {
			book.CoverURL = v.ImageLinks.Large
		} else {
			book.CoverURL = v.ImageLinks.Thumbnail
		}

		for _, id := range v.IndustryIdentifiers {
			if id.Type == "ISBN_13" {
				book.ISBN = id.Identifier
				break
			}
		}

		books = append(books, book)
	}

	return books, nil
}

// extractYear extracts the year from a date string
func extractYear(date string) string {
	re := regexp.MustCompile(`\d{4}`)
	match := re.FindString(date)
	return match
}
