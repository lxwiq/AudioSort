from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Dict

from .models import BookMetadata


class AudioSortConfig:
    """Gestionnaire de configuration pour AudioSort"""

    def __init__(self, config_path: Path | None = None):
        if config_path is None:
            # Chercher le fichier de configuration dans le répertoire du script
            script_dir = Path(__file__).parent.parent
            config_path = script_dir / "audiosort_config.json"

        self.config_path = config_path
        self._config: Dict[str, Any] = self._load_config()

    def _load_config(self) -> Dict[str, Any]:
        """Charger la configuration depuis le fichier JSON"""
        if self.config_path.exists():
            try:
                with open(self.config_path, 'r', encoding='utf-8') as f:
                    return json.load(f)
            except (json.JSONDecodeError, IOError) as e:
                print(f"⚠️  Erreur de chargement de la configuration: {e}")
                print("📋 Utilisation de la configuration par défaut")

        # Configuration par défaut
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
        """Sauvegarder la configuration dans le fichier JSON"""
        try:
            with open(self.config_path, 'w', encoding='utf-8') as f:
                json.dump(self._config, f, indent=2, ensure_ascii=False)
        except IOError as e:
            print(f"⚠️  Erreur de sauvegarde de la configuration: {e}")

    def get_default_input_folder(self) -> str:
        """Récupérer le dossier d'entrée par défaut"""
        return self._config.get("default_input_folder", "")

    def set_default_input_folder(self, folder_path: str) -> None:
        """Définir le dossier d'entrée par défaut"""
        self._config["default_input_folder"] = str(folder_path)
        self.save_config()

    def get_default_output_folder(self) -> str:
        """Récupérer le dossier de sortie par défaut"""
        return self._config.get("default_output_folder", "_AudioSort_output_")

    def set_default_output_folder(self, folder_path: str) -> None:
        """Définir le dossier de sortie par défaut"""
        self._config["default_output_folder"] = str(folder_path)
        self.save_config()

    def get_last_settings(self) -> Dict[str, Any]:
        """Récupérer les derniers paramètres utilisés"""
        return self._config.get("last_used_settings", {})

    def save_last_settings(self, settings: Dict[str, Any]) -> None:
        """Sauvegarder les derniers paramètres utilisés"""
        self._config["last_used_settings"] = settings
        self.save_config()

    def get_author_mapping(self, author_name: str) -> str | None:
        """Récupérer un mapping d'auteur"""
        return self._config.get("author_mappings", {}).get(author_name)

    def add_author_mapping(self, original_name: str, corrected_name: str) -> None:
        """Ajouter un mapping d'auteur"""
        if "author_mappings" not in self._config:
            self._config["author_mappings"] = {}
        self._config["author_mappings"][original_name] = corrected_name
        self.save_config()

    def get_series_mapping(self, series_name: str) -> str | None:
        """Récupérer un mapping de série"""
        return self._config.get("series_mappings", {}).get(series_name)

    def add_series_mapping(self, original_name: str, corrected_name: str) -> None:
        """Ajouter un mapping de série"""
        if "series_mappings" not in self._config:
            self._config["series_mappings"] = {}
        self._config["series_mappings"][original_name] = corrected_name
        self.save_config()

    def should_skip_existing_folders(self) -> bool:
        """Vérifier s'il faut sauter les dossiers existants"""
        return self._config.get("skip_existing_folders", False)

    def get_conflict_resolution(self) -> str:
        """Récupérer la méthode de résolution de conflits"""
        return self._config.get("conflict_resolution", "merge")

    def learn_from_metadata(self, metadata: BookMetadata) -> None:
        """Apprendre des métadonnées pour améliorer les futurs traitements"""
        # Apprendre les noms d'auteurs
        for author in metadata.authors:
            if author and author != "Unknown Author":
                # Nettoyer et normaliser le nom
                clean_author = self._clean_author_name(author)
                if clean_author != author:
                    self.add_author_mapping(author, clean_author)

        # Apprendre les noms de séries
        if metadata.series:
            clean_series = self._clean_series_name(metadata.series)
            if clean_series != metadata.series:
                self.add_series_mapping(metadata.series, clean_series)

    def _clean_author_name(self, author: str) -> str:
        """Nettoyer et normaliser un nom d'auteur"""
        # Supprimer les espaces multiples
        author = ' '.join(author.split())

        # Corriger les variations courantes
        corrections = {
            "J.K Rowling": "J.K. Rowling",
            "J K Rowling": "J.K. Rowling",
            "Stephen King": "Stephen King",
            "Agatha Christie": "Agatha Christie",
        }

        # Correction basique des points et espaces
        for incorrect, correct in corrections.items():
            if author.lower() == incorrect.lower():
                return correct

        return author

    def _clean_series_name(self, series: str) -> str:
        """Nettoyer et normaliser un nom de série"""
        # Supprimer les espaces multiples
        series = ' '.join(series.split())

        # Corrections courantes
        corrections = {
            "Harry Potter": "Harry Potter",
            "Harry Potter (French)": "Harry Potter",
        }

        for incorrect, correct in corrections.items():
            if series.lower() == incorrect.lower():
                return correct

        return series

    def get_existing_authors_in_output(self, output_folder: Path) -> Dict[str, Path]:
        """Analyser les auteurs existants dans le dossier de sortie"""
        if not output_folder.exists():
            return {}

        authors = {}
        for author_folder in output_folder.iterdir():
            if author_folder.is_dir() and not author_folder.name.startswith('_'):
                # Nettoyer le nom du dossier pour obtenir le nom de l'auteur
                author_name = author_folder.name.replace('_', ' ')
                authors[author_name.lower()] = author_folder

        return authors

    def get_existing_series_for_author(self, output_folder: Path, author_name: str) -> Dict[str, Path]:
        """Analyser les séries existantes pour un auteur"""
        if not output_folder.exists():
            return {}

        # Trouver le dossier de l'auteur
        author_folder_name = author_name.replace(' ', '_')
        author_folder = output_folder / author_folder_name

        if not author_folder.exists():
            # Chercher des variations
            for folder in output_folder.iterdir():
                if folder.is_dir() and folder.name.replace('_', ' ').lower() == author_name.lower():
                    author_folder = folder
                    break
            else:
                return {}

        series = {}
        for series_folder in author_folder.iterdir():
            if series_folder.is_dir():
                # Identifier si c'est une série ou un livre solo
                folder_name = series_folder.name.replace('_', ' ')
                if "series" in folder_name.lower():
                    # Extraire le nom de la série
                    series_name = folder_name.replace(" series", "").strip()
                    series[series_name.lower()] = series_folder
                else:
                    # C'est probablement un livre solo, mais on peut quand même l'enregistrer
                    series[folder_name.lower()] = series_folder

        return series

    def suggest_destination_based_on_existing(self, metadata: BookMetadata, output_folder: Path) -> tuple[str, str]:
        """
        Suggérer une destination basée sur les auteurs et séries existants

        Returns:
            tuple: (author_folder, title_folder)
        """
        # Récupérer les auteurs existants
        existing_authors = self.get_existing_authors_in_output(output_folder)

        # Chercher une correspondance d'auteur
        author_name = metadata.primary_author()

        # Vérifier les mappings d'auteurs
        mapped_author = self.get_author_mapping(author_name)
        if mapped_author:
            author_name = mapped_author

        # Chercher si cet auteur existe déjà
        author_folder = None
        for existing_name_lower, existing_path in existing_authors.items():
            if author_name.lower() == existing_name_lower:
                author_folder = existing_path.name
                break

        # Si pas d'auteur trouvé, créer un nouveau
        if not author_folder:
            author_folder = author_name.replace(' ', '_')

        # Maintenant chercher les séries pour cet auteur
        if metadata.series:
            existing_series = self.get_existing_series_for_author(output_folder, author_name)

            # Vérifier les mappings de séries
            series_name = metadata.series
            mapped_series = self.get_series_mapping(series_name)
            if mapped_series:
                series_name = mapped_series

            # Chercher si cette série existe déjà
            for existing_name_lower, existing_path in existing_series.items():
                if series_name.lower() == existing_name_lower:
                    # Utiliser le dossier de série existant
                    return author_folder, existing_path.name

            # Créer un nouveau dossier de série
            series_folder = f"{series_name.replace(' ', '_')}_Series"
            return author_folder, series_folder

        # Pour un livre solo
        title_folder = metadata.title.replace(' ', '_') if metadata.title else "Unknown_Title"
        return author_folder, title_folder