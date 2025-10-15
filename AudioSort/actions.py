from __future__ import annotations

import json
import re
import shutil
from pathlib import Path
from typing import Iterable

import requests

from .metadata_sources import USER_AGENT
from .models import BookMetadata, ProcessingPlan


AUDIO_EXTENSIONS = {".mp3", ".m4b", ".m4a", ".ogg", ".flac", ".wma"}


def prepare_output(metadata: BookMetadata, plan: ProcessingPlan) -> Path:
    author_folder = _slugify(metadata.primary_author()) or "_unknown_"
    title_folder = _slugify(metadata.title or plan.source_folder.name)
    destination = plan.destination_root / author_folder / title_folder
    if plan.dry_run:
        return destination
    destination.mkdir(parents=True, exist_ok=True)
    return destination


def relocate_source(metadata: BookMetadata, plan: ProcessingPlan, destination: Path) -> None:
    if plan.dry_run:
        return
    if plan.copy:
        shutil.copytree(plan.source_folder, destination, dirs_exist_ok=True)
    else:
        try:
            plan.source_folder.rename(destination)
        except OSError:
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
    (destination / "metadata.opf").write_text(content, encoding="utf-8")


def write_info(destination: Path, metadata: BookMetadata, dry_run: bool) -> None:
    if dry_run:
        return
    if not metadata.summary:
        return
    (destination / "info.txt").write_text(metadata.summary, encoding="utf-8")


def write_json(destination: Path, metadata: BookMetadata, dry_run: bool) -> None:
    if dry_run:
        return
    (destination / "metadata.json").write_text(json.dumps(metadata.as_dict(), indent=2, ensure_ascii=False), encoding="utf-8")


def download_cover(destination: Path, metadata: BookMetadata, session: requests.Session, dry_run: bool) -> None:
    if dry_run or not metadata.cover_url:
        return
    response = session.get(metadata.cover_url, headers={"user-agent": USER_AGENT}, timeout=20)
    if not response.ok:
        return
    suffix = _guess_extension(response.headers.get("content-type", ""))
    filename = destination / f"cover{suffix}"
    filename.write_bytes(response.content)


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
