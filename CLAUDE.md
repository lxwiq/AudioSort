# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 🎧 AudioSort 3.0 - Project Overview

AudioSort is a Python-based audiobook organization tool that automatically scans collections, fetches metadata from multiple sources, and organizes files into structured formats compatible with popular players like AudiobookShelf and SmartAudioBookPlayer.

## 📁 Project Structure

```
AudioSort/
├── AudioSort/                    # Main Python package
│   ├── __init__.py             # Package initialization (v1.0.0)
│   ├── __main__.py             # Entry point for `python -m AudioSort`
│   ├── cli.py                  # Command-line interface and orchestration
│   ├── models.py               # Data models (BookMetadata, ProcessingPlan)
│   ├── metadata_sources.py     # API integrations (Google Books, Open Library, etc.)
│   ├── actions.py              # Core file operations and metadata generation
│   └── search.py               # DuckDuckGo search functionality
├── audiosort.sh                # Comprehensive shell wrapper (recommended)
├── requirements.txt            # Dependencies (requests, beautifulsoup4, tinytag)
├── template.opf                # AudiobookShelf metadata template
├── README.md                   # Comprehensive user documentation
└── EXAMPLES.md                 # Detailed usage examples
```

## 🚀 Development Commands

### Environment Setup
```bash
# Create and activate virtual environment
python3 -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

### Running AudioSort
```bash
# Recommended: Use shell script
./audiosort.sh "audiobook_folder"

# Direct Python module usage
python -m AudioSort "audiobook_folder"

# Development/testing modes
./audiosort.sh --preview "test_folder"     # Preview mode
./audiosort.sh --debug --verbose "folder"   # Debug mode
./audiosort.sh --safe "folder"              # Copy mode (safe)
```

### Common Development Tasks
```bash
# Test with preview mode before actual processing
python -m AudioSort "test_folder" --dry-run --scan --auto

# Debug metadata fetching
python -m AudioSort "problem_folder" --debug --scan --auto

# Test specific functionality
python -m AudioSort "folder" --opf --cover --flatten --rename
```

## 🏗️ Architecture & Core Components

### Entry Points
1. **`audiosort.sh`** - Shell wrapper (recommended for users)
2. **`python -m AudioSort`** - Direct module execution
3. **`AudioSort.__main__.py`** → `cli.main()` - Program entry point

### Core Modules

#### `cli.py` - Main Orchestration
- **Purpose**: Command-line parsing, workflow coordination
- **Key Functions**: `main()`, argument parsing, mode handling
- **Size**: ~14KB - Largest module, contains primary business logic

#### `models.py` - Data Structures
- **`BookMetadata`**: Complete audiobook metadata (authors, series, etc.)
- **`ProcessingPlan`**: Configuration for file operations
- Uses dataclasses for structured data with sensible defaults

#### `metadata_sources.py` - API Integrations
- **Sources**: Google Books API, Open Library, Audible, Goodreads
- **Session Management**: Uses `requests.Session` for connection reuse
- **Fallback Strategy**: Multiple sources with intelligent fallback

#### `actions.py` - File Operations
- **Core Operations**: File moving/copying, metadata generation, folder creation
- **Conflict Management**: Handles existing folders and file overwrites
- **Output Formats**: Generates `.opf`, `.txt`, and cover art

#### `search.py` - Search Functionality
- **Purpose**: DuckDuckGo integration for finding metadata URLs
- **Usage**: Bridge between folder names and API sources

## 🎯 Key Features & Implementation

### Multi-Source Metadata Fetching
```python
# Order of operations in metadata_sources.py
1. DuckDuckGo search (Audible/Goodreads URLs)
2. Google Books API (primary source)
3. Open Library (fallback)
4. Smart folder name extraction
```

### Intelligent Series Grouping
- Automatically groups books from same series
- Merges new books into existing series folders
- Conflict detection and warnings before overwriting
- Located in `actions.py:prepare_output()`

### Supported Audio Formats
- `.mp3`, `.m4a`, `.m4b`, `.wma`, `.flac`, `.ogg`
- Detection via file extension scanning
- Metadata extraction using `tinytag`

## 🔧 Configuration & Options

### Shell Script Modes (audiosort.sh)
- **Auto Mode**: Complete automatic processing (default)
- **Preview Mode**: `--preview` - Test without modifications
- **Safe Mode**: `--safe` - Copy instead of move files
- **Basic Mode**: `--basic` - Simple organization without metadata

### Key CLI Flags
```bash
--scan          # Automatic subdirectory scanning
--auto          # Automatic metadata search
--flatten       # Flatten multi-disc folders
--rename        # Rename to indexed format
--opf           # Generate metadata.opf (AudiobookShelf)
--infotxt       # Generate info.txt (SmartAudioBookPlayer)
--cover         # Download cover art
--copy          # Copy instead of move
--dry-run       # Preview mode
-O "folder"     # Custom output directory
```

### Environment Variables
```bash
AUDIOSORT_OUTPUT="/path/to/output"  # Default output folder
AUDIOSORT_DEBUG=true                # Enable debug logging
```

## 🧪 Testing & Development

### No Formal Test Suite
- Project currently lacks automated tests
- Manual testing via preview mode recommended
- Use `--dry-run` extensively before real processing

### Development Workflow
1. **Always test with preview mode first**
2. **Use debug mode for troubleshooting**: `--debug --verbose`
3. **Check output structure** before deleting originals
4. **Validate with small collections** first

### Debugging Common Issues
```bash
# Check metadata extraction
python -m AudioSort "folder" --debug --scan --auto

