# 🎧 AudioSort 3.0

<div align="center">

![AudioSort Logo](https://img.shields.io/badge/AudioSort-3.0-blue?style=for-the-badge)
![Python](https://img.shields.io/badge/Python-3.7%2B-green?style=for-the-badge&logo=python)
![License](https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge)

*Organize your audiobook collection with intelligence and automation*

[Features](#-features) • [Installation](#-installation) • [Usage](#-usage) • [Examples](#-examples) • [API](#-api-sources)

</div>

## 🌟 Description

**AudioSort** is a powerful Python tool that automatically organizes your audiobook collection by fetching metadata from multiple sources and creating the perfect structure for your favorite players like **AudiobookShelf** and **SmartAudioBookPlayer**.

> 🎯 **Perfect for:** Audiobook enthusiasts who want to quickly organize large collections with complete metadata and consistent structure.

---

## ✨ Features

### 🚀 **Automatic Scanning**
- Automatically detects all audiobook folders in subdirectories
- Analyzes audio files to identify content
- Supports formats: `.mp3`, `.m4a`, `.m4b`, `.wma`, `.flac`, `.ogg`

### 🌍 **Multi-Source Search**
- **📚 Google Books API** - World's largest book database
- **📖 Open Library** - Collaborative database of 20M+ books
- **🎧 Audible** - Professional audiobook metadata
- **📚 Goodreads** - Reader community
- **🤖 Smart Fallback** - Extraction from folder names

### 🏗️ **Intelligent Organization**
```
{output_folder}/
├── {Author}/
│   └── {Series}_Series/       ← Smart series grouping
│       ├── 01 - {Title}.mp3
│       ├── 02 - {Title}.mp3
│       ├── metadata.opf      ← AudiobookShelf
│       ├── info.txt          ← SmartAudioBookPlayer
│       └── cover.jpg
```

**✨ NEW: Smart Series Grouping**
- Automatically groups books from the same series together
- Merges new books into existing series folders
- Warns about file conflicts before overwriting
- Supports adding books to existing collections

### 🎛️ **Complete Options**
- `--flatten`: Flatten multi-disc folders
- `--rename`: Rename files to indexed format
- `--opf`: Generate metadata.opf (AudiobookShelf)
- `--infotxt`: Generate text summary (SmartAudioBookPlayer)
- `--cover`: Download cover art automatically
- `--copy`: Copy instead of moving files
- `--dry-run`: Preview mode to test before executing

---

## 🚀 Installation

### Prerequisites
- Python 3.7 or higher
- pip (Python package manager)

### Quick Installation
```bash
# Clone the repository
git clone https://github.com/lxwiq/AudioSort.git
cd AudioSort

# Create virtual environment (recommended)
python3 -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

### Dependencies
- `requests` - HTTP requests
- `beautifulsoup4` - HTML parsing
- `tinytag` - ID3 metadata reading

---

## 📖 Usage

### Basic Command
```bash
python -m AudioSort "audiobook_folder"
```

### Complete Automatic Scan 🌟
```bash
python -m AudioSort "my_collection/" --scan --auto --flatten --rename --opf --infotxt --cover
```

### Using the Shell Script (Recommended)
```bash
# Make script executable
chmod +x audiosort.sh

# Automatic mode (recommended)
./audiosort.sh "My Audiobook Collection"

# Preview mode
./audiosort.sh --preview "Harry Potter Collection"

# Safe mode (copy instead of move)
./audiosort.sh --safe "Audiobooks"
```

### Preview Mode (Test)
```bash
python -m AudioSort "my_collection/" --scan --auto --dry-run
```

### Available Options

| Option | Description | Example |
|--------|-------------|---------|
| `--scan` | Automatic scan of subdirectories | `--scan` |
| `--auto` | Automatic metadata search | `--auto` |
| `--flatten` | Flatten multi-disc folders | `--flatten` |
| `--rename` | Rename to indexed format | `--rename` |
| `--opf` | Generate metadata.opf | `--opf` |
| `--infotxt` | Generate info.txt | `--infotxt` |
| `--cover` | Download cover art | `--cover` |
| `--copy` | Copy instead of move | `--copy` |
| `--dry-run` | Preview mode without changes | `--dry-run` |
| `-O` | Custom output folder | `-O "MyBooks"` |
| `-d` | Enable debug logging | `--debug` |

### Shell Script Quick Modes

| Mode | Description | Command |
|------|-------------|---------|
| **Auto** | Complete automatic mode | `./audiosort.sh "folder"` |
| **Preview** | Test without modifications | `./audiosort.sh --preview "folder"` |
| **Safe** | Copy mode (no moving) | `./audiosort.sh --safe "folder"` |
| **Basic** | Simple organization | `./audiosort.sh --basic "folder"` |

### 🔧 **Conflict Management**

AudioSort intelligently handles existing folders and files:

#### **Existing Series Folders**
```bash
# If J.K._Rowling/Harry_Potter_Series/ already exists
./audiosort.sh "Harry Potter Book 3"

# Output:
📁 Adding to existing folder: /path/to/J.K._Rowling/Harry_Potter_Series
⚠️  2 files will be overwritten:
   - metadata.opf
   - cover.jpg
```

#### **File Overwriting**
- 📝 Metadata files (`.opf`, `.txt`, `.json`) are updated with new information
- 🖼️ Cover images are replaced with higher quality versions
- ⚠️ Clear warnings before any overwriting occurs
- ✅ Use `--dry-run` to preview changes first

---

## 🎚️ Examples

### Example 1: Harry Potter Collection
```bash
# Before
MyCollection/
├── T1 - Harry Potter - Sorcerer's Stone [mp3 64kbps]/
├── T2 - Harry Potter - Chamber of Secrets [mp3 64kbps]/
└── ...

# Command
./audiosort.sh "MyCollection/"

# After
_AudioSort_output_/
└── J.K._Rowling/
    ├── Harry_Potter_and_the_Sorcerers_Stone/
    │   ├── 01 - Harry Potter Sorcerers Stone.mp3
    │   ├── 02 - Harry Potter Sorcerers Stone.mp3
    │   ├── metadata.opf
    │   ├── info.txt
    │   └── cover.jpg
    └── Harry_Potter_and_the_Chamber_of_Secrets/
        └── ...
```

### Example 2: Multiple Authors
```bash
# Scan with different authors
./audiosort.sh "Audiobooks/" --opf --cover

# Automatic result
_AudioSort_output_/
├── J.K._Rowling/
│   └── Harry_Potter/
├── Stephen_King/
│   └── It/
└── Agatha_Christie/
    └── Murder_on_the_Orient_Express/
```

### Example 3: Preview Mode
```bash
# See what will happen without modifying
./audiosort.sh --preview "Collection/"

# Expected output
=== SUCCESS ===
(dry-run) T1 - Harry Potter - Sorcerer's Stone
(dry-run) T2 - Harry Potter - Chamber of Secrets
```

---

## 🔍 API Sources

AudioSort uses multiple sources to find metadata:

### 1. **Google Books API** 📚
- ✅ World's largest book database
- ✅ Multi-language support (English, French, etc.)
- ✅ Complete metadata and HD covers
- ✅ 100% free

### 2. **Open Library** 📖
- ✅ Collaborative database (20M+ books)
- ✅ Public API with no limits
- ✅ Complete metadata
- ✅ Open source

### 3. **Audible** 🎧
- ✅ Professional audiobook metadata
- ✅ Narrator information
- ✅ Durations and series
- ✅ Official cover art

### 4. **Goodreads** 📚
- ✅ Reader community
- ✅ Ratings and reviews
- ✅ Complete metadata

### 5. **Smart Fallback** 🤖
- ✅ Extraction from folder names
- ✅ Automatic Harry Potter series detection
- ✅ Intelligent title cleaning

---

## 🎯 Search Workflow

1. **Analyze** folder/audio files
2. **Extract** optimized search term
3. **Search** DuckDuckGo (Audible/Goodreads)
4. **API** Google Books (if failed)
5. **API** Open Library (if failed)
6. **Fallback** intelligent (if all failed)
7. **Organize** automatically with metadata

---

## 📝 Generated Files

### metadata.opf (AudiobookShelf)
```xml
<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0">
  <metadata>
    <title>Harry Potter and the Sorcerer's Stone</title>
    <creator>J. K. Rowling</creator>
    <series>Harry Potter</series>
    <series_index>1</series_index>
  </metadata>
</package>
```

### info.txt (SmartAudioBookPlayer)
```
Harry Potter and the Sorcerer's Stone
Author: J. K. Rowling
Series: Harry Potter #1
Duration: 8h 32m
Language: English
```

---

## 🔧 Configuration

### Environment Variables
```bash
export AUDIOSORT_OUTPUT="/path/to/output"  # Default folder
export AUDIOSORT_DEBUG=true                # Debug logging
```

### Custom OPF Template
```bash
python -m AudioSort "book/" --template "my_template.opf"
```

---

## 🚨 Troubleshooting

### Common Issues

**Q: AudioSort can't find metadata**
```bash
# Try with more debug
python -m AudioSort "folder/" --debug --scan --auto

# Check the extracted search term
# It will be shown in logs with "Suggested search term"
```

**Q: I don't like the organization**
```bash
# Use --dry-run to test before
python -m AudioSort "folder/" --dry-run --scan --auto

# Use --copy to not lose originals
python -m AudioSort "folder/" --copy --scan --auto
```

**Q: Audio files are not detected**
```bash
# Check supported formats
# Formats: .mp3, .m4a, .m4b, .wma, .flac, .ogg
```

### Debug Logs
```bash
python -m AudioSort "folder/" --debug
# or
./audiosort.sh --debug "folder/"
```

---

## 🤝 Contributing

Contributions are welcome!

1. Fork the project
2. Create a branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Contribution Ideas
- [ ] Support for new APIs (WorldCat, etc.)
- [ ] Graphical interface (GUI)
- [ ] Calibre plugin
- [ ] Support for .cue files
- [ ] Integration with Plex/Jellyfin

---

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- **Google Books API** - For access to their massive database
- **Open Library** - For their open and collaborative API
- **Audible** - For quality audiobook metadata
- **Goodreads** - For the reader community

---

## 📞 Support

- 🐛 **Bugs**: [GitHub Issues](https://github.com/lxwiq/AudioSort/issues)
- 💡 **Suggestions**: [GitHub Discussions](https://github.com/lxwiq/AudioSort/discussions)
- 📧 **Email**: your-email@example.com

---

<div align="center">

**Made with ❤️ for audiobook lovers**

[⭐ Star this project] | [🍴 Fork] | [🐛 Report Bug]

</div>