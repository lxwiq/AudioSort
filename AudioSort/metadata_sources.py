from __future__ import annotations

import json
import re
from typing import Callable, Optional

import requests
from bs4 import BeautifulSoup

from .models import BookMetadata


USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/117.0"


class MetadataError(RuntimeError):
    pass


def fetch_metadata(url: str, session: requests.Session) -> BookMetadata:
    url = url.strip()
    if "audible" in url:
        return _fetch_audible(url, session)
    if "goodreads" in url:
        return _fetch_goodreads(url, session)
    if "books.google.com" in url or "play.google.com" in url:
        return _fetch_google_books(url, session)
    if "openlibrary.org" in url:
        return _fetch_open_library(url, session)
    raise MetadataError(f"Unsupported metadata url: {url}")


def fetch_metadata_by_search(search_term: str, session: requests.Session) -> BookMetadata | None:
    """
    Try to fetch metadata by searching multiple free sources.
    Returns the first successful result.
    """
    # Try Google Books first
    metadata = _search_google_books(search_term, session)
    if metadata:
        return metadata

    # Try Open Library
    metadata = _search_open_library(search_term, session)
    if metadata:
        return metadata

    return None


def _fetch_audible(url: str, session: requests.Session) -> BookMetadata:
    asin_match = re.search(r"/pd/[\w-]+Audiobook/(\w{10})", url)
    metadata = BookMetadata(url=url)
    if asin_match:
        asin = asin_match.group(1)
        metadata.asin = asin
        api_url = f"https://api.audible.com/1.0/catalog/products/{asin}"
        params = {"response_groups": "contributors,product_desc,series,product_extended_attrs,media"}
        response = session.get(api_url, params=params, headers={"user-agent": USER_AGENT}, timeout=20)
        if response.ok:
            product = response.json().get("product")
            if product:
                return _audible_from_product(url, product)

    response = session.get(url, headers={"user-agent": USER_AGENT}, timeout=20)
    if not response.ok:
        raise MetadataError(f"Audible request failed with {response.status_code}")
    soup = BeautifulSoup(response.text, "html.parser")
    json_ld = _first_json_ld(soup, predicate=lambda d: d.get("@type") in {"Audiobook", "Book"})
    if not json_ld:
        raise MetadataError("Audible page did not expose structured metadata")
    metadata.title = json_ld.get("name", "")
    metadata.authors = _normalize_people(json_ld.get("author"))
    metadata.subtitle = json_ld.get("alternateName", "")
    metadata.summary = json_ld.get("description", "")
    metadata.narrators = _normalize_people(json_ld.get("readBy"))
    metadata.publisher = json_ld.get("publisher", {}).get("name", "") if isinstance(json_ld.get("publisher"), dict) else json_ld.get("publisher", "")
    metadata.publish_year = _extract_year(json_ld.get("datePublished", ""))
    metadata.genres = _as_list(json_ld.get("genre"))
    metadata.isbn = json_ld.get("isbn", "")
    if not metadata.asin:
        asin_candidate = json_ld.get("sku") or json_ld.get("productID")
        if isinstance(asin_candidate, str) and asin_candidate.startswith("B0"):
            metadata.asin = asin_candidate
    metadata.cover_url = _resolve_cover(json_ld)
    metadata.language = json_ld.get("inLanguage", "")
    duration = json_ld.get("duration", "")
    if isinstance(duration, str):
        metadata.duration_minutes = _duration_iso8601_to_minutes(duration)
    return metadata


def _audible_from_product(url: str, product: dict) -> BookMetadata:
    metadata = BookMetadata(url=url)
    metadata.title = product.get("title", "")
    metadata.subtitle = product.get("subtitle", "")
    metadata.summary = _clean_html(product.get("publisher_summary", ""))
    metadata.publisher = product.get("publisher_name", "")
    metadata.publish_year = _extract_year(product.get("release_date", ""))
    metadata.language = (product.get("language", {}) or {}).get("name", "") if isinstance(product.get("language"), dict) else product.get("language", "")
    metadata.duration_minutes = str(product.get("runtime_length_min", "")) if product.get("runtime_length_min") else ""
    metadata.asin = product.get("asin", "")
    metadata.genres = [category.get("name", "") for category in product.get("categories", []) if category.get("name")]
    metadata.authors = [author.get("name", "") for author in product.get("authors", []) if author.get("name")]
    metadata.narrators = [narrator.get("name", "") for narrator in product.get("narrators", []) if narrator.get("name")]
    if product.get("series"):
        metadata.series = product["series"][0].get("title", "")
        metadata.series_position = str(product["series"][0].get("sequence", ""))
    metadata.cover_url = _cover_from_product(product)
    metadata.isbn = product.get("isbn", "")
    return metadata


