# 📚 AudioSort 3.0 - Usage Examples

This file contains practical examples of how to use AudioSort 3.0 with both the Python module and the shell script.

## 🚀 Quick Start with Shell Script (Recommended)

```bash
# Make the script executable
chmod +x audiosort.sh

# Basic usage - automatic mode
./audiosort.sh "My Audiobook Collection"

# Preview mode - see what will happen without changes
./audiosort.sh --preview "Harry Potter Collection"

# Safe mode - copy instead of moving files
./audiosort.sh --safe "Audiobooks"
```

## 📖 Common Scenarios

### Scenario 1: New User with Harry Potter Collection

```bash
# Step 1: Preview what will be done
./audiosort.sh --preview "/Users/John/Downloads/Harry Potter"

# Step 2: Run in safe mode (copy, not move)
./audiosort.sh --safe "/Users/John/Downloads/Harry Potter"

# Step 3: Verify the result in _AudioSort_output_
# If satisfied, run the real command
./audiosort.sh "/Users/John/Downloads/Harry Potter"
```

### Scenario 2: Large Mixed Collection

```bash
# For a large collection with multiple authors
./audiosort.sh --verbose --opf --cover "/Users/John/Audiobooks"

# This will:
# - Scan all subdirectories automatically
# - Search metadata from multiple sources
# - Generate OPF files for AudiobookShelf
# - Download cover art
# - Show detailed progress
```

### Scenario 3: Organizing French Audiobooks

```bash
# French audiobooks often need the --auto flag for better metadata detection
./audiosort.sh --auto --opf --infotxt "/Users/John/Livres Audio"

# The script will automatically:
# - Detect French audiobooks
# - Search Google Books API with French titles
# - Use smart fallback for Harry Potter French editions
# - Generate metadata in French when available
```

### Scenario 4: Simple Organization Without Metadata

```bash
# When you just want basic organization without internet search
./audiosort.sh --basic "/Users/John/Unsorted Audiobooks"

# This will:
# - Scan and organize by folder names
# - Rename files to indexed format
# - Not search the internet for metadata
# - Create clean {Author}/{Title} structure
```

## 🔧 Advanced Usage

### Custom Output Directory

```bash
# Organize to a specific folder
./audiosort.sh --output "/Users/John/Bibliothèque" "/Users/John/Downloads/Audiobooks"
```

### Debug Mode

```bash
# When something goes wrong, use debug mode
./audiosort.sh --debug --verbose "Problematic Folder"

# This will show:
# - Detailed search terms
# - API responses
# - Error messages
# - Processing steps
```

### Using Python Module Directly

```bash
# Equivalent of ./audiosort.sh "folder"
python -m AudioSort "folder" --scan --auto --flatten --rename --opf --infotxt --cover

# Equivalent of ./audiosort.sh --preview "folder"
python -m AudioSort "folder" --scan --auto --flatten --rename --opf --infotxt --cover --dry-run

# Equivalent of ./audiosort.sh --safe "folder"
python -m AudioSort "folder" --scan --auto --flatten --rename --opf --infotxt --cover --copy
```

## 📁 Expected Results

### Before AudioSort
```
MyAudiobooks/
├── T1 - Harry Potter - Sorcerer's Stone [mp3 64kbps]/
│   ├── Disc 1/
│   │   ├── track01.mp3
│   │   └── track02.mp3
│   └── Disc 2/
│       ├── track03.mp3
│       └── track04.mp3
├── Stephen King - It [mp3]/
│   ├── part1.mp3
│   └── part2.mp3
└── random_audiobook/
    └── audio.m4b
```

### After AudioSort
```
_AudioSort_output_/
├── J.K._Rowling/
│   └── Harry_Potter_and_the_Sorcerers_Stone/
│       ├── 01 - Harry Potter Sorcerers Stone.mp3
│       ├── 02 - Harry Potter Sorcerers Stone.mp3
│       ├── 03 - Harry Potter Sorcerers Stone.mp3
│       ├── 04 - Harry Potter Sorcerers Stone.mp3
│       ├── metadata.opf
│       ├── info.txt
│       └── cover.jpg
├── Stephen_King/
│   └── It/
│       ├── 01 - It.mp3
│       ├── 02 - It.mp3
│       ├── metadata.opf
│       ├── info.txt
│       └── cover.jpg
└── Unknown_Author/
    └── random_audiobook/
        ├── 01 - random_audiobook.m4b
        └── metadata.opf
```

## 🎯 Troubleshooting Examples

### Problem: No Audio Files Detected

```bash
# Check what files are in your folder
find "YourFolder" -type f | head -10

# Supported formats: mp3, m4a, m4b, wma, flac, ogg
# If you have different formats, convert them first
```

### Problem: Wrong Metadata Found

```bash
# Use preview mode to check what metadata will be used
./audiosort.sh --preview "YourFolder"

# Check the debug output to see search terms
./audiosort.sh --debug "YourFolder"

# If wrong, try basic mode for folder name organization
./audiosort.sh --basic "YourFolder"
```

### Problem: Script Not Working

```bash
# Check if virtual environment exists
ls -la venv/

# Recreate if needed
rm -rf venv
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Try again
./audiosort.sh --debug "TestFolder"
```

## 🎨 Integration with Other Tools

### AudiobookShelf Integration

```bash
# Organize directly to AudiobookShelf library folder
./audiosort.sh --output "/path/to/audiobookshelf/library" "New Audiobooks"

# AudiobookShelf will automatically detect the new books
# Thanks to the metadata.opf files
```

### Plex Integration

```bash
# Organize for Plex media server
./audiosort.sh --output "/path/to/plex/Audiobooks" "Collection"

# Plex will recognize the {Author}/{Title} structure
```

### Calibre Integration

```bash
# Organize first, then import to Calibre
./audiosort.sh --opf --cover "Audiobooks"
# Import the _AudioSort_output_ folder to Calibre as eBooks
```

## 💡 Pro Tips

1. **Always use preview mode first** with large collections
2. **Use safe mode** when organizing valuable collections
3. **Enable debug** when troubleshooting issues
4. **Check the output** before deleting original folders
5. **Use verbose mode** to monitor progress on large collections

## 🔗 What to Do Next

1. **Test with a small collection** first
2. **Verify the output structure** meets your needs
3. **Import to your audiobook player** (AudiobookShelf, etc.)
4. **Delete original folders** only after verification
5. **Consider contributing** to the project if you find issues

---

## 📞 Need Help?

- 📖 **README**: Check the main README.md for detailed documentation
- 🐛 **Issues**: Report bugs on GitHub
- 💡 **Discussions**: Ask questions on GitHub Discussions
- 📧 **Email**: Contact the development team