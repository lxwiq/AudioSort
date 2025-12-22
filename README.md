# AudioSort

A CLI tool for organizing audiobook collections with automatic metadata fetching.

Built in Go with [Charm](https://charm.sh) libraries for a modern terminal experience.

## Features

- **Parallel Processing** - Scan, fetch metadata, and organize files concurrently using goroutines
- **Multiple Metadata Sources** - Google Books, Open Library, Audible, BnF (French National Library)
- **Hybrid Interface** - Classic CLI for scripting + interactive TUI for manual review
- **Smart Organization** - Auto-detects and matches your existing folder structure
- **Player Presets** - Ready-to-use formats for AudiobookShelf, Plex, and others
- **Resilient** - Continues on errors, caches metadata, resumes intelligently

## Installation

Requires Go 1.21+

```bash
git clone https://github.com/lxwiq/AudioSort.git
cd AudioSort
go build -o audiosort ./cmd/audiosort
```

## Usage

### Scan and Organize

```bash
audiosort scan ./audiobooks                           # Basic scan
audiosort scan ./audiobooks -o ~/Audiobooks           # Custom output
audiosort scan ./audiobooks --format audiobookshelf   # Specific format
audiosort scan ./audiobooks --dry-run                 # Preview only
audiosort scan ./audiobooks --copy                    # Copy instead of move
```

### Interactive Mode

```bash
audiosort ui              # Launch TUI
audiosort ui ./audiobooks # Open TUI on specific folder
```

### Other Commands

```bash
audiosort search "Harry Potter"    # Search metadata only
audiosort config show              # View configuration
audiosort cache clear              # Clear metadata cache
```

## Output Formats

| Format | Generated Files | Target |
|--------|-----------------|--------|
| `audiobookshelf` | metadata.opf + cover.jpg | AudiobookShelf |
| `plex` | cover.jpg | Plex |
| `json` | metadata.json + cover.jpg | Custom/API |
| `all` | All files | Maximum compatibility |

## Configuration

Config file: `~/.config/audiosort/config.yaml`

```yaml
sources:
  - googlebooks
  - openlibrary
  - audible
  - bnf

output_format: audiobookshelf
default_output: ~/Audiobooks
copy_mode: false
parallel_workers: 4
skip_existing: true
preferred_language: fr
```

## How It Works

### Processing Pipeline

1. **Scan** - Recursively finds folders containing audio files (.mp3, .m4a, .m4b, .flac, .ogg, .wma)
2. **Fetch** - Queries metadata sources in parallel, uses cache for already-seen books
3. **Detect** - Analyzes destination folder to match existing organization pattern
4. **Organize** - Moves/copies files to structured folders with metadata files

### Metadata Sources

Sources are queried in priority order with parallel fetching:

1. **Google Books** - Large database, good international coverage
2. **Open Library** - Open source, 20M+ books
3. **Audible** - Best for audiobook-specific metadata (narrators, duration)
4. **BnF** - French National Library, excellent for French books

### Caching

- Metadata is cached locally for 30 days
- Processed books are tracked to avoid re-processing
- Use `--force` to ignore cache and re-process

### Error Handling

- Continues processing on individual book errors
- Summary at end shows: processed / skipped / errors
- Detailed error log available for troubleshooting

## Interactive TUI

The TUI mode provides:

- Real-time progress during scan and fetch
- Table view of detected audiobooks with metadata preview
- Keyboard navigation to review, edit, or skip books
- Confirmation before processing

Key bindings:
- `j/k` or arrows: Navigate
- `Enter`: Process selected
- `s`: Skip selected
- `q`: Quit

## License

MIT License
