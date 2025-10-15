from __future__ import annotations

import re
from typing import Iterable, List

import requests
from bs4 import BeautifulSoup

from .metadata_sources import USER_AGENT


SEARCH_ENDPOINT = "https://duckduckgo.com/html/"


def auto_search(search_term: str, site: str, session: requests.Session, limit: int = 3) -> List[str]:
    queries = _build_queries(search_term, site)
    results: list[str] = []
    for query in queries:
        response = session.get(SEARCH_ENDPOINT, params={"q": query, "kl": "us-en"}, headers={"user-agent": USER_AGENT}, timeout=20)
        if not response.ok:
            continue
        soup = BeautifulSoup(response.text, "html.parser")
        for anchor in soup.select("a.result__a"):
            href = anchor.get("href", "")
            if not href:
                continue
            url_match = re.search(r"uddg=([^&]+)", href)
            candidate = url_match.group(1) if url_match else href
            candidate = requests.utils.unquote(candidate)
            if _matches_site(candidate, site) and candidate not in results:
                results.append(candidate)
                if len(results) >= limit:
                    return results
    return results


def _build_queries(search_term: str, site: str) -> Iterable[str]:
    if site == "audible":
        return [f"site:audible.com {search_term}"]
    if site == "goodreads":
        return [f"site:goodreads.com {search_term}"]
    return [f"site:audible.com {search_term}", f"site:goodreads.com {search_term}"]


def _matches_site(url: str, site: str) -> bool:
    if site == "audible":
        return "audible" in url
    if site == "goodreads":
        return "goodreads" in url
    return "audible" in url or "goodreads" in url
