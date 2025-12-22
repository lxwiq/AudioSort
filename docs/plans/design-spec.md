# AudioSort Go - Design Specification

This document provides the complete technical specification for implementing AudioSort in Go.

## Project Structure

```
audiosort/
├── cmd/
│   └── audiosort/
│       └── main.go              # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go              # Root Cobra command
│   │   ├── scan.go              # audiosort scan
│   │   ├── search.go            # audiosort search
│   │   ├── ui.go                # audiosort ui
│   │   └── config.go            # audiosort config
│   ├── tui/
│   │   ├── app.go               # Bubble Tea application
│   │   ├── models/              # View states
│   │   ├── views/               # View rendering
│   │   └── components/          # Reusable components
│   ├── core/
│   │   ├── scanner.go           # Audiobook detection
│   │   ├── organizer.go         # Organization logic
│   │   ├── pipeline.go          # Parallel processing pipeline
│   │   └── detector.go          # Destination pattern detection
│   ├── metadata/
│   │   ├── source.go            # MetadataSource interface
│   │   ├── googlebooks.go       # Google Books implementation
│   │   ├── openlibrary.go       # Open Library implementation
│   │   ├── audible.go           # Audible implementation
│   │   ├── bnf.go               # BnF implementation
│   │   └── fetcher.go           # Parallel fetch orchestration
│   ├── output/
│   │   ├── writer.go            # OutputWriter interface
│   │   ├── opf.go               # .opf generation
│   │   ├── json.go              # .json generation
│   │   └── cover.go             # Cover download
│   ├── cache/
│   │   └── store.go             # Metadata cache + state
│   └── config/
│       └── config.go            # Configuration management
├── pkg/
│   └── models/
│       ├── book.go              # BookMetadata, Audiobook
│       └── errors.go            # Error types
└── go.mod
```

## Dependencies

```
module audiosort

go 1.21

require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/bubbles v0.18.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/spf13/cobra v1.8.0
    go.etcd.io/bbolt v1.3.8
    golang.org/x/sync v0.6.0
    gopkg.in/yaml.v3 v3.0.1
)
```

## Data Models

### pkg/models/book.go

```go
package models

import "time"

type BookMetadata struct {
    Title          string   `json:"title"`
    Authors        []string `json:"authors"`
    Narrators      []string `json:"narrators"`
    Series         string   `json:"series,omitempty"`
    SeriesPosition string   `json:"series_position,omitempty"`
    Summary        string   `json:"summary,omitempty"`
    Publisher      string   `json:"publisher,omitempty"`
    PublishYear    string   `json:"publish_year,omitempty"`
    Language       string   `json:"language,omitempty"`
    Genres         []string `json:"genres,omitempty"`
    ISBN           string   `json:"isbn,omitempty"`
    ASIN           string   `json:"asin,omitempty"`
    CoverURL       string   `json:"cover_url,omitempty"`
    SourceURL      string   `json:"source_url,omitempty"`
    Source         string   `json:"source"`
}

func (b BookMetadata) PrimaryAuthor() string {
    if len(b.Authors) > 0 {
        return b.Authors[0]
    }
    return "_unknown_"
}

type Audiobook struct {
    Path     string
    Files    []AudioFile
    Metadata *BookMetadata
    Status   Status
    Error    error
}

type AudioFile struct {
    Path     string
    Size     int64
    Duration time.Duration
}

type Status string

const (
    StatusPending    Status = "pending"
    StatusProcessing Status = "processing"
    StatusDone       Status = "done"
    StatusError      Status = "error"
    StatusSkipped    Status = "skipped"
)
```

### pkg/models/errors.go

```go
package models

import (
    "errors"
    "fmt"
    "time"
)

var (
    ErrNoMetadataFound   = errors.New("no metadata found for query")
    ErrSourceTimeout     = errors.New("metadata source timeout")
    ErrInvalidPath       = errors.New("invalid audiobook path")
    ErrDestinationExists = errors.New("destination already exists")
)

type Result struct {
    Audiobook *Audiobook
    Success   bool
    Error     error
    Duration  time.Duration
}

type Summary struct {
    Total     int
    Processed int
    Skipped   int
    Errors    int
    Duration  time.Duration
    Failures  []Result
}

func (s Summary) String() string {
    return fmt.Sprintf("✓ %d processed | ○ %d skipped | ✗ %d errors",
        s.Processed, s.Skipped, s.Errors)
}
```

## Metadata Source Interface

### internal/metadata/source.go