def _cover_from_product(product: dict) -> str:
    images = product.get("product_images") or []
    for image in images:
        if isinstance(image, str):
            return image
        if isinstance(image, dict):
            url = image.get("image_url") or image.get("https_image_url")
            if url:
                return url
    return ""


def _fetch_goodreads(url: str, session: requests.Session) -> BookMetadata:
    response = session.get(url, headers={"user-agent": USER_AGENT}, timeout=20)
    if not response.ok:
        raise MetadataError(f"Goodreads request failed with {response.status_code}")
    soup = BeautifulSoup(response.text, "html.parser")
    metadata = BookMetadata(url=url)

    ld = _first_json_ld(soup, predicate=lambda d: d.get("@type") in {"Book", "CreativeWork"})
    if ld:
        metadata.title = _extract_title(ld)
        metadata.subtitle = ld.get("alternateName", "")
        metadata.authors = _normalize_people(ld.get("author"))
        metadata.summary = ld.get("description", "")
        metadata.publisher = ld.get("publisher", {}).get("name", "") if isinstance(ld.get("publisher"), dict) else ld.get("publisher", "")
        metadata.publish_year = _extract_year(ld.get("datePublished", ""))
        metadata.genres = _as_list(ld.get("genre"))
        metadata.isbn = ld.get("isbn", "")
        metadata.series = _extract_series_from_title(ld)
        metadata.series_position = _extract_series_position_from_title(ld)
        metadata.cover_url = ld.get("image", "") if isinstance(ld.get("image"), str) else ""

    if not metadata.title:
        title_node = soup.select_one("#bookTitle")
        if title_node:
            metadata.title = title_node.get_text(strip=True)
    if not metadata.authors:
        author_node = soup.select_one("#bookAuthors")
        if author_node:
            author_links = author_node.select("a.authorName")
            metadata.authors = [a.get_text(strip=True) for a in author_links if a.get_text(strip=True)]
    if not metadata.summary:
        description = soup.select_one("#description")
        if description:
            spans = description.find_all("span")
            if spans:
                metadata.summary = spans[-1].get_text(strip=True)
    if not metadata.series:
        series_node = soup.select_one("#bookSeries")
        if series_node:
            series_text = series_node.get_text(strip=True)
            match = re.search(r"^(.+?)(?:,?\s+#\d+.*)?$", series_text)
            if match:
                metadata.series = match.group(1)
            pos = re.search(r"#([\d\.]+)", series_text)
            if pos:
                metadata.series_position = pos.group(1)
    return metadata


def _first_json_ld(soup: BeautifulSoup, predicate: Optional[Callable[[dict], bool]] = None) -> Optional[dict]:
    scripts = soup.find_all("script", attrs={"type": "application/ld+json"})
    for script in scripts:
        try:
            data = json.loads(script.string or "{}")
        except json.JSONDecodeError:
            continue
        if isinstance(data, list):
            for entry in data:
                if isinstance(entry, dict) and (predicate is None or predicate(entry)):
                    return entry
        elif isinstance(data, dict):
            if predicate is None or predicate(data):
                return data
    return None


def _normalize_people(value) -> list[str]:
    if isinstance(value, dict):
        name = value.get("name")
        return [name] if name else []
    if isinstance(value, list):
        result = []
        for item in value:
            if isinstance(item, dict):
                name = item.get("name")
                if name:
                    result.append(name)
            elif isinstance(item, str):
                result.append(item)
        return result
    if isinstance(value, str):
        return [value]
    return []


def _extract_year(value: str) -> str:
    if not value:
        return ""
    match = re.search(r"(\d{4})", value)
    return match.group(1) if match else ""


