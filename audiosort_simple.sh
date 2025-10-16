#!/bin/bash

# AudioSort 3.0 - Simplified Interface
# A simple and intuitive way to organize your audiobooks

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="$SCRIPT_DIR/venv"
DEFAULT_OUTPUT="$HOME/Library"

# Header simple
echo -e "${CYAN}"
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    🎧 AudioSort 3.0 🎧                      ║"
echo "║              The smart audiobook organizer                  ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Simplified utility functions
check_venv() {
    if [ ! -d "$VENV_DIR" ]; then
        echo -e "${YELLOW}🔧 Initial configuration...${NC}"
        python3 -m venv "$VENV_DIR" 2>/dev/null
        if [ $? -ne 0 ]; then
            echo -e "${RED}❌ Unable to create Python environment${NC}"
            echo "💡 Install Python 3.7+: https://www.python.org/downloads/"
            exit 1
        fi
    fi

    source "$VENV_DIR/bin/activate"

    # Silent dependency installation
    python -c "import requests, beautifulsoup4, tinytag" 2>/dev/null || {
        echo -e "${YELLOW}📦 Installing dependencies...${NC}"
        pip install -r "$SCRIPT_DIR/requirements.txt" --quiet 2>/dev/null
    }
}

analyze_folder() {
    local folder="$1"

    if [ ! -d "$folder" ]; then
        echo -e "${RED}❌ The folder does not exist: $folder${NC}"
        return 1
    fi

    # Count audio files
    local audio_count=$(find "$folder" -type f \( -name "*.mp3" -o -name "*.m4a" -o -name "*.m4b" -o -name "*.wma" -o -name "*.flac" -o -name "*.ogg" \) | wc -l)
    local folder_count=$(find "$folder" -type d -name "*" | wc -l)

    if [ "$audio_count" -eq 0 ]; then
        echo -e "${RED}❌ No audio files found${NC}"
        echo "   Supported formats: MP3, M4A, M4B, WMA, FLAC, OGG"
        return 1
    fi

    # Detect if it's a collection or a single book
    if [ "$folder_count" -gt 2 ]; then
        echo -e "${BLUE}📚 Collection detected:${NC}"
        echo "   📁 $folder_count folders"
        echo "   🎧 $audio_count audio files"
        echo "   🔍 Automatic scan enabled"
        return 2  # Collection mode
    else
        echo -e "${GREEN}📖 Single book detected:${NC}"
        echo "   🎧 $audio_count audio files"
        return 1  # Single book mode
    fi
}

smart_preview() {
    local input="$1"
    local output="$2"

    echo ""
    echo -e "${PURPLE}📋 Organization summary:${NC}"
    echo "   📂 Source folder: $(basename "$input")"
    echo "   📁 Destination: $output"
    echo ""

    # Quickly analyze content
    if [ -d "$input" ]; then
        echo -e "${BLUE}🔍 What I will do:${NC}"
        echo "   ✅ Scan all subfolders"
        echo "   ✅ Search metadata automatically"
        echo "   ✅ Organize by author/series"
        echo "   ✅ Download covers"
        echo "   ✅ Generate compatibility files"
        echo ""

        # Detect some examples
        local examples=$(find "$input" -maxdepth 2 -type d \( -name "*Harry*" -o -name "*Potter*" -o -name "*King*" -o -name "*Rowling*" \) | head -3)
        if [ -n "$examples" ]; then
            echo -e "${YELLOW}📚 Some titles detected:${NC}"
            echo "$examples" | sed 's|.*/||' | head -3 | while read line; do
                echo "   📖 $line"
            done
        fi
    fi

    echo ""
    echo -e "${CYAN}⚡ Guaranteed compatibility:${NC}"
    echo "   📱 AudiobookShelf ✅"
    echo "   📱 SmartAudioBookPlayer ✅"
    echo ""
}