```go
package metadata

import (
    "context"
    "audiosort/pkg/models"
)

type MetadataSource interface {
    Name() string
    Search(ctx context.Context, query string) ([]models.BookMetadata, error)
    Priority() int
    Supports(language string) bool
}
```

### internal/metadata/fetcher.go

```go
package metadata

import (
    "context"
    "net/http"
    "sync"
    "time"

    "audiosort/internal/cache"
    "audiosort/pkg/models"
)

type Fetcher struct {
    sources []MetadataSource
    cache   *cache.Store
    client  *http.Client
}

func NewFetcher(sources []MetadataSource, cache *cache.Store) *Fetcher {
    return &Fetcher{
        sources: sources,
        cache:   cache,
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

func (f *Fetcher) Fetch(ctx context.Context, query string) (*models.BookMetadata, error) {
    // 1. Check cache first
    if cached, ok := f.cache.GetMetadata(query); ok {
        return &cached.Metadata, nil
    }

    // 2. Fetch from all sources in parallel
    results := make(chan *models.BookMetadata, len(f.sources))
    var wg sync.WaitGroup

    for _, source := range f.sources {
        wg.Add(1)
        go func(s MetadataSource) {
            defer wg.Done()
            books, err := s.Search(ctx, query)
            if err == nil && len(books) > 0 {
                results <- &books[0]
            }
        }(source)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    // 3. Return first result (sources are ordered by priority)
    for result := range results {
        f.cache.SetMetadata(query, *result)
        return result, nil
    }

    return nil, models.ErrNoMetadataFound
}
```

## Parallel Processing Pipeline

### internal/core/pipeline.go

```go
package core

import (
    "context"

    "golang.org/x/sync/errgroup"
    "audiosort/pkg/models"
    "audiosort/internal/metadata"
)

type Pipeline struct {
    scanner   *Scanner
    fetcher   *metadata.Fetcher
    organizer *Organizer
    reporter  *Reporter
    workers   int
}

func (p *Pipeline) Run(ctx context.Context, inputPath string) (*models.Summary, error) {
    books := make(chan *models.Audiobook, 100)
    enriched := make(chan *models.Audiobook, 100)
    results := make(chan models.Result, 100)

    g, ctx := errgroup.WithContext(ctx)

    // Stage 1: Scan for audiobooks
    g.Go(func() error {
        defer close(books)
        return p.scanner.Scan(ctx, inputPath, books)
    })

    // Stage 2: Fetch metadata (worker pool)
    g.Go(func() error {
        defer close(enriched)
        return p.fetcher.Process(ctx, books, enriched, p.workers)
    })

    // Stage 3: Organize files (worker pool)
    g.Go(func() error {
        defer close(results)
        return p.organizer.Process(ctx, enriched, results, p.workers)
    })

    // Stage 4: Collect results
    var summary *models.Summary
    g.Go(func() error {
        summary = p.reporter.Collect(ctx, results)
        return nil
    })

    if err := g.Wait(); err != nil {
        return nil, err
    }

    return summary, nil
}
```

## Scanner Implementation

### internal/core/scanner.go

```go
package core

import (
    "context"
    "os"
    "path/filepath"

    "audiosort/pkg/models"
)

var audioExtensions = map[string]bool{
    ".mp3":  true,
    ".m4a":  true,
    ".m4b":  true,
    ".flac": true,
    ".ogg":  true,
    ".wma":  true,
}

type Scanner struct{}

func (s *Scanner) Scan(ctx context.Context, root string, out chan<- *models.Audiobook) error {
    return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return nil // Continue on error
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        if !d.IsDir() {
            return nil
        }

        // Check if directory contains audio files
        audioFiles := s.findAudioFiles(path)
        if len(audioFiles) == 0 {
            return nil
        }

        // Check if parent also has audio (avoid duplicates)
        parentAudio := s.findAudioFiles(filepath.Dir(path))
        if len(parentAudio) > 0 {
            return nil
        }

        out <- &models.Audiobook{
            Path:   path,
            Files:  audioFiles,
            Status: models.StatusPending,
        }

        return filepath.SkipDir
    })
}

func (s *Scanner) findAudioFiles(dir string) []models.AudioFile {
    var files []models.AudioFile

    entries, err := os.ReadDir(dir)
    if err != nil {
        return files
    }

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        ext := filepath.Ext(entry.Name())
        if audioExtensions[ext] {
            info, _ := entry.Info()
            files = append(files, models.AudioFile{
                Path: filepath.Join(dir, entry.Name()),
                Size: info.Size(),
            })
        }
    }

    return files
}
```

