package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"audiosort/pkg/models"
)

const bookInfoBaseURL = "https://api.bookinfo.pro"

// BookInfo implements MetadataSource for BookInfo.pro API (GoodReads alternative)
type BookInfo struct {
	client *http.Client
}

// NewBookInfo creates a new BookInfo metadata source
func NewBookInfo(client *http.Client) *BookInfo {
	return &BookInfo{client: client}
}

// Name returns the source identifier
func (b *BookInfo) Name() string {
	return "bookinfo"
}

// Priority returns the source priority (1 = highest)
func (b *BookInfo) Priority() int {
	return 2 // Higher priority than Google Books for audiobook metadata
}

// Supports checks if the source supports the given language
func (b *BookInfo) Supports(lang string) bool {
	return true
}

// Search queries BookInfo API for metadata
func (b *BookInfo) Search(ctx context.Context, query string) ([]models.BookMetadata, error) {
	// Step 1: Search for works
	searchURL := fmt.Sprintf("%s/search?q=%s", bookInfoBaseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search request failed with status %d", resp.StatusCode)
	}

	var searchResults []struct {
		BookID int64 `json:"bookId"`
		WorkID int64 `json:"workId"`
		Author struct {
			ID int64 `json:"id"`
		} `json:"author"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResults); err != nil {
		return nil, err
	}

	if len(searchResults) == 0 {
		return nil, nil
	}

	// Step 2: Get work details for first result
	workID := searchResults[0].WorkID
	work, err := b.getWork(ctx, workID)
	if err != nil {
		return nil, err
	}

	return []models.BookMetadata{*work}, nil
}

// getWork fetches detailed work information
func (b *BookInfo) getWork(ctx context.Context, workID int64) (*models.BookMetadata, error) {
	workURL := fmt.Sprintf("%s/work/%d", bookInfoBaseURL, workID)

	req, err := http.NewRequestWithContext(ctx, "GET", workURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("work request failed with status %d", resp.StatusCode)
	}

	var work bookInfoWork
	if err := json.NewDecoder(resp.Body).Decode(&work); err != nil {
		return nil, err
	}

	return b.workToMetadata(&work), nil
}

// workToMetadata converts BookInfo work to BookMetadata
func (b *BookInfo) workToMetadata(work *bookInfoWork) *models.BookMetadata {
	meta := &models.BookMetadata{
		Title:       work.Title,
		PublishYear: extractYear(work.ReleaseDateRaw),
		Genres:      work.Genres,
		Source:      "bookinfo",
	}

	// Extract authors
	for _, author := range work.Authors {
		meta.Authors = append(meta.Authors, author.Name)
	}

	// Extract series info
	if len(work.Series) > 0 {
		series := work.Series[0]
		meta.Series = series.Title
		if len(series.LinkItems) > 0 {
			meta.SeriesPosition = series.LinkItems[0].PositionInSeries
		}
	}

	// Get best book info (description, cover, ISBN, etc.)
	if len(work.Books) > 0 {
		// Prefer English editions
		bestBook := work.Books[0]
		for _, book := range work.Books {
			if book.Language == "eng" && book.Description != "" {
				bestBook = book
				break
			}
		}

		meta.Summary = bestBook.Description
		meta.Publisher = bestBook.Publisher
		meta.CoverURL = bestBook.ImageURL
		meta.ISBN = bestBook.ISBN13
		meta.ASIN = bestBook.ASIN

		// Extract narrators from contributors
		for _, contrib := range bestBook.Contributors {
			if contrib.Role == "Narrator" {
				meta.Narrators = append(meta.Narrators, contrib.Name)
			}
		}
	}

	return meta
}

// BookInfo API response structures
type bookInfoWork struct {
	ForeignID      int64            `json:"ForeignId"`
	Title          string           `json:"Title"`
	FullTitle      string           `json:"FullTitle"`
	ReleaseDate    string           `json:"ReleaseDate"`
	ReleaseDateRaw string           `json:"ReleaseDateRaw"`
	Genres         []string         `json:"Genres"`
	Books          []bookInfoBook   `json:"Books"`
	Series         []bookInfoSeries `json:"Series"`
	Authors        []bookInfoAuthor `json:"Authors"`
}

type bookInfoBook struct {
	ForeignID    int64                  `json:"ForeignId"`
	ASIN         string                 `json:"Asin"`
	Description  string                 `json:"Description"`
	ISBN13       string                 `json:"Isbn13"`
	Title        string                 `json:"Title"`
	Language     string                 `json:"Language"`
	Format       string                 `json:"Format"`
	Publisher    string                 `json:"Publisher"`
	ImageURL     string                 `json:"ImageUrl"`
	NumPages     int                    `json:"NumPages"`
	Contributors []bookInfoContributor  `json:"Contributors"`
}

type bookInfoContributor struct {
	ForeignID int64  `json:"ForeignId"`
	Role      string `json:"Role"`
	Name      string `json:"Name,omitempty"`
}

type bookInfoSeries struct {
	ForeignID int64              `json:"ForeignId"`
	Title     string             `json:"Title"`
	LinkItems []bookInfoSeriesLink `json:"LinkItems"`
}

type bookInfoSeriesLink struct {
	ForeignWorkID    int64  `json:"ForeignWorkId"`
	PositionInSeries string `json:"PositionInSeries"`
	SeriesPosition   int    `json:"SeriesPosition"`
}

type bookInfoAuthor struct {
	ForeignID   int64  `json:"ForeignId"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	ImageURL    string `json:"ImageUrl"`
}