execute_intelligent() {
    local input="$1"
    local output="$2"

    # Automatically build optimized command
    local cmd="python -m AudioSort \"$input\""
    cmd="$cmd -O \"$output\""
    cmd="$cmd --scan --auto"
    cmd="$cmd --flatten --rename"
    cmd="$cmd --opf --infotxt --cover"
    cmd="$cmd --copy"  # Safer by default

    echo -e "${BLUE}🚀 Starting organization...${NC}"
    echo -e "${CYAN}Command: $cmd${NC}"
    echo ""

    # Execute
    source "$VENV_DIR/bin/activate"
    eval "$cmd"

    local result=$?

    if [ $result -eq 0 ]; then
        echo ""
        echo -e "${GREEN}🎉 Organization completed successfully!${NC}"
        echo -e "${GREEN}📁 Your library is ready: $output${NC}"

        # Show a preview of the result
        if [ -d "$output" ] && [ "$(ls -A "$output" 2>/dev/null)" ]; then
            echo ""
            echo -e "${BLUE}📊 Preview of your library:${NC}"
            find "$output" -maxdepth 2 -type d | head -5 | while read line; do
                echo "   📂 $(basename "$line")"
            done
        fi
    else
        echo ""
        echo -e "${RED}❌ An error occurred${NC}"
        echo -e "${YELLOW}💡 Try with: ./audiosort.sh --debug \"$input\"${NC}"
    fi

    return $result
}

# Main interactive menu
show_menu() {
    echo ""
    echo -e "${YELLOW}👋 Welcome! What would you like to do?${NC}"
    echo ""
    echo "1️⃣  Organize my audiobook collection (recommended)"
    echo "2️⃣  See what will be changed (preview mode)"
    echo "3️⃣  Add only a few books"
    echo "4️⃣  Use a specific folder"
    echo "5️⃣  Help / Configuration"
    echo ""
    read -p "Choose an option [1-5] : " choice

    case $choice in
        1)
            echo ""
            echo -e "${PURPLE}📚 Complete Organization Mode${NC}"
            echo ""
            read -p "Enter your audiobook folder: " input_folder

            if [ -z "$input_folder" ]; then
                # Look for common folders
                echo -e "${YELLOW}💡 Common folders detected:${NC}"
                find "$HOME" -maxdepth 2 -type d \( -name "*audiobook*" -o -name "*Audio*" -o -name "*Book*" \) 2>/dev/null | head -3 | while read line; do
                    echo "   📁 $line"
                done
                echo ""
                read -p "Folder to organize: " input_folder
            fi

            if [ -n "$input_folder" ]; then
                analyze_folder "$input_folder"
                local mode=$?

                smart_preview "$input_folder" "$DEFAULT_OUTPUT"

                echo -e "${CYAN}Confirm organization? (y/N) : ${NC}"
                read -r confirm

                if [[ $confirm =~ ^[Yy]$ ]]; then
                    execute_intelligent "$input_folder" "$DEFAULT_OUTPUT"
                else
                    echo -e "${YELLOW}❌ Operation cancelled${NC}"
                fi
            fi
            ;;
        2)
            echo ""
            echo -e "${PURPLE}🔍 Preview Mode${NC}"
            echo "I will show you what will be organized without modifying your files."
            echo ""
            read -p "Folder to analyze: " input_folder

            if [ -n "$input_folder" ]; then
                analyze_folder "$input_folder"
                smart_preview "$input_folder" "$DEFAULT_OUTPUT"

                echo ""
                echo -e "${GREEN}✅ That's it! No files have been modified.${NC}"
                echo -e "${YELLOW}💡 To start organization, choose option 1${NC}"
            fi
            ;;
        3)
            echo ""
            echo -e "${PURPLE}📖 Quick Add Mode${NC}"
            echo "Perfect for adding a few new books to your collection."
            echo ""
            read -p "Folder of the new book(s): " input_folder

            if [ -n "$input_folder" ]; then
                # Simpler mode for a few books
                analyze_folder "$input_folder"

                echo ""
                echo -e "${BLUE}🎯 Simple mode:${NC}"
                echo "   ✅ Search for metadata"
                echo "   ✅ Organize in your library"
                echo "   ✅ Safe copy (nothing is deleted)"
                echo ""

                echo -e "${CYAN}Add to your library? (y/N) : ${NC}"
                read -r confirm

                if [[ $confirm =~ ^[Yy]$ ]]; then
                    execute_intelligent "$input_folder" "$DEFAULT_OUTPUT"
                fi
            fi
            ;;
        4)
            echo ""
            echo -e "${PURPLE}🎯 Custom Mode${NC}"
            echo ""
            read -p "Source folder: " input_folder
            read -p "Destination folder [$DEFAULT_OUTPUT]: " output_folder

            if [ -z "$output_folder" ]; then
                output_folder="$DEFAULT_OUTPUT"
            fi

            if [ -n "$input_folder" ]; then
                analyze_folder "$input_folder"
                smart_preview "$input_folder" "$output_folder"

                echo -e "${CYAN}Start organization? (y/N) : ${NC}"
                read -r confirm

                if [[ $confirm =~ ^[Yy]$ ]]; then
                    execute_intelligent "$input_folder" "$output_folder"
                fi
            fi
            ;;
        5)
            echo ""
            echo -e "${PURPLE}❓ Help & Configuration${NC}"
            echo ""
            echo "🎯 AudioSort automatically organizes your audiobooks:"
            echo ""
            echo "📁 Structure created:"
            echo "   Library/"
            echo "   ├── J.K._Rowling/"
            echo "   │   ├── Harry_Potter_Series/"
            echo "   │   │   ├── 01 - Harry Potter.mp3"
            echo "   │   │   ├── metadata.opf (AudiobookShelf)"
            echo "   │   │   ├── info.txt (SmartAudioBookPlayer)"
            echo "   │   │   └── cover.jpg"
            echo "   │   └── Other_Series/"
            echo "   └── Stephen_King/"
            echo ""
            echo "🔧 Features:"
            echo "   ✅ Automatic metadata search (Google Books, Open Library)"
            echo "   ✅ Cover downloads"
            echo "   ✅ Organization by author and series"
            echo "   ✅ AudiobookShelf and SmartAudioBookPlayer compatibility"
            echo "   ✅ Safe copy mode (your files remain intact)"
            echo ""
            echo "📱 Compatible with:"
            echo "   • AudiobookShelf (desktop/mobile server)"
            echo "   • SmartAudioBookPlayer (Android)"
            echo "   • Plex, Jellyfin, etc."
            echo ""
            echo "💡 Tips:"
            echo "   • Use preview mode (option 2) to test"
            echo "   • Your original files are never deleted"
            echo "   • The program automatically detects series"
            echo ""

            echo -e "${YELLOW}Press Enter to return to menu...${NC}"
            read
            show_menu
            ;;
        *)
            echo -e "${RED}❌ Invalid option${NC}"
            show_menu
            ;;
    esac
}