## Destination Pattern Detection

### internal/core/detector.go

```go
package core

import (
    "os"
    "path/filepath"
    "strings"

    "audiosort/pkg/models"
)

type PatternDetector struct{}

type Pattern struct {
    Template string // e.g., "{author}/{title}" or "{author}/{series}/{title}"
}

func (d *PatternDetector) Detect(destRoot string) Pattern {
    entries, err := os.ReadDir(destRoot)
    if err != nil || len(entries) == 0 {
        return Pattern{Template: "{author}/{title}"}
    }

    // Sample first few directories to detect pattern
    for _, entry := range entries[:min(5, len(entries))] {
        if !entry.IsDir() {
            continue
        }

        subEntries, _ := os.ReadDir(filepath.Join(destRoot, entry.Name()))
        for _, sub := range subEntries {
            if sub.IsDir() {
                return Pattern{Template: "{author}/{title}"}
            }
        }
    }

    return Pattern{Template: "{author}/{title}"}
}

func (d *PatternDetector) BuildPath(pattern Pattern, metadata *models.BookMetadata) string {
    result := pattern.Template
    result = strings.ReplaceAll(result, "{author}", sanitize(metadata.PrimaryAuthor()))
    result = strings.ReplaceAll(result, "{title}", sanitize(metadata.Title))
    result = strings.ReplaceAll(result, "{series}", sanitize(metadata.Series))
    return result
}

func sanitize(s string) string {
    replacer := strings.NewReplacer(
        "/", "-",
        "\\", "-",
        ":", "-",
        "*", "",
        "?", "",
        "\"", "",
        "<", "",
        ">", "",
        "|", "",
    )
    return strings.TrimSpace(replacer.Replace(s))
}
```

## Cache Implementation

### internal/cache/store.go

```go
package cache

import (
    "encoding/json"
    "time"

    "go.etcd.io/bbolt"
    "audiosort/pkg/models"
)

type Store struct {
    db *bbolt.DB
}

type CachedBook struct {
    Query     string              `json:"query"`
    Metadata  models.BookMetadata `json:"metadata"`
    FetchedAt time.Time           `json:"fetched_at"`
}

type ProcessedBook struct {
    SourcePath  string    `json:"source_path"`
    DestPath    string    `json:"dest_path"`
    ProcessedAt time.Time `json:"processed_at"`
    Checksum    string    `json:"checksum"`
}

var (
    metadataBucket  = []byte("metadata")
    processedBucket = []byte("processed")
)

func NewStore(path string) (*Store, error) {
    db, err := bbolt.Open(path, 0600, nil)
    if err != nil {
        return nil, err
    }

    db.Update(func(tx *bbolt.Tx) error {
        tx.CreateBucketIfNotExists(metadataBucket)
        tx.CreateBucketIfNotExists(processedBucket)
        return nil
    })

    return &Store{db: db}, nil
}

func (s *Store) GetMetadata(query string) (*CachedBook, bool) {
    var cached CachedBook

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(metadataBucket)
        data := b.Get([]byte(query))
        if data == nil {
            return nil
        }
        return json.Unmarshal(data, &cached)
    })

    if err != nil || cached.Query == "" {
        return nil, false
    }

    if time.Since(cached.FetchedAt) > 30*24*time.Hour {
        return nil, false
    }

    return &cached, true
}

func (s *Store) SetMetadata(query string, metadata models.BookMetadata) error {
    cached := CachedBook{
        Query:     query,
        Metadata:  metadata,
        FetchedAt: time.Now(),
    }

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(metadataBucket)
        data, err := json.Marshal(cached)
        if err != nil {
            return err
        }
        return b.Put([]byte(query), data)
    })
}

func (s *Store) IsProcessed(path string) bool {
    var exists bool

    s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(processedBucket)
        exists = b.Get([]byte(path)) != nil
        return nil
    })

    return exists
}

func (s *Store) MarkProcessed(sourcePath, destPath string) error {
    processed := ProcessedBook{
        SourcePath:  sourcePath,
        DestPath:    destPath,
        ProcessedAt: time.Now(),
    }

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(processedBucket)
        data, err := json.Marshal(processed)
        if err != nil {
            return err
        }
        return b.Put([]byte(sourcePath), data)
    })
}

func (s *Store) Close() error {
    return s.db.Close()
}
```

## Configuration

### internal/config/config.go

