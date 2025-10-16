from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Dict

from .models import BookMetadata


class AudioSortConfig:
    """Configuration manager for AudioSort"""

    def __init__(self, config_path: Path | None = None):
        if config_path is None:
            # Look for the configuration file in the script directory
            script_dir = Path(__file__).parent.parent
            config_path = script_dir / "audiosort_config.json"

        self.config_path = config_path
        self._config: Dict[str, Any] = self._load_config()

    def _load_config(self) -> Dict[str, Any]:
        """Load configuration from JSON file"""
        if self.config_path.exists():
            try:
                with open(self.config_path, 'r', encoding='utf-8') as f:
                    return json.load(f)
            except (json.JSONDecodeError, IOError) as e:
                print(f"⚠️  Configuration loading error: {e}")
                print("📋 Using default configuration")

        # Default configuration
        return {
            "default_input_folder": "",
            "default_output_folder": "_AudioSort_output_",
            "last_used_settings": {
                "scan": True,
                "auto": True,
                "flatten": True,
                "rename": True,
                "opf": True,
                "infotxt": True,
                "cover": True,
                "copy": False
            },
            "author_mappings": {},
            "series_mappings": {},
            "preferred_metadata_source": "both",
            "auto_create_config": True,
            "conflict_resolution": "merge",
            "skip_existing_folders": False
        }

    def save_config(self) -> None:
        """Save configuration to JSON file"""
        try:
            with open(self.config_path, 'w', encoding='utf-8') as f:
                json.dump(self._config, f, indent=2, ensure_ascii=False)
        except IOError as e:
            print(f"⚠️  Configuration save error: {e}")

    def get_default_input_folder(self) -> str:
        """Get default input folder"""
        return self._config.get("default_input_folder", "")

    def set_default_input_folder(self, folder_path: str) -> None:
        """Set default input folder"""
        self._config["default_input_folder"] = str(folder_path)
        self.save_config()

    def get_default_output_folder(self) -> str:
        """Get default output folder"""
        return self._config.get("default_output_folder", "_AudioSort_output_")

    def set_default_output_folder(self, folder_path: str) -> None:
        """Set default output folder"""
        self._config["default_output_folder"] = str(folder_path)
        self.save_config()

    def get_last_settings(self) -> Dict[str, Any]:
        """Get last used settings"""
        return self._config.get("last_used_settings", {})

    def save_last_settings(self, settings: Dict[str, Any]) -> None:
        """Save last used settings"""
        self._config["last_used_settings"] = settings
        self.save_config()

    def get_author_mapping(self, author_name: str) -> str | None:
        """Get author mapping"""
        return self._config.get("author_mappings", {}).get(author_name)

    def add_author_mapping(self, original_name: str, corrected_name: str) -> None:
        """Add author mapping"""
        if "author_mappings" not in self._config:
            self._config["author_mappings"] = {}
        self._config["author_mappings"][original_name] = corrected_name
        self.save_config()

    def get_series_mapping(self, series_name: str) -> str | None:
        """Get series mapping"""
        return self._config.get("series_mappings", {}).get(series_name)

    def add_series_mapping(self, original_name: str, corrected_name: str) -> None:
        """Add series mapping"""
        if "series_mappings" not in self._config:
            self._config["series_mappings"] = {}
        self._config["series_mappings"][original_name] = corrected_name
        self.save_config()

    def should_skip_existing_folders(self) -> bool:
        """Check if existing folders should be skipped"""
        return self._config.get("skip_existing_folders", False)

    def get_conflict_resolution(self) -> str:
        """Get conflict resolution method"""
        return self._config.get("conflict_resolution", "merge")

    def learn_from_metadata(self, metadata: BookMetadata) -> None:
        """Learn from metadata to improve future processing"""
        # Learn author names
        for author in metadata.authors:
            if author and author != "Unknown Author":
                # Clean and normalize the name
                clean_author = self._clean_author_name(author)
                if clean_author != author:
                    self.add_author_mapping(author, clean_author)

        # Learn series names
        if metadata.series:
            clean_series = self._clean_series_name(metadata.series)
            if clean_series != metadata.series:
                self.add_series_mapping(metadata.series, clean_series)

    def _clean_author_name(self, author: str) -> str:
        """Clean and normalize an author name"""
        # Remove multiple spaces
        author = ' '.join(author.split())

        # Correct common variations
        corrections = {
            "J.K Rowling": "J.K. Rowling",
            "J K Rowling": "J.K. Rowling",
            "Stephen King": "Stephen King",
            "Agatha Christie": "Agatha Christie",
        }

        # Basic correction of periods and spaces
        for incorrect, correct in corrections.items():
            if author.lower() == incorrect.lower():
                return correct

        return author

    def _clean_series_name(self, series: str) -> str:
        """Clean and normalize a series name"""
        # Remove multiple spaces
        series = ' '.join(series.split())

        # Common corrections
        corrections = {
            "Harry Potter": "Harry Potter",
            "Harry Potter (French)": "Harry Potter",
        }

        for incorrect, correct in corrections.items():
            if series.lower() == incorrect.lower():
                return correct

        return series

    def get_existing_authors_in_output(self, output_folder: Path) -> Dict[str, Path]:
        """Analyze existing authors in the output folder"""
        if not output_folder.exists():
            return {}

        authors = {}
        for author_folder in output_folder.iterdir():
            if author_folder.is_dir() and not author_folder.name.startswith('_'):
                # Clean folder name to get author name
                author_name = author_folder.name.replace('_', ' ')
                authors[author_name.lower()] = author_folder

        return authors

    def get_existing_series_for_author(self, output_folder: Path, author_name: str) -> Dict[str, Path]:
        """Analyze existing series for an author"""
        if not output_folder.exists():
            return {}

        # Find author folder
        author_folder_name = author_name.replace(' ', '_')
        author_folder = output_folder / author_folder_name

        if not author_folder.exists():
            # Look for variations
            for folder in output_folder.iterdir():
                if folder.is_dir() and folder.name.replace('_', ' ').lower() == author_name.lower():
                    author_folder = folder
                    break
            else:
                return {}

        series = {}
        for series_folder in author_folder.iterdir():
            if series_folder.is_dir():
                # Identify if it's a series or a solo book
                folder_name = series_folder.name.replace('_', ' ')
                if "series" in folder_name.lower():
                    # Extract series name
                    series_name = folder_name.replace(" series", "").strip()
                    series[series_name.lower()] = series_folder
                else:
                    # It's probably a solo book, but we can still register it
                    series[folder_name.lower()] = series_folder

        return series

    def suggest_destination_based_on_existing(self, metadata: BookMetadata, output_folder: Path) -> tuple[str, str]:
        """
        Suggest destination based on existing authors and series

        Returns:
            tuple: (author_folder, title_folder)
        """
        # Get existing authors
        existing_authors = self.get_existing_authors_in_output(output_folder)

        # Look for author match
        author_name = metadata.primary_author()

        # Check author mappings
        mapped_author = self.get_author_mapping(author_name)
        if mapped_author:
            author_name = mapped_author

        # Look if this author already exists
        author_folder = None
        for existing_name_lower, existing_path in existing_authors.items():
            if author_name.lower() == existing_name_lower:
                author_folder = existing_path.name
                break

        # If no author found, create a new one
        if not author_folder:
            author_folder = author_name.replace(' ', '_')

        # Now look for series for this author
        if metadata.series:
            existing_series = self.get_existing_series_for_author(output_folder, author_name)

            # Check series mappings
            series_name = metadata.series
            mapped_series = self.get_series_mapping(series_name)
            if mapped_series:
                series_name = mapped_series

            # Look if this series already exists
            for existing_name_lower, existing_path in existing_series.items():
                if series_name.lower() == existing_name_lower:
                    # Use existing series folder
                    return author_folder, existing_path.name

            # Create new series folder
            series_folder = f"{series_name.replace(' ', '_')}_Series"
            return author_folder, series_folder

        # For a solo book
        title_folder = metadata.title.replace(' ', '_') if metadata.title else "Unknown_Title"
        return author_folder, title_folder