# Check if an argument was passed on the command line
if [ $# -eq 0 ]; then
    # Interactive mode
    check_venv
    show_menu
else
    # Direct mode (compatibility with the old script)
    if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
        echo "AudioSort 3.0 - Simplified Interface"
        echo ""
        echo "Usage:"
        echo "  $0                           # Interactive mode (recommended)"
        echo "  $0 \"audiobook/folder\"       # Direct organization"
        echo "  $0 --help                    # This help"
        echo ""
        echo "Examples:"
        echo "  $0 \"/Users/Me/My Audiobooks\""
        echo "  $0 \"Harry Potter Collection\""
        echo ""
        exit 0
    else
        # Direct mode with folder passed as parameter
        input_folder="$1"

        echo -e "${BLUE}🎯 Direct Mode: $input_folder${NC}"

        check_venv
        analyze_folder "$input_folder"
        smart_preview "$input_folder" "$DEFAULT_OUTPUT"

        echo -e "${CYAN}Start organization? (y/N) : ${NC}"
        read -r confirm

        if [[ $confirm =~ ^[Yy]$ ]]; then
            execute_intelligent "$input_folder" "$DEFAULT_OUTPUT"
        else
            echo -e "${YELLOW}❌ Operation cancelled${NC}"
            echo -e "${YELLOW}💡 Simply use: $0 for guided mode${NC}"
        fi
    fi
fi

echo ""
echo -e "${CYAN}Thank you for using AudioSort! 📚✨${NC}"