```go
package config

import (
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Sources           []string `yaml:"sources"`
    OutputFormat      string   `yaml:"output_format"`
    DefaultOutput     string   `yaml:"default_output"`
    CopyMode          bool     `yaml:"copy_mode"`
    ParallelWorkers   int      `yaml:"parallel_workers"`
    SkipExisting      bool     `yaml:"skip_existing"`
    PreferredLanguage string   `yaml:"preferred_language"`
}

func DefaultConfig() *Config {
    return &Config{
        Sources:           []string{"googlebooks", "openlibrary", "audible", "bnf"},
        OutputFormat:      "audiobookshelf",
        DefaultOutput:     "",
        CopyMode:          false,
        ParallelWorkers:   4,
        SkipExisting:      true,
        PreferredLanguage: "en",
    }
}

func configPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "audiosort", "config.yaml")
}

func Load() (*Config, error) {
    cfg := DefaultConfig()

    data, err := os.ReadFile(configPath())
    if err != nil {
        if os.IsNotExist(err) {
            return cfg, nil
        }
        return nil, err
    }

    if err := yaml.Unmarshal(data, cfg); err != nil {
        return nil, err
    }

    return cfg, nil
}

func (c *Config) Save() error {
    path := configPath()

    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }

    data, err := yaml.Marshal(c)
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}
```

## TUI Architecture

### internal/tui/app.go

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/progress"
    "github.com/charmbracelet/bubbles/table"

    "audiosort/pkg/models"
)

type AppState int

const (
    StateScanning AppState = iota
    StateFetching
    StateReview
    StateProcessing
    StateDone
)

type Model struct {
    state      AppState
    audiobooks []models.Audiobook
    selected   int

    spinner  spinner.Model
    progress progress.Model
    table    table.Model

    width, height int
    err error
}

func NewModel() Model {
    return Model{
        state:    StateScanning,
        spinner:  spinner.New(),
        progress: progress.New(),
    }
}

func (m Model) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil

    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "enter":
            if m.state == StateReview {
                return m, m.processSelected()
            }
        case "j", "down":
            if m.selected < len(m.audiobooks)-1 {
                m.selected++
            }
        case "k", "up":
            if m.selected > 0 {
                m.selected--
            }
        case "s":
            if m.state == StateReview {
                m.audiobooks[m.selected].Status = models.StatusSkipped
            }
        }

    case ScanCompleteMsg:
        m.audiobooks = msg.Books
        m.state = StateFetching
        return m, m.startFetching()

    case FetchCompleteMsg:
        m.state = StateReview
        return m, nil

    case ProcessCompleteMsg:
        m.state = StateDone
        return m, nil

    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }

    return m, nil
}

func (m Model) View() string {
    switch m.state {
    case StateScanning:
        return m.viewScanning()
    case StateFetching:
        return m.viewFetching()
    case StateReview:
        return m.viewReview()
    case StateProcessing:
        return m.viewProcessing()
    case StateDone:
        return m.viewDone()
    }
    return ""
}

type ScanCompleteMsg struct {
    Books []models.Audiobook
}

type FetchCompleteMsg struct{}

type ProcessCompleteMsg struct {
    Summary models.Summary
}
```

## CLI Commands

### cmd/audiosort/main.go

```go
package main

