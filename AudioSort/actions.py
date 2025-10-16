from __future__ import annotations

import json
import re
import shutil
from pathlib import Path
from typing import Iterable

import requests

from .config import AudioSortConfig
from .metadata_sources import USER_AGENT
from .models import BookMetadata, ProcessingPlan


AUDIO_EXTENSIONS = {".mp3", ".m4b", ".m4a", ".ogg", ".flac", ".wma"}


def prepare_output(metadata: BookMetadata, plan: ProcessingPlan, config: AudioSortConfig | None = None) -> Path:
    """
    Prepare destination using configuration and intelligent algorithm
    """
    if config is None:
        config = AudioSortConfig()

    # Use intelligent algorithm to suggest destination
    author_folder, title_folder = config.suggest_destination_based_on_existing(metadata, plan.destination_root)

    # Learn from metadata to improve future detections
    config.learn_from_metadata(metadata)

    destination = plan.destination_root / author_folder / title_folder
    if plan.dry_run:
        return destination
    destination.mkdir(parents=True, exist_ok=True)
    return destination


def _get_smart_title_folder(metadata: BookMetadata, plan: ProcessingPlan) -> str:
    """
    Smart title folder generation that groups series books together.

    Examples:
    - Harry Potter #1 + Harry Potter #2 → "Harry_Potter_Series"
    - Standalone book → "Book_Title"
    """

    # If it's part of a series, create a series folder
    if metadata.series and metadata.series_position:
        series_name = _slugify(metadata.series)

        # Check if there are already other books from this series
        author_folder = _slugify(metadata.primary_author()) or "_unknown_"
        existing_series_folders = _find_existing_series_folders(plan.destination_root / author_folder, series_name)

        if existing_series_folders:
            # Use the existing series folder name
            return existing_series_folders[0]
        else:
            # Create a new series folder
            return f"{series_name}_Series"

    # For standalone books, use the title
    return _slugify(metadata.title or plan.source_folder.name)


def _find_existing_series_folders(base_path: Path, series_name: str) -> list[str]:
    """Find existing folders that contain books from the same series."""
    if not base_path.exists():
        return []

    existing_folders = []
    for folder in base_path.iterdir():
        if folder.is_dir():
            # Check if folder name suggests it's from this series
            if series_name.lower() in folder.name.lower() or "series" in folder.name.lower():
                existing_folders.append(folder.name)

    return sorted(existing_folders)


def relocate_source(metadata: BookMetadata, plan: ProcessingPlan, destination: Path, config: AudioSortConfig | None = None) -> None:
    """
    Move/copy source files to destination with intelligent conflict management
    """
    if plan.dry_run:
        return

    if config is None:
        config = AudioSortConfig()

    # Check if destination already exists and contains files
    if destination.exists() and any(destination.iterdir()):
        print(f"📁 Adding to existing folder: {destination}")

        # Analyze potential conflicts
        existing_files = {f.name for f in destination.rglob("*") if f.is_file()}
        source_files = {f.name for f in plan.source_folder.rglob("*") if f.is_file()}
        conflicts = existing_files & source_files

        if conflicts:
            print(f"⚠️  {len(conflicts)} files will be updated:")
            for conflict in sorted(list(conflicts)[:5]):  # Show max 5 conflicts
                file_type = "📄 Metadata" if conflict.endswith(('.opf', '.txt', '.json')) else "🎵 Audio" if conflict.endswith(tuple(AUDIO_EXTENSIONS)) else "🖼️ Image"
                print(f"   {file_type} - {conflict}")
            if len(conflicts) > 5:
                print(f"   ... and {len(conflicts) - 5} other files")
            print()

            # Intelligent conflict management according to configuration
            conflict_resolution = config.get_conflict_resolution()
            if conflict_resolution == "merge":
                print("🔄 Merge mode: Existing files will be updated")
            elif conflict_resolution == "skip":
                if config.should_skip_existing_folders():
                    print("⏭️  Existing folder, processing ignored")
                    return

    # Show operation details
    operation = "📋 Copy" if plan.copy else "📋 Move"
    print(f"{operation} to: {destination}")

    # Show information about the book being added
    if metadata.series:
        print(f"📚 Series: {metadata.series} #{metadata.series_position}")
    print(f"📖 Title: {metadata.title}")
    print(f"✍️  Author: {metadata.primary_author()}")
    print()

    # Always use copytree with dirs_exist_ok=True to merge contents
    if plan.copy:
        shutil.copytree(plan.source_folder, destination, dirs_exist_ok=True)
    else:
        # Try to move first
        try:
            plan.source_folder.rename(destination)
        except OSError as e:
            print(f"⚠️  Unable to move folder (destination exists): {e}")
            print("📋 Merging content instead...")
            # Copy content then delete source
            shutil.copytree(plan.source_folder, destination, dirs_exist_ok=True)
            shutil.rmtree(plan.source_folder, ignore_errors=True)


