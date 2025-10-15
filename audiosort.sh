#!/bin/bash

# AudioSort 3.0 - Script d'automatisation pour organiser vos audiobooks
# Auteur: AudioSort Team
# License: MIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# AudioSort configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="$SCRIPT_DIR/venv"
AUDIOSORT_MAIN="$SCRIPT_DIR/AudioSort/__main__.py"

# Default options
DEFAULT_OUTPUT="_AudioSort_output_"
DEFAULT_MODE="scan_auto"

# ASCII Art Header
echo -e "${CYAN}"
cat << "EOF"
╔══════════════════════════════════════════════════════════════╗
║                    🎧 AudioSort 3.0 🎧                      ║
║              Organisez vos audiobooks intelligemment          ║
╚══════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

# Help function
show_help() {
    echo -e "${BLUE}AudioSort 3.0 - Script d'automatisation${NC}"
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 [OPTIONS] <dossier_source>"
    echo ""
    echo -e "${YELLOW}Modes rapides:${NC}"
    echo "  $0 <dossier>           Mode automatique complet (recommandé)"
    echo "  $0 --preview <dossier> Mode preview (test sans modifier)"
    echo "  $0 --safe <dossier>    Mode copie (ne déplace pas les fichiers)"
    echo "  $0 --basic <dossier>   Mode basique (pas de métadonnées)"
    echo ""
    echo -e "${YELLOW}Options:${NC}"
    echo "  -o, --output DOSSIER   Dossier de sortie (défaut: $DEFAULT_OUTPUT)"
    echo "  -v, --verbose          Mode détaillé"
    echo "  -d, --debug            Mode debug"
    echo "  -c, --copy             Copier au lieu de déplacer"
    echo "  --flatten              Aplatir les dossiers multi-disques"
    echo "  --rename               Renommer les fichiers ordonnés"
    echo "  --opf                  Générer metadata.opf (AudiobookShelf)"
    echo "  --infotxt              Générer info.txt (SmartAudioBookPlayer)"
    echo "  --cover                Télécharger les pochettes"
    echo "  --no-scan              Ne pas scanner les sous-dossiers"
    echo "  --no-auto              Ne pas chercher les métadonnées automatiquement"
    echo "  --dry-run              Mode preview (test)"
    echo ""
    echo -e "${YELLOW}Exemples:${NC}"
    echo "  $0 '/Users/Moi/Mes Audiobooks'"
    echo "  $0 --preview '/Users/Moi/Harry Potter'"
    echo "  $0 --output '/Users/Moi/Bibliothèque' '/Users/Moi/Audiobooks'"
    echo "  $0 --verbose --opf --cover '/Users/Moi/Collection'"
    echo ""
    echo -e "${YELLOW}Modes disponibles:${NC}"
    echo "  • auto      : Scan + recherche métadonnées automatique"
    echo "  • preview   : Mode test sans modifications"
    echo "  • safe      : Mode copie sécurisée"
    echo "  • basic     : Organisation simple sans métadonnées"
    echo ""
}

# Check if virtual environment exists
check_venv() {
    if [ ! -d "$VENV_DIR" ]; then
        echo -e "${YELLOW}⚠️  Environnement virtuel non trouvé. Création en cours...${NC}"
        python3 -m venv "$VENV_DIR"
        if [ $? -ne 0 ]; then
            echo -e "${RED}❌ Erreur lors de la création de l'environnement virtuel${NC}"
            exit 1
        fi
        echo -e "${GREEN}✅ Environnement virtuel créé${NC}"
    fi
}

# Install dependencies if needed
check_dependencies() {
    source "$VENV_DIR/bin/activate"

    # Check if required packages are installed
    python -c "import requests, beautifulsoup4, tinytag" 2>/dev/null
    if [ $? -ne 0 ]; then
        echo -e "${YELLOW}📦 Installation des dépendances...${NC}"
        pip install -r "$SCRIPT_DIR/requirements.txt" --quiet
        if [ $? -ne 0 ]; then
            echo -e "${RED}❌ Erreur lors de l'installation des dépendances${NC}"
            exit 1
        fi
        echo -e "${GREEN}✅ Dépendances installées${NC}"
    fi
}