import (
    "os"
    "audiosort/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### internal/cli/root.go

```go
package cli

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "audiosort",
    Short: "Organize audiobook collections with automatic metadata",
    Long:  `AudioSort scans audiobook folders, fetches metadata from multiple sources,
and organizes files into a clean structure for players like AudiobookShelf.`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.AddCommand(scanCmd)
    rootCmd.AddCommand(uiCmd)
    rootCmd.AddCommand(searchCmd)
    rootCmd.AddCommand(configCmd)
    rootCmd.AddCommand(cacheCmd)
}
```

### internal/cli/scan.go

```go
package cli

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "audiosort/internal/config"
    "audiosort/internal/core"
)

var scanCmd = &cobra.Command{
    Use:   "scan [path]",
    Short: "Scan and organize audiobooks",
    Args:  cobra.ExactArgs(1),
    RunE:  runScan,
}

var (
    outputDir    string
    outputFormat string
    dryRun       bool
    force        bool
    copyMode     bool
)

func init() {
    scanCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory")
    scanCmd.Flags().StringVar(&outputFormat, "format", "audiobookshelf", "Output format")
    scanCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without making changes")
    scanCmd.Flags().BoolVar(&force, "force", false, "Re-process already cached books")
    scanCmd.Flags().BoolVar(&copyMode, "copy", false, "Copy files instead of moving")
}

func runScan(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    if outputDir == "" {
        outputDir = cfg.DefaultOutput
    }

    pipeline := core.NewPipeline(cfg)

    ctx := context.Background()
    summary, err := pipeline.Run(ctx, args[0])
    if err != nil {
        return err
    }

    fmt.Println(summary)
    return nil
}
```

## Metadata Sources

### Google Books (internal/metadata/googlebooks.go)

```go
package metadata

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"

    "audiosort/pkg/models"
)

type GoogleBooks struct {
    client *http.Client
}

func NewGoogleBooks(client *http.Client) *GoogleBooks {
    return &GoogleBooks{client: client}
}

func (g *GoogleBooks) Name() string     { return "googlebooks" }
func (g *GoogleBooks) Priority() int    { return 1 }
func (g *GoogleBooks) Supports(lang string) bool { return true }

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
```

### Open Library (internal/metadata/openlibrary.go)

```go
package metadata

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"

    "audiosort/pkg/models"
)

type OpenLibrary struct {
    client *http.Client
}

func NewOpenLibrary(client *http.Client) *OpenLibrary {
    return &OpenLibrary{client: client}
}

func (o *OpenLibrary) Name() string     { return "openlibrary" }
func (o *OpenLibrary) Priority() int    { return 2 }
func (o *OpenLibrary) Supports(lang string) bool { return true }

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
            book.Genres = doc.Subject[:min(5, len(doc.Subject))]
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
```

## Output Writers

### OPF Writer (internal/output/opf.go)

```go
package output

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "audiosort/pkg/models"
)

type OPFWriter struct{}

func (w *OPFWriter) Write(destPath string, metadata *models.BookMetadata) error {
    content := w.generate(metadata)
    opfPath := filepath.Join(destPath, "metadata.opf")
    return os.WriteFile(opfPath, []byte(content), 0644)
}

func (w *OPFWriter) generate(m *models.BookMetadata) string {
    var sb strings.Builder

    sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
    sb.WriteString("\n")
    sb.WriteString(`<package version="2.0" xmlns="http://www.idpf.org/2007/opf">`)
    sb.WriteString("\n  <metadata>\n")

    sb.WriteString(fmt.Sprintf("    <dc:title>%s</dc:title>\n", escapeXML(m.Title)))

    for _, author := range m.Authors {
        sb.WriteString(fmt.Sprintf("    <dc:creator>%s</dc:creator>\n", escapeXML(author)))
    }

    if m.Summary != "" {
        sb.WriteString(fmt.Sprintf("    <dc:description>%s</dc:description>\n", escapeXML(m.Summary)))
    }

    if m.Publisher != "" {
        sb.WriteString(fmt.Sprintf("    <dc:publisher>%s</dc:publisher>\n", escapeXML(m.Publisher)))
    }

    if m.PublishYear != "" {
        sb.WriteString(fmt.Sprintf("    <dc:date>%s</dc:date>\n", m.PublishYear))
    }

    if m.Language != "" {
        sb.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", m.Language))
    }

    if m.Series != "" {
        sb.WriteString(fmt.Sprintf("    <meta name=\"calibre:series\" content=\"%s\"/>\n", escapeXML(m.Series)))
        if m.SeriesPosition != "" {
            sb.WriteString(fmt.Sprintf("    <meta name=\"calibre:series_index\" content=\"%s\"/>\n", m.SeriesPosition))
        }
    }

    for _, narrator := range m.Narrators {
        sb.WriteString(fmt.Sprintf("    <meta name=\"narrator\" content=\"%s\"/>\n", escapeXML(narrator)))
    }

    sb.WriteString("  </metadata>\n</package>\n")

    return sb.String()
}

func escapeXML(s string) string {
    replacer := strings.NewReplacer(
        "&", "&amp;",
        "<", "&lt;",
        ">", "&gt;",
        "\"", "&quot;",
        "'", "&apos;",
    )
    return replacer.Replace(s)
}
```

## Implementation Order

1. **pkg/models** - Data structures
2. **internal/config** - Configuration loading
3. **internal/cache** - bbolt cache store
4. **internal/metadata/source.go** - Interface
5. **internal/metadata/googlebooks.go** - First source
6. **internal/metadata/fetcher.go** - Orchestration
7. **internal/core/scanner.go** - File scanning
8. **internal/core/detector.go** - Pattern detection
9. **internal/core/organizer.go** - File operations
10. **internal/core/pipeline.go** - Pipeline assembly
11. **internal/output/** - Writers
12. **internal/cli/** - Cobra commands
13. **internal/tui/** - Bubble Tea interface
14. **cmd/audiosort/main.go** - Entry point
