package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"audiosort/pkg/models"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// Audible implements MetadataSource against the Audible catalog API. Unlike the
// book-oriented sources it returns audiobook-specific fields — most importantly
// narrators and series position — plus the ASIN.
type Audible struct {
	client *http.Client
	region string // Audible TLD, e.g. "com", "fr", "de"
}

// NewAudible creates a new Audible metadata source for the given region TLD
// (defaults to "com").
func NewAudible(client *http.Client, region string) *Audible {
	if client == nil {
		client = &http.Client{}
	}
	if region == "" {
		region = "com"
	}
	return &Audible{client: client, region: region}
}

// Name returns the source identifier
func (a *Audible) Name() string { return "audible" }

// Priority returns the source priority (0 = highest: best for audiobooks)
func (a *Audible) Priority() int { return 0 }

// Supports checks if the source supports the given language
func (a *Audible) Supports(lang string) bool { return true }

// Search queries the Audible catalog for audiobook metadata.
func (a *Audible) Search(ctx context.Context, query string) ([]models.BookMetadata, error) {
	params := url.Values{}
	params.Set("keywords", query)
	params.Set("num_results", "10")
	params.Set("products_sort_by", "Relevance")
	params.Set("response_groups", "contributors,product_desc,series,media,product_attrs")

	apiURL := fmt.Sprintf("https://api.audible.%s/1.0/catalog/products?%s", a.region, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AudioSort/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audible API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseAudibleResponse(body)
}

// audibleProduct mirrors the subset of the Audible catalog product we use.
type audibleProduct struct {
	ASIN          string            `json:"asin"`
	Title         string            `json:"title"`
	Authors       []audibleName     `json:"authors"`
	Narrators     []audibleName     `json:"narrators"`
	Series        []audibleSeries   `json:"series"`
	PublisherName string            `json:"publisher_name"`
	ReleaseDate   string            `json:"release_date"`
	Language      string            `json:"language"`
	Summary       string            `json:"merchandising_summary"`
	ProductImages map[string]string `json:"product_images"`
}

type audibleName struct {
	Name string `json:"name"`
}

type audibleSeries struct {
	Title    string `json:"title"`
	Sequence string `json:"sequence"`
}

// parseAudibleResponse maps an Audible catalog JSON payload to BookMetadata.
// It is split out from Search so it can be unit-tested without a network call.
func parseAudibleResponse(body []byte) ([]models.BookMetadata, error) {
	var result struct {
		Products []audibleProduct `json:"products"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var books []models.BookMetadata
	for _, p := range result.Products {
		book := models.BookMetadata{
			Title:       p.Title,
			Authors:     names(p.Authors),
			Narrators:   names(p.Narrators),
			Publisher:   p.PublisherName,
			PublishYear: extractYear(p.ReleaseDate),
			Language:    p.Language,
			Summary:     stripHTML(p.Summary),
			CoverURL:    largestImage(p.ProductImages),
			ASIN:        p.ASIN,
			Source:      "audible",
		}
		if len(p.Series) > 0 {
			book.Series = p.Series[0].Title
			book.SeriesPosition = p.Series[0].Sequence
		}
		books = append(books, book)
	}
	return books, nil
}

func names(list []audibleName) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, n := range list {
		if n.Name != "" {
			out = append(out, n.Name)
		}
	}
	return out
}

// largestImage returns the image URL with the largest numeric size key.
func largestImage(images map[string]string) string {
	if len(images) == 0 {
		return ""
	}
	sizes := make([]int, 0, len(images))
	for k := range images {
		if n, err := strconv.Atoi(k); err == nil {
			sizes = append(sizes, n)
		}
	}
	if len(sizes) == 0 {
		// No numeric keys; return any value deterministically.
		keys := make([]string, 0, len(images))
		for k := range images {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return images[keys[0]]
	}
	sort.Ints(sizes)
	return images[strconv.Itoa(sizes[len(sizes)-1])]
}

// stripHTML removes HTML tags and unescapes entities from a summary string.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}
