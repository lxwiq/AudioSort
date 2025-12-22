# AudioSort

A modern CLI tool for organizing audiobook collections with automatic metadata fetching.

Built in Go with [Charm](https://charm.sh) libraries for a beautiful terminal experience.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)

---

## Features

- **Interactive TUI** - Beautiful terminal interface with colors, progress bars, and keyboard navigation
- **Parallel Processing** - Fast scanning and metadata fetching using goroutines
- **Multiple Metadata Sources** - Google Books, Open Library (more coming soon)
- **Smart Organization** - Auto-detects your existing folder structure
- **Player Presets** - Ready-to-use formats for AudiobookShelf, Plex, and more
- **Local Caching** - Metadata cached locally to avoid redundant API calls

---

## Installation

### From Source

Requires Go 1.21+

```bash
git clone https://github.com/lxwiq/AudioSort.git
cd AudioSort
go build -o audiosort ./cmd/audiosort
```

### From Releases

Download the latest binary from the [Releases](https://github.com/lxwiq/AudioSort/releases) page.

---

## Quick Start

### Interactive Mode (Recommended)

Simply run without arguments to launch the interactive menu:

```bash
./audiosort
```

This opens the main menu where you can:
- **Scan** - Organize your audiobooks
- **Search** - Look up book metadata
- **Config** - View/edit settings
- **Cache** - Manage cached data

### Command Line Mode

For scripting or quick actions:

```bash
# Scan a directory
./audiosort scan /path/to/audiobooks

# Scan with options
./audiosort scan /path/to/audiobooks -o ~/Organized --copy

# Search for metadata
./audiosort search "Harry Potter"

# View help
./audiosort --help
```

---

## Usage Guide

### Main Menu

When you run `./audiosort`, you'll see:

```
 █▀█ █░█ █▀▄ █ █▀█ █▀ █▀█ █▀█ ▀█▀
 █▀█ █▄█ █▄▀ █ █▄█ ▄█ █▄█ █▀▄ ░█░

 Organize your audiobook collection with style

 [s] Scan    - Scan and organize audiobooks
 [/] Search  - Search for book metadata
 [c] Config  - View and manage settings
 [x] Cache   - Manage metadata cache

 v1.0.0

 ↑/↓ navigate  •  enter select  •  q quit
```

**Navigation:**
- `↑/↓` or `j/k` - Move up/down
- `Enter` or `Space` - Select option
- `s`, `/`, `c`, `x` - Quick keys
- `q` - Quit

### Scanning Audiobooks

1. Select **Scan** from the menu (or press `s`)
2. Enter the path to your audiobooks folder
   - Use `~` for home directory (e.g., `~/Music/Audiobooks`)
   - Press `Tab` for autocomplete
3. Review the detected audiobooks
4. Select which ones to process
5. Watch the progress!

**Scan Screen Controls:**
- `Space` - Toggle selection
- `a` - Select/deselect all
- `Enter` - Start processing
- `s` - Skip selected
- `?` - Show help
- `q` - Quit

### Searching Metadata

1. Select **Search** from the menu (or press `/`)
2. Enter your search query (title, author, ISBN...)
3. Browse results from multiple sources
4. Press `Enter` on a result to see full details

### Configuration

Your settings are stored in `~/.config/audiosort/config.yaml`

| Setting | Description | Default |
|---------|-------------|---------|
| `sources` | Metadata sources to query | googlebooks, openlibrary |
| `output_format` | Output format preset | audiobookshelf |
| `default_output` | Default destination folder | (none) |
| `copy_mode` | Copy files instead of moving | false |
| `parallel_workers` | Number of parallel workers | 4 |
| `skip_existing` | Skip already processed books | true |
| `preferred_language` | Preferred metadata language | en |

Example configuration:

```yaml
sources:
  - googlebooks
  - openlibrary

output_format: audiobookshelf
default_output: ~/Audiobooks
copy_mode: false
parallel_workers: 4
skip_existing: true
preferred_language: fr
```

---

## Output Formats

AudioSort can generate different output formats depending on your player:

| Format | Files Generated | Best For |
|--------|----------------|----------|
| `audiobookshelf` | `metadata.opf` + `cover.jpg` | AudiobookShelf |
| `plex` | `cover.jpg` | Plex Media Server |
| `json` | `metadata.json` + `cover.jpg` | Custom integrations |
| `all` | All of the above | Maximum compatibility |

### Folder Structure

AudioSort organizes your books into this structure:

```
Audiobooks/
├── Author Name/
│   ├── Book Title/
│   │   ├── audiofile.mp3
│   │   ├── metadata.opf
│   │   └── cover.jpg
│   └── Another Book/
│       └── ...
└── Another Author/
    └── ...
```

---

## Supported Formats

### Audio Files

AudioSort detects these audio formats:
- `.mp3`
- `.m4a`
- `.m4b`
- `.flac`
- `.ogg`
- `.wma`

### Metadata Sources

| Source | Coverage | Best For |
|--------|----------|----------|
| **Google Books** | Worldwide | General books, international |
| **Open Library** | 20M+ books | Open source alternative |

---

## Command Reference

```
Usage:
  audiosort [command]

Available Commands:
  scan        Scan and organize audiobooks
  search      Search for book metadata
  config      Manage configuration
  cache       Manage metadata cache
  help        Help about any command

Flags:
      --config string   Config file (default: ~/.config/audiosort/config.yaml)
  -h, --help            Help for audiosort
  -v, --verbose         Verbose output

Use "audiosort [command] --help" for more information about a command.
```

### Scan Command

```bash
audiosort scan [path] [flags]

Flags:
  -o, --output string   Output directory
      --format string   Output format (audiobookshelf|plex|json|all)
      --copy            Copy files instead of moving
      --workers int     Number of parallel workers (default 4)
```

---

## How It Works

### Processing Pipeline

```
┌─────────┐    ┌─────────┐    ┌──────────┐    ┌───────────┐
│  Scan   │───>│  Fetch  │───>│  Detect  │───>│  Organize │
│ folders │    │ metadata│    │  pattern │    │   files   │
└─────────┘    └─────────┘    └──────────┘    └───────────┘
     │              │              │               │
     v              v              v               v
  Find audio    Query APIs    Match existing   Move/copy +
  files in      in parallel   folder structure generate
  directories   + use cache                    metadata
```

### Caching

- Metadata is cached locally in `~/.cache/audiosort/`
- Cache expires after 30 days
- Use the **Cache** menu to clear if needed

---

## Troubleshooting

### "Path not found"

Make sure the path exists and is a directory. Use `~` for home:
```bash
# Good
~/Music/Audiobooks
/Users/me/Audiobooks

# Bad
Music/Audiobooks  # (relative path may not work)
```

### "No audiobooks found"

AudioSort looks for folders containing audio files. Make sure your structure is:
```
Audiobooks/
├── Book 1/
│   └── audio.mp3    # Audio file inside folder
└── Book 2/
    └── audio.m4b
```

Not:
```
Audiobooks/
├── audio1.mp3       # Files directly in root
└── audio2.mp3
```

### Metadata not found

Try a more specific search query:
- Include author name: `"Harry Potter Rowling"`
- Use ISBN if available
- Try alternative title spellings

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

## Acknowledgments

- [Charm](https://charm.sh) - Beautiful TUI libraries for Go
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [BoltDB](https://github.com/etcd-io/bbolt) - Local caching
