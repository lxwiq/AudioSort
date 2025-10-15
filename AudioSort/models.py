from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import List


@dataclass(slots=True)
class BookMetadata:
    title: str = ""
    authors: List[str] = field(default_factory=list)
    subtitle: str = ""
    summary: str = ""
    narrators: List[str] = field(default_factory=list)
    publisher: str = ""
    publish_year: str = ""
    genres: List[str] = field(default_factory=list)
    isbn: str = ""
    asin: str = ""
    series: str = ""
    series_position: str = ""
    duration_minutes: str = ""
    language: str = ""
    cover_url: str = ""
    url: str = ""

    def primary_author(self) -> str:
        return self.authors[0] if self.authors else "_unknown_"

    def primary_narrator(self) -> str:
        return self.narrators[0] if self.narrators else ""

    def as_dict(self) -> dict[str, object]:
        return {
            "title": self.title,
            "authors": self.authors,
            "subtitle": self.subtitle,
            "summary": self.summary,
            "narrators": self.narrators,
            "publisher": self.publisher,
            "publish_year": self.publish_year,
            "genres": self.genres,
            "isbn": self.isbn,
            "asin": self.asin,
            "series": self.series,
            "series_position": self.series_position,
            "duration_minutes": self.duration_minutes,
            "language": self.language,
            "cover_url": self.cover_url,
            "url": self.url,
        }


@dataclass(slots=True)
class ProcessingPlan:
    source_folder: Path
    destination_root: Path
    copy: bool
    flatten: bool
    rename_tracks: bool
    create_opf: bool
    create_info: bool
    download_cover: bool
    emit_json: bool
    dry_run: bool
