from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path
from typing import Iterable

import requests
from tinytag import TinyTag

from . import __version__
from .actions import (
    download_cover,
    flatten,
    prepare_output,
    relocate_source,
    rename_tracks,
    write_info,
    write_json,
    write_opf,
)
from .config import AudioSortConfig
from .metadata_sources import MetadataError, fetch_metadata, fetch_metadata_by_search
from .models import BookMetadata, ProcessingPlan
from .search import auto_search


LOG = logging.getLogger("audiosort")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="python -m AudioSort",
        description="Organize audiobook folders using web metadata without clipboard dependencies.",
    )
    parser.add_argument("folders", metavar="folder", nargs="*", help="Audiobook folders to process")
    parser.add_argument("-O", "--output", help="Destination root folder", default="_AudioSort_output_")
    parser.add_argument("-c", "--copy", action="store_true", help="Copy instead of move")
    parser.add_argument("-f", "--flatten", action="store_true", help="Flatten multi-disc folders")
    parser.add_argument("-r", "--rename", action="store_true", help="Rename tracks to indexed format")
    parser.add_argument("-o", "--opf", action="store_true", help="Write metadata.opf file")
    parser.add_argument("-i", "--infotxt", action="store_true", help="Write info.txt summary")
    parser.add_argument("-j", "--json", action="store_true", help="Write metadata.json summary")
    parser.add_argument("-k", "--cover", action="store_true", help="Download cover art when available")
    parser.add_argument("-s", "--site", choices=["audible", "goodreads", "both"], default="both", help="Preferred metadata source")
    parser.add_argument("-a", "--auto", action="store_true", help="Automatically search DuckDuckGo for metadata URL")
    parser.add_argument("--scan", action="store_true", help="Automatically scan for audiobook folders in subdirectories")
    parser.add_argument("-d", "--dry-run", action="store_true", help="Display actions without modifying files")
    parser.add_argument("--template", default="template.opf", help="Path to OPF template")
    parser.add_argument("--debug", action="store_true", help="Enable debug logs")
    parser.add_argument("--config", help="Path to configuration file (default: audiosort_config.json)")
    parser.add_argument("--save-config", action="store_true", help="Save current settings as default")
    parser.add_argument("--reset-config", action="store_true", help="Reset configuration to defaults")
    parser.add_argument("--skip-existing", action="store_true", help="Skip folders that already exist in output")
    parser.add_argument("-V", "--version", action="version", version=f"AudioSort {__version__}")
    return parser