# Verify file detection
find "folder" -type f \( -name "*.mp3" -o -name "*.m4b" \)

# Test API connectivity
python -c "import requests; print(requests.get('https://www.googleapis.com').status_code)"
```

## 🔍 Integration Points

### AudiobookShelf Compatibility
- Generates `metadata.opf` files using `template.opf`
- Creates `{Author}/{Title}` structure
- Series support with `series_index`

### SmartAudioBookPlayer Support
- Generates `info.txt` summaries
- Includes duration, narrator, and series information

### Extensibility
- **New metadata sources**: Add to `metadata_sources.py`
- **New output formats**: Extend functions in `actions.py`
- **Custom templates**: Modify `template.opf`

## 🚨 Important Development Notes

### Conflict Management System
- **Recent feature**: Intelligent series folder handling
- **Location**: `actions.py:prepare_output()`
- **Behavior**: Warns before overwriting existing files
- **Metadata updates**: `.opf` and `.txt` files are refreshed

### Python 3.9+ Compatibility
- Uses `dataclasses` with `field(default_factory=list)`
- No `slots=True` in dataclasses (compatibility fix in recent commits)
- Type hints with `dict[str, object]` format

### API Limitations & Fallbacks
- Google Books API: Free but may have rate limits
- Open Library: No API limits, reliable fallback
- DuckDuckGo: For URL discovery, no API key needed

## 📝 File Generation

### metadata.opf (AudiobookShelf)
```xml
<!-- Based on template.opf -->
<package version="2.0">
  <metadata>
    <title>{title}</title>
    <creator>{author}</creator>
    <series>{series}</series>
    <series_index>{position}</series_index>
  </metadata>
</package>
```

### info.txt (SmartAudioBookPlayer)
```
{title}
Author: {author}
Series: {series} #{position}
Duration: {duration}
Language: {language}
```

## 🎯 Best Practices for Development

1. **Always use preview mode** before actual processing
2. **Test with small collections** first
3. **Use debug mode** when troubleshooting API issues
4. **Check git history** for recent conflict management improvements
5. **Validate audio file formats** are supported
6. **Respect API rate limits** when adding new sources
7. **Maintain backward compatibility** with existing output structure

## 🔄 Common Workflows

### Adding New Metadata Source
1. Add function to `metadata_sources.py`
2. Update search order in main workflow
3. Add error handling and fallback logic
4. Test with `--debug` flag

### Modifying Output Structure
1. Update `actions.py:prepare_output()`
2. Modify template files if needed
3. Test conflict resolution with existing folders
4. Verify compatibility with target players

### Debugging Metadata Issues
```bash
# Step 1: Check search terms
./audiosort.sh --debug "problem_folder"

# Step 2: Test individual sources
python -c "from AudioSort.metadata_sources import *; print(fetch_google_books_data('search term'))"

# Step 3: Preview results
./audiosort.sh --preview "problem_folder"
```