def _duration_iso8601_to_minutes(value: str) -> str:
    match = re.search(r"PT(?:(\d+)H)?(?:(\d+)M)?", value)
    if not match:
        return ""
    hours = int(match.group(1) or 0)
    minutes = int(match.group(2) or 0)
    total = hours * 60 + minutes
    return str(total) if total else ""


def _clean_html(value: str) -> str:
    if not value:
        return ""
    return BeautifulSoup(value, "html.parser").get_text(separator=" ", strip=True)


def _as_list(value) -> list[str]:
    if not value:
        return []
    if isinstance(value, list):
        return [str(v) for v in value if v]
    if isinstance(value, str):
        return [value]
    return []


def _resolve_cover(data: dict) -> str:
    image = data.get("image")
    if isinstance(image, str):
        return image
    if isinstance(image, list):
        for entry in image:
            if isinstance(entry, str):
                return entry
            if isinstance(entry, dict) and entry.get("url"):
                return entry["url"]
    if isinstance(image, dict):
        return image.get("url", "")
    return ""


def _extract_title(data: dict) -> str:
    name = data.get("name", "")
    if not name:
        return ""
    match = re.search(r"^(.+?)(\s*\(.*\))?$", name)
    return match.group(1) if match else name


def _extract_series_from_title(data: dict) -> str:
    title = data.get("name", "")
    match = re.search(r"\((.+?),\s*#\d+[\w\.]*\)", title)
    return match.group(1) if match else ""


def _extract_series_position_from_title(data: dict) -> str:
    title = data.get("name", "")
    match = re.search(r"#([\d\.]+)\)", title)
    return match.group(1) if match else ""


def _search_google_books(search_term: str, session: requests.Session) -> BookMetadata | None:
    """Search Google Books API for metadata."""
    try:
        # Clean search term for better results
        clean_term = _clean_search_term(search_term)
        url = f"https://www.googleapis.com/books/v1/volumes"
        params = {
            "q": clean_term,
            "maxResults": 5,
            "langRestrict": "fr,en"  # Prioritize French and English
        }

        response = session.get(url, params=params, timeout=10)
        if not response.ok:
            return None

        data = response.json()
        items = data.get("items", [])
        if not items:
            return None

        # Take the first result
        book_data = items[0].get("volumeInfo", {})
        return _parse_google_books_data(book_data, search_term)

    except Exception:
        return None


def _parse_google_books_data(book_data: dict, original_search_term: str) -> BookMetadata:
    """Parse Google Books API response into BookMetadata."""
    metadata = BookMetadata(url="")

    # Title
    metadata.title = book_data.get("title", original_search_term)

    # Authors
    authors = book_data.get("authors", [])
    if authors:
        metadata.authors = authors

    # Description
    metadata.summary = book_data.get("description", "")

    # Publisher and year
    publisher = book_data.get("publisher", "")
    published_date = book_data.get("publishedDate", "")
    if publisher:
        metadata.publisher = publisher
    if published_date:
        metadata.publish_year = _extract_year(published_date)

    # ISBN
    identifiers = book_data.get("industryIdentifiers", [])
    for identifier in identifiers:
        if identifier.get("type") == "ISBN_13":
            metadata.isbn = identifier.get("identifier", "")
            break
        elif identifier.get("type") == "ISBN_10" and not metadata.isbn:
            metadata.isbn = identifier.get("identifier", "")

    # Language
    metadata.language = book_data.get("language", "")

    # Categories (genres)
    categories = book_data.get("categories", [])
    if categories:
        metadata.genres = categories

    # Cover
    image_links = book_data.get("imageLinks", {})
    if image_links:
        # Prefer large cover, fallback to small
        metadata.cover_url = image_links.get("large") or image_links.get("thumbnail", "")

    # Series information - extract from title or description
    title = metadata.title
    desc = metadata.summary

    # Try to extract series from title
    series_match = re.search(r'(.+?),?\s+#(\d+)', title)
    if series_match:
        metadata.series = series_match.group(1).strip()
        metadata.series_position = series_match.group(2)

    # Try to extract series from description
    if not metadata.series and desc:
        series_match = re.search(r'(.+?)\s*book\s*(\d+)', desc, re.IGNORECASE)
        if series_match:
            metadata.series = series_match.group(1).strip()
            metadata.series_position = series_match.group(2)

    # Special handling for Harry Potter
    if "harry potter" in title.lower():
        metadata.series = "Harry Potter"
        # Extract book number from title or search term
        book_match = re.search(r'(\d+)', original_search_term)
        if book_match:
            metadata.series_position = book_match.group(1)
        else:
            # Try to extract from title
            book_titles = {
                "philosopher's stone": "1", "sorcerer's stone": "1",
                "chamber of secrets": "2", "prisoner of azkaban": "3",
                "goblet of fire": "4", "order of the phoenix": "5",
                "half-blood prince": "6", "deathly hallows": "7",
                "école des sorciers": "1", "chambre des secrets": "2",
                "prisonnier d'azkaban": "3", "coupe de feu": "4",
                "ordre du phénix": "5", "prince de sang-mêlé": "6",
                "reliques de la mort": "7"
            }
            title_lower = title.lower()
            for title_key, book_num in book_titles.items():
                if title_key in title_lower:
                    metadata.series_position = book_num
                    break

    return metadata


