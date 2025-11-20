"""Sentiment analysis using VADER."""
from typing import Optional
from vaderSentiment.vaderSentiment import SentimentIntensityAnalyzer


class SentimentAnalyzer:
    """Analyzes sentiment of text using VADER."""

    def __init__(self, enabled: bool = True):
        """
        Initialize sentiment analyzer.

        Args:
            enabled: Whether sentiment analysis is enabled
        """
        self.enabled = enabled
        self.analyzer = SentimentIntensityAnalyzer() if enabled else None

    def analyze(self, text: str) -> Optional[float]:
        """
        Analyze sentiment of text.

        Args:
            text: Text to analyze

        Returns:
            Compound score from -1 (negative) to +1 (positive)
            Returns None if disabled or text is empty
        """
        if not self.enabled or not self.analyzer:
            return None

        if not text or not text.strip():
            return None

        scores = self.analyzer.polarity_scores(text)
        return scores["compound"]  # Compound score: -1 to +1

    def analyze_article(self, title: str, content: str = "") -> Optional[float]:
        """
        Analyze sentiment of article.

        Args:
            title: Article title
            content: Article content (optional)

        Returns:
            Sentiment score or None
        """
        # Weight title more heavily than content
        combined_text = f"{title} {title} {content}"
        return self.analyze(combined_text)