# Validate input directory
validate_input() {
    local input_dir="$1"

    if [ -z "$input_dir" ]; then
        echo -e "${RED}❌ Erreur: Veuillez spécifier un dossier source${NC}"
        echo -e "${BLUE}Usage: $0 <dossier_source>${NC}"
        exit 1
    fi

    if [ ! -d "$input_dir" ]; then
        echo -e "${RED}❌ Erreur: Le dossier '$input_dir' n'existe pas${NC}"
        exit 1
    fi

    # Check if directory contains audio files
    local has_audio=$(find "$input_dir" -type f \( -name "*.mp3" -o -name "*.m4a" -o -name "*.m4b" -o -name "*.wma" -o -name "*.flac" -o -name "*.ogg" \) | head -1)
    if [ -z "$has_audio" ]; then
        echo -e "${YELLOW}⚠️  Attention: Aucun fichier audio trouvé dans '$input_dir'${NC}"
        echo -e "${YELLOW}   Formats supportés: mp3, m4a, m4b, wma, flac, ogg${NC}"
        read -p "Continuer quand même? (o/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Oo]$ ]]; then
            exit 1
        fi
    fi
}

# Build AudioSort command
build_command() {
    local cmd="python"
    local args=""

    # Add main module
    args="$args -m AudioSort"

    # Add input directory
    args="$args \"$INPUT_DIR\""

    # Add output directory
    args="$args -O \"$OUTPUT_DIR\""

    # Add scan and auto flags (default behavior)
    if [ "$SCAN" = "true" ]; then
        args="$args --scan"
    fi

    if [ "$AUTO" = "true" ]; then
        args="$args --auto"
    fi

    # Add optional flags
    if [ "$FLATTEN" = "true" ]; then
        args="$args --flatten"
    fi

    if [ "$RENAME" = "true" ]; then
        args="$args --rename"
    fi

    if [ "$OPF" = "true" ]; then
        args="$args --opf"
    fi

    if [ "$INFOTXT" = "true" ]; then
        args="$args --infotxt"
    fi

    if [ "$COVER" = "true" ]; then
        args="$args --cover"
    fi

    if [ "$COPY" = "true" ]; then
        args="$args --copy"
    fi

    if [ "$DRY_RUN" = "true" ]; then
        args="$args --dry-run"
    fi

    if [ "$DEBUG" = "true" ]; then
        args="$args --debug"
    fi

    echo "$cmd $args"
}

# Execute AudioSort command
execute_audiosort() {
    local cmd=$(build_command)

    echo -e "${BLUE}🚀 Lancement d'AudioSort...${NC}"
    echo -e "${CYAN}Commande: $cmd${NC}"
    echo ""

    # Activate virtual environment and run
    source "$VENV_DIR/bin/activate"
    eval "$cmd"
    local exit_code=$?

    echo ""
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}🎉 AudioSort terminé avec succès!${NC}"
        echo -e "${GREEN}📁 Résultat dans: $OUTPUT_DIR${NC}"

        # Show results if directory exists and is not empty
        if [ -d "$OUTPUT_DIR" ] && [ "$(ls -A "$OUTPUT_DIR" 2>/dev/null)" ]; then
            echo ""
            echo -e "${BLUE}📊 Structure créée:${NC}"
            tree "$OUTPUT_DIR" 2>/dev/null || find "$OUTPUT_DIR" -type d | head -10
        fi
    else
        echo -e "${RED}❌ AudioSort a rencontré une erreur (code: $exit_code)${NC}"
        echo -e "${YELLOW}💡 Utilisez --debug pour plus de détails${NC}"
    fi

    return $exit_code
}

# Quick mode handlers
handle_preview_mode() {
    echo -e "${PURPLE}🔍 Mode Preview activé${NC}"
    echo -e "${YELLOW}Aucun fichier ne sera modifié${NC}"
    echo ""

    DRY_RUN="true"
    SCAN="true"
    AUTO="true"
    FLATTEN="true"
    RENAME="true"
    OPF="true"
    INFOTXT="true"
    COVER="true"
}

handle_safe_mode() {
    echo -e "${PURPLE}🛡️  Mode Sécurisé activé${NC}"
    echo -e "${YELLOW}Les fichiers seront copiés (pas de déplacement)${NC}"
    echo ""

    COPY="true"
    SCAN="true"
    AUTO="true"
    FLATTEN="true"
    RENAME="true"
    OPF="true"
    INFOTXT="true"
    COVER="true"
}

