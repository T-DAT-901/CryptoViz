"""Cryptocurrency symbol detection."""
from typing import List, Dict


class CryptoDetector:
    """Detects cryptocurrency symbols in text."""

    def __init__(self, symbols_config: List[Dict[str, any]]):
        """
        Initialize detector with symbol configuration.

        Args:
            symbols_config: List of dicts with 'name' and 'keywords'
        """
        self.symbols = {}
        for symbol_config in symbols_config:
            name = symbol_config.get("name", "").lower()
            keywords = [kw.lower() for kw in symbol_config.get("keywords", [])]
            self.symbols[name] = keywords

    def detect(self, text: str) -> List[str]:
        """
        Detect crypto symbols in text.

        Args:
            text: Text to analyze (title + content)

        Returns:
            List of detected symbol names (e.g., ["btc", "eth"])
        """
        text_lower = text.lower()
        found = []

        for symbol, keywords in self.symbols.items():
            if any(keyword in text_lower for keyword in keywords):
                found.append(symbol)

        return found

    def detect_in_article(self, title: str, content: str = "") -> List[str]:
        """
        Detect symbols in article title and content.

        Args:
            title: Article title
            content: Article content (optional)

        Returns:
            List of detected symbols
        """
        combined_text = f"{title} {content}"
        return self.detect(combined_text)