def flatten(destination: Path, metadata: BookMetadata, dry_run: bool) -> None:
    if dry_run:
        return
    tracks = _collect_audio(destination, only_nested=True)
    if not tracks:
        return
    padding = 3 if len(tracks) >= 100 else 2
    for index, track in enumerate(sorted(tracks), start=1):
        new_name = f"{str(index).zfill(padding)} - {clean_title(metadata.title or track.stem)}{track.suffix}"
        track.rename(destination / new_name)
    for track in tracks:
        parent = track.parent
        if parent != destination and parent.exists():
            shutil.rmtree(parent, ignore_errors=True)


def rename_tracks(destination: Path, metadata: BookMetadata, dry_run: bool) -> None:
    if dry_run:
        return
    tracks = _collect_audio(destination)
    if not tracks:
        return
    padding = 3 if len(tracks) >= 100 else 2
    for index, track in enumerate(sorted(tracks), start=1):
        new_name = f"{str(index).zfill(padding)} - {clean_title(metadata.title or track.stem)}{track.suffix}"
        track.rename(track.with_name(new_name))


def write_opf(destination: Path, metadata: BookMetadata, template_path: Path, dry_run: bool) -> None:
    if dry_run:
        return
    if not template_path.is_file():
        return

    opf_file = destination / "metadata.opf"
    if opf_file.exists():
        print(f"📝 metadata.opf already exists, updating with new metadata...")

    content = template_path.read_text(encoding="utf-8")
    replacements = {
        "__AUTHOR__": metadata.primary_author(),
        "__TITLE__": metadata.title,
        "__SUMMARY__": metadata.summary,
        "__SUBTITLE__": metadata.subtitle,
        "__NARRATOR__": metadata.primary_narrator(),
        "__PUBLISHER__": metadata.publisher,
        "__PUBLISHYEAR__": metadata.publish_year,
        "__GENRES__": ", ".join(metadata.genres),
        "__ISBN__": metadata.isbn,
        "__ASIN__": metadata.asin,
        "__SERIES__": metadata.series,
        "__VOLUMENUMBER__": metadata.series_position,
    }
    for placeholder, value in replacements.items():
        content = content.replace(placeholder, value or "")
    opf_file.write_text(content, encoding="utf-8")


def write_info(destination: Path, metadata: BookMetadata, dry_run: bool) -> None:
    if dry_run:
        return
    if not metadata.summary:
        return

    info_file = destination / "info.txt"
    if info_file.exists():
        print(f"📝 info.txt already exists, updating with new metadata...")

    info_file.write_text(metadata.summary, encoding="utf-8")


def write_json(destination: Path, metadata: BookMetadata, dry_run: bool) -> None:
    if dry_run:
        return

    json_file = destination / "metadata.json"
    if json_file.exists():
        print(f"📝 metadata.json already exists, updating with new metadata...")

    json_file.write_text(json.dumps(metadata.as_dict(), indent=2, ensure_ascii=False), encoding="utf-8")


def download_cover(destination: Path, metadata: BookMetadata, session: requests.Session, dry_run: bool) -> None:
    if dry_run or not metadata.cover_url:
        return

    # Validate URL format before making request
    cover_url = metadata.cover_url.strip()
    if not cover_url.startswith(('http://', 'https://')):
        print(f"⚠️  Invalid cover URL format: {cover_url}")
        return

    response = session.get(cover_url, headers={"user-agent": USER_AGENT}, timeout=20)
    if not response.ok:
        return
    suffix = _guess_extension(response.headers.get("content-type", ""))
    cover_file = destination / f"cover{suffix}"

    if cover_file.exists():
        print(f"🖼️  Cover file already exists, updating with new cover...")

    cover_file.write_bytes(response.content)


def clean_title(value: str) -> str:
    return re.sub(r"[^\w\-\.\(\) ]+", "", value).strip()


def _slugify(value: str) -> str:
    value = clean_title(value)
    value = value.replace(" ", "_")
    return value


def _collect_audio(destination: Path, *, only_nested: bool = False) -> list[Path]:
    tracks: list[Path] = []
    for path in destination.rglob("*"):
        if path.is_file() and path.suffix.lower() in AUDIO_EXTENSIONS:
            if only_nested and path.parent == destination:
                continue
            tracks.append(path)
    return tracks


def _guess_extension(content_type: str) -> str:
    mapping = {
        "image/jpeg": ".jpg",
        "image/png": ".png",
        "image/webp": ".webp",
    }
    return mapping.get(content_type.lower(), ".jpg")