handle_basic_mode() {
    echo -e "${PURPLE}📦 Mode Basique activé${NC}"
    echo -e "${YELLOW}Organisation simple sans métadonnées${NC}"
    echo ""

    SCAN="true"
    AUTO="false"
    FLATTEN="true"
    RENAME="true"
}

handle_auto_mode() {
    echo -e "${PURPLE}🤖 Mode Automatique complet activé${NC}"
    echo -e "${YELLOW}Scan + recherche métadonnées + organisation complète${NC}"
    echo ""

    SCAN="true"
    AUTO="true"
    FLATTEN="true"
    RENAME="true"
    OPF="true"
    INFOTXT="true"
    COVER="true"
}

# Default values
OUTPUT_DIR="$DEFAULT_OUTPUT"
SCAN="true"
AUTO="true"
FLATTEN="false"
RENAME="false"
OPF="false"
INFOTXT="false"
COVER="false"
COPY="false"
DRY_RUN="false"
DEBUG="false"
VERBOSE="false"

# Parse command line arguments
INPUT_DIR=""
MODE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            show_help
            exit 0
            ;;
        --preview)
            MODE="preview"
            shift
            ;;
        --safe)
            MODE="safe"
            shift
            ;;
        --basic)
            MODE="basic"
            shift
            ;;
        --auto)
            MODE="auto"
            shift
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="true"
            DEBUG="true"
            shift
            ;;
        -d|--debug)
            DEBUG="true"
            shift
            ;;
        -c|--copy)
            COPY="true"
            shift
            ;;
        --flatten)
            FLATTEN="true"
            shift
            ;;
        --rename)
            RENAME="true"
            shift
            ;;
        --opf)
            OPF="true"
            shift
            ;;
        --infotxt)
            INFOTXT="true"
            shift
            ;;
        --cover)
            COVER="true"
            shift
            ;;
        --no-scan)
            SCAN="false"
            shift
            ;;
        --no-auto)
            AUTO="false"
            shift
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        -*)
            echo -e "${RED}❌ Option inconnue: $1${NC}"
            echo -e "${BLUE}Utilisez --help pour voir les options disponibles${NC}"
            exit 1
            ;;
        *)
            if [ -z "$INPUT_DIR" ]; then
                INPUT_DIR="$1"
            else
                echo -e "${RED}❌ Plusieurs dossiers spécifiés: '$INPUT_DIR' et '$1'${NC}"
                exit 1
            fi
            shift
            ;;
    esac
done

# Apply mode if specified
case $MODE in
    preview)
        handle_preview_mode
        ;;
    safe)
        handle_safe_mode
        ;;
    basic)
        handle_basic_mode
        ;;
    auto)
        handle_auto_mode
        ;;
esac

# If no mode specified and no flags, use auto mode
if [ -z "$MODE" ] && [ "$FLATTEN" = "false" ] && [ "$RENAME" = "false" ] && [ "$OPF" = "false" ]; then
    handle_auto_mode
fi

# Validate input
validate_input "$INPUT_DIR"

# Convert to absolute path
INPUT_DIR="$(realpath "$INPUT_DIR")"
OUTPUT_DIR="$(realpath "$OUTPUT_DIR")"

# Show configuration
if [ "$VERBOSE" = "true" ]; then
    echo -e "${BLUE}⚙️  Configuration:${NC}"
    echo -e "  📁 Dossier source: $INPUT_DIR"
    echo -e "  📁 Dossier sortie: $OUTPUT_DIR"
    echo -e "  🔍 Scan: $SCAN"
    echo -e "  🤖 Auto: $AUTO"
    echo -e "  📦 Flatten: $FLATTEN"
    echo -e "  ✏️  Rename: $RENAME"
    echo -e "  📄 OPF: $OPF"
    echo -e "  📝 Info TXT: $INFOTXT"
    echo -e "  🖼️  Cover: $COVER"
    echo -e "  📋 Copy: $COPY"
    echo -e "  👀 Dry Run: $DRY_RUN"
    echo ""
fi

# Setup environment
check_venv
check_dependencies

# Execute AudioSort
execute_audiosort

# Final message
echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                    Merci d'avoir utilisé AudioSort !             ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo -e "${GREEN}🌟 Site web: https://github.com/votre-username/AudioSort${NC}"
echo -e "${GREEN}⭐ Star ce projet sur GitHub !${NC}"