def main(argv: Iterable[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    logging.basicConfig(level=logging.DEBUG if args.debug else logging.INFO, format="%(levelname)s - %(message)s")

    # Initialize configuration
    config_path = Path(args.config) if args.config else None
    config = AudioSortConfig(config_path)

    # Handle configuration commands that don't require a folder
    if args.reset_config:
        if config_path and config_path.exists():
            config_path.unlink()
            print("🗑️  Configuration deleted")
        else:
            print("🗑️  No configuration file found")
        return 0

    # Verify that at least one folder is specified for other commands
    if not args.folders:
        LOG.error("At least one folder must be specified")
        parser.print_help()
        return 2

    # Use configuration default values if not specified
    if not args.output and config.get_default_output_folder():
        args.output = config.get_default_output_folder()
        print(f"📁 Default output folder: {args.output}")

    session = requests.Session()

    template_path = Path(args.template).resolve()
    destination_root = Path(args.output).resolve()

    # Handle scan mode
    if args.scan:
        if len(args.folders) != 1:
            LOG.error("Scan mode requires exactly one folder to scan")
            return 2

        # Use default input folder if available
        input_folder = args.folders[0]
        if not input_folder and config.get_default_input_folder():
            input_folder = config.get_default_input_folder()
            print(f"📂 Default input folder: {input_folder}")

        root_folder = Path(input_folder).resolve()
        if not root_folder.is_dir():
            LOG.error("Folder does not exist: %s", root_folder)
            return 2

        # Save input folder for next time
        config.set_default_input_folder(str(root_folder))

        folders = scan_for_audiobooks(root_folder)
    else:
        folders = [Path(folder).resolve() for folder in args.folders]
        for folder in folders:
            if not folder.is_dir():
                LOG.error("Folder does not exist: %s", folder)
                return 2

    # Save current settings if requested
    if args.save_config:
        settings = {
            "scan": args.scan,
            "auto": args.auto,
            "flatten": args.flatten,
            "rename": args.rename,
            "opf": args.opf,
            "infotxt": args.infotxt,
            "cover": args.cover,
            "copy": args.copy,
            "skip_existing_folders": args.skip_existing
        }
        config.save_last_settings(settings)
        config.set_default_output_folder(str(destination_root))
        print("💾 Settings saved to configuration")

    successes: list[str] = []
    failures: list[str] = []
    skipped: list[str] = []

    for folder in folders:
        LOG.info("Processing %s", folder)
        plan = ProcessingPlan(
            source_folder=folder,
            destination_root=destination_root,
            copy=args.copy,
            flatten=args.flatten,
            rename_tracks=args.rename,
            create_opf=args.opf,
            create_info=args.infotxt,
            download_cover=args.cover,
            emit_json=args.json,
            dry_run=args.dry_run,
        )

        metadata_url = obtain_metadata_url(folder, session, args.site, args.auto, args.scan)
        if not metadata_url:
            # Try to fetch metadata from APIs directly by search term
            if args.auto or args.scan:
                search_term = infer_search_term(folder)
                LOG.info("Searching metadata APIs for %s", folder.name)
                metadata = fetch_metadata_by_search(search_term, session)
                if metadata:
                    LOG.info("Found metadata via API search for %s", folder.name)
                else:
                    # Use fallback metadata if everything fails
                    if args.scan:
                        LOG.info("Using fallback metadata for %s", folder.name)
                        metadata = create_fallback_metadata(folder)
                    else:
                        LOG.warning("No metadata found, skipping %s", folder.name)
                        skipped.append(folder.name)
                        continue
            else:
                LOG.warning("No metadata selected, skipping %s", folder.name)
                skipped.append(folder.name)
                continue
        else:
            try:
                metadata = fetch_metadata(metadata_url, session)
            except MetadataError as exc:
                LOG.error("%s", exc)
                failures.append(f"{folder.name}: {exc}")
                continue

        enrich_with_id3_defaults(metadata, folder)
        destination = prepare_output(metadata, plan, config)
        LOG.info("Destination: %s", destination)

        # Check if existing folders should be skipped
        if args.skip_existing and destination.exists() and any(destination.iterdir()):
            LOG.info("Skipping existing folder: %s", destination)
            skipped.append(f"{folder.name} (already exists)")
            continue

        if plan.dry_run:
            successes.append(f"(dry-run) {folder.name}")
            continue

        relocate_source(metadata, plan, destination, config)
        if plan.flatten:
            flatten(destination, metadata, plan.dry_run)
        if plan.rename_tracks:
            rename_tracks(destination, metadata, plan.dry_run)
        if plan.create_opf:
            write_opf(destination, metadata, template_path, plan.dry_run)
        if plan.create_info:
            write_info(destination, metadata, plan.dry_run)
        if plan.emit_json:
            write_json(destination, metadata, plan.dry_run)
        if plan.download_cover:
            download_cover(destination, metadata, session, plan.dry_run)

        successes.append(folder.name)

    _summarize(successes, failures, skipped)
    return 0 if not failures else 1


def obtain_metadata_url(folder: Path, session: requests.Session, site: str, auto: bool, scan_mode: bool = False) -> str | None:
    search_term = infer_search_term(folder)
    if auto:
        candidates = auto_search(search_term, site, session)
        if candidates:
            if not scan_mode:
                print(f"Selected {candidates[0]} from automatic search")
            else:
                LOG.info("Auto-selected metadata for %s", folder.name)
            return candidates[0]
        else:
            if not scan_mode:
                print(f"No automatic results found for {folder.name}")
            else:
                LOG.info("No automatic results found for %s", folder.name)

    # In scan mode without auto (or auto failed), we can't ask for input, so skip
    if scan_mode:
        LOG.warning("Skipping %s - automatic search failed, use --auto with --scan for automatic processing", folder.name)
        return None

    print(f"Folder: {folder.name}")
    print(f"Suggested search term: {search_term}")
    print("Enter metadata URL (Audible or Goodreads), type 'search' to list suggestions, or 'skip' to ignore")
    while True:
        response = input("> ").strip()
        if not response:
            continue
        if response.lower() == "skip":
            return None
        if response.lower() == "search":
            suggestions = auto_search(search_term, site, session)
            if not suggestions:
                print("No results found, paste URL manually or type skip")
                continue
            for idx, suggestion in enumerate(suggestions, start=1):
                print(f"{idx}. {suggestion}")
            selection = input("Select number or paste URL: ").strip()
            if selection.isdigit():
                index = int(selection) - 1
                if 0 <= index < len(suggestions):
                    return suggestions[index]
                print("Invalid selection")
                continue
            if selection:
                return selection
            continue
        if response.startswith("http"):
            return response
        print("Provide a valid URL, 'search', or 'skip'")


def scan_for_audiobooks(root_path: Path) -> list[Path]:
    """
    Scan for audiobook folders recursively.
    An audiobook folder is defined as a folder containing audio files.
    """
    audio_extensions = {".mp3", ".m4a", ".m4b", ".wma", ".flac", ".ogg"}
    audiobook_folders = []

    LOG.info("Scanning for audiobook folders in: %s", root_path)

    for folder in root_path.rglob("*"):
        if not folder.is_dir():
            continue

        # Skip if this is the output folder
        if "_AudioSort_output_" in folder.parts:
            continue

        # Check if folder contains audio files
        has_audio = any(
            file.suffix.lower() in audio_extensions
            for file in folder.iterdir()
            if file.is_file()
        )

        if has_audio:
            # Check if parent folder also has audio (to avoid duplicates)
            parent_has_audio = any(
                file.suffix.lower() in audio_extensions
                for file in folder.parent.iterdir()
                if file.is_file()
            ) if folder.parent != root_path else False

            # Only add if parent doesn't have audio (this is the top-level folder with audio)
            if not parent_has_audio:
                audiobook_folders.append(folder)
                LOG.debug("Found audiobook folder: %s", folder)

    LOG.info("Found %d audiobook folders", len(audiobook_folders))
    return audiobook_folders


def create_fallback_metadata(folder: Path) -> BookMetadata:
    """
    Create fallback metadata by parsing folder name.
    Examples:
    - "J.K. Rowling - Harry Potter - T1" → J.K. Rowling, Harry Potter T1
    - "Stephen King - Ca" → Stephen King, Ca
    - "T1 - Harry Potter - L'Ecole des Sorciers" → Harry Potter, L'Ecole des Sorciers T1
    """
    import re

    folder_name = folder.name
    metadata = BookMetadata(url="", title=folder_name)

    # Default author and title extraction
    if " - " in folder_name:
        parts = [p.strip() for p in folder_name.split(" - ")]
        if len(parts) >= 2:
            # Check if first part looks like a series number (T1, Tome 1, etc.)
            if re.match(r'^T?\d+$', parts[0]) or re.match(r'^Tome\s*\d+$', parts[0], re.IGNORECASE):
                # This is a series number, look for author in other parts or use default
                metadata.authors = ["Unknown Author"]
                metadata.title = " - ".join(parts[1:])
            else:
                # First part might be an author
                if any(char.isupper() for char in parts[0]) and len(parts[0].split()) <= 4:
                    metadata.authors = [parts[0]]
                    metadata.title = " - ".join(parts[1:])
                else:
                    metadata.title = folder_name
                    metadata.authors = ["Unknown Author"]
        else:
            metadata.title = folder_name
            metadata.authors = ["Unknown Author"]
    else:
        metadata.title = folder_name
        metadata.authors = ["Unknown Author"]

    # Special handling for Harry Potter
    if "Harry Potter" in folder_name:
        metadata.authors = ["J.K. Rowling"]
        metadata.series = "Harry Potter"

        # Extract series number
        series_match = re.search(r'T(?:ome)?\s*(\d+)', folder_name, re.IGNORECASE)
        if series_match:
            metadata.series_position = series_match.group(1)
            # Remove "T1", "Tome 1", etc. from title
            metadata.title = re.sub(r'^\s*T(?:ome)?\s*\d+\s*[-\s]*', '', metadata.title, flags=re.IGNORECASE).strip()

        # Remove [mp3 64kbps] from title
        metadata.title = metadata.title.replace("[mp3 64kbps]", "").strip()

    # General series info extraction for other books
    else:
        series_match = re.search(r'T(?:ome)?\s*(\d+)', folder_name, re.IGNORECASE)
        if series_match:
            series_number = series_match.group(1)
            # Remove "T1", "Tome 1", etc. from title
            metadata.title = re.sub(r'[-\s]*T(?:ome)?\s*\d+', '', metadata.title, flags=re.IGNORECASE).strip()
            # Try to guess series name from title
            if len(metadata.title.split()) > 2:
                # Use first 2-3 words as series name
                words = metadata.title.split()
                metadata.series = " ".join(words[:2])
                metadata.series_position = series_number

    # Clean up title
    metadata.title = metadata.title.replace("[mp3 64kbps]", "").strip()

    return metadata


def infer_search_term(folder: Path) -> str:
    for audio in folder.rglob("*"):
        if audio.suffix.lower() in {".mp3", ".m4a", ".m4b", ".wma", ".flac", ".ogg"}:
            try:
                tag = TinyTag.get(str(audio))
            except Exception:
                continue
            album = (tag.album or "").strip()
            artist = (tag.artist or "").strip()
            if album and artist:
                return f"{album} by {artist}"
            if album:
                return album
    return folder.name


def enrich_with_id3_defaults(metadata: BookMetadata, folder: Path) -> None:
    if metadata.title:
        return
    metadata.title = folder.name


def _summarize(successes: list[str], failures: list[str], skipped: list[str]) -> None:
    print()
    if successes:
        print("=== SUCCESS ===")
        for entry in successes:
            print(entry)
    if skipped:
        print("\n=== SKIPPED ===")
        for entry in skipped:
            print(entry)
    if failures:
        print("\n=== FAILED ===")
        for entry in failures:
            print(entry)


if __name__ == "__main__":
    sys.exit(main())