def _search_open_library(search_term: str, session: requests.Session) -> BookMetadata | None:
    """Search Open Library API for metadata."""
    try:
        clean_term = _clean_search_term(search_term)
        url = "https://openlibrary.org/search.json"
        params = {
            "q": clean_term,
            "limit": 5,
            "fields": "key,title,author_name,first_publish_year,publisher,isbn,cover_i,language,subject,first_sentence"
        }

        response = session.get(url, params=params, timeout=10)
        if not response.ok:
            return None

        data = response.json()
        docs = data.get("docs", [])
        if not docs:
            return None

        # Take the first result
        book_data = docs[0]
        return _parse_open_library_data(book_data, search_term)

    except Exception:
        return None


def _parse_open_library_data(book_data: dict, original_search_term: str) -> BookMetadata:
    """Parse Open Library API response into BookMetadata."""
    metadata = BookMetadata(url="")

    # Title
    metadata.title = book_data.get("title", original_search_term)

    # Authors
    authors = book_data.get("author_name", [])
    if authors:
        metadata.authors = authors

    # Publisher
    publishers = book_data.get("publisher", [])
    if publishers:
        metadata.publisher = publishers[0]

    # Year
    metadata.publish_year = str(book_data.get("first_publish_year", ""))

    # ISBN
    isbns = book_data.get("isbn", [])
    if isbns:
        metadata.isbn = isbns[0]

    # Language
    languages = book_data.get("language", [])
    if languages:
        metadata.language = languages[0]

    # Subjects (genres)
    subjects = book_data.get("subject", [])
    if subjects:
        metadata.genres = subjects[:5]  # Limit to first 5 subjects

    # Cover
    cover_id = book_data.get("cover_i")
    if cover_id:
        metadata.cover_url = f"https://covers.openlibrary.org/b/id/{cover_id}-L.jpg"

    # Series information from Open Library would require additional API calls
    # For now, try to extract from title
    title = metadata.title
    series_match = re.search(r'(.+?),?\s+#(\d+)', title)
    if series_match:
        metadata.series = series_match.group(1).strip()
        metadata.series_position = series_match.group(2)

    # Special handling for Harry Potter
    if "harry potter" in title.lower():
        metadata.series = "Harry Potter"
        metadata.authors = ["J.K. Rowling"]  # Override author if it's not correct

        # Extract book number
        book_match = re.search(r'(\d+)', original_search_term)
        if book_match:
            metadata.series_position = book_match.group(1)

    return metadata


def _clean_search_term(search_term: str) -> str:
    """Clean search term for better API results."""
    import re

    # Remove common audio file indicators
    term = re.sub(r'\[mp3.*?\]', '', search_term, flags=re.IGNORECASE)
    term = re.sub(r'\(audiobook\)', '', term, flags=re.IGNORECASE)
    term = re.sub(r'by\s.+$', '', term, flags=re.IGNORECASE)  # Remove narrator

    # Replace common separators with spaces
    term = re.sub(r'[-_]', ' ', term)

    # Remove "T1", "Tome 1", etc. for series search
    term = re.sub(r'^T?\d+\s*[-:]?\s*', '', term.strip())

    return term.strip()
