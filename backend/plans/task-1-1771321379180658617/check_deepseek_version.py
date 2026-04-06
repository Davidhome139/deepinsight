#!/usr/bin/env python3
"""
Script to check for Deepseek's latest version using the GitHub API.
Fetches the latest release information and determines if it's recent based on current date.
"""

import re
import sys
import json
import logging
import argparse
from typing import Optional, Dict, Any
from datetime import datetime, timezone
import requests
from urllib.parse import urljoin

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('deepseek_version_check.log')
    ]
)
logger = logging.getLogger(__name__)


class DeepseekVersionChecker:
    """Check for Deepseek's latest version using GitHub API."""
    
    # GitHub API base URL
    GITHUB_API_BASE = "https://api.github.com/"
    
    # Default repository to check (DeepSeek-Coder)
    DEFAULT_REPO = "deepseek-ai/deepseek-coder"
    
    def __init__(self, github_token: Optional[str] = None):
        """
        Initialize the version checker.
        
        Args:
            github_token: Optional GitHub token for higher rate limits
        """
        self.session = requests.Session()
        if github_token:
            self.session.headers.update({
                "Authorization": f"token {github_token}",
                "Accept": "application/vnd.github.v3+json"
            })
        else:
            self.session.headers.update({
                "Accept": "application/vnd.github.v3+json"
            })
    
    def fetch_latest_release(self, repo: str) -> Optional[Dict[str, Any]]:
        """
        Fetch the latest release from GitHub API.
        
        Args:
            repo: Repository in format "owner/repo"
            
        Returns:
            Dictionary containing release information or None if error
        """
        url = urljoin(self.GITHUB_API_BASE, f"repos/{repo}/releases/latest")
        
        try:
            logger.info(f"Fetching latest release from {repo}")
            response = self.session.get(url, timeout=10)
            response.raise_for_status()
            
            release_data = response.json()
            logger.info(f"Successfully fetched release: {release_data.get('tag_name')}")
            return release_data
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to fetch release from GitHub API: {e}")
            return None
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse GitHub API response: {e}")
            return None
    
    def parse_release_info(self, release_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse relevant information from GitHub release data.
        
        Args:
            release_data: GitHub API release response
            
        Returns:
            Dictionary with parsed release information
        """
        try:
            tag_name = release_data.get("tag_name", "")
            published_at = release_data.get("published_at")
            
            # Parse version from tag name
            version = self.extract_version_from_tag(tag_name)
            
            # Parse date
            release_date = None
            if published_at:
                try:
                    release_date = datetime.fromisoformat(published_at.replace('Z', '+00:00'))
                except ValueError:
                    logger.warning(f"Could not parse date: {published_at}")
            
            return {
                "tag_name": tag_name,
                "version": version,
                "release_date": release_date,
                "published_at": published_at,
                "prerelease": release_data.get("prerelease", False),
                "html_url": release_data.get("html_url", ""),
                "body": release_data.get("body", "")[:500]  # First 500 chars
            }
        except Exception as e:
            logger.error(f"Error parsing release info: {e}")
            return {}
    
    def extract_version_from_tag(self, tag: str) -> Optional[str]:
        """
        Extract version number from a tag string.
        
        Args:
            tag: Git tag string (e.g., "v1.2.3", "release-2024.01.15")
            
        Returns:
            Extracted version string or None if not found
        """
        if not tag:
            return None
        
        # Try various version patterns
        patterns = [
            r'v?(\d+\.\d+\.\d+(?:-[a-zA-Z0-9]+)?)',  # v1.2.3 or 1.2.3-beta
            r'v?(\d+\.\d+)',  # v1.2 or 1.2
            r'(\d{4}\.\d{2}\.\d{2})',  # 2024.01.15
            r'(\d{4}-\d{2}-\d{2})',  # 2024-01-15
        ]
        
        for pattern in patterns:
            match = re.search(pattern, tag)
            if match:
                return match.group(1)
        
        return tag  # Return the tag itself if no pattern matches
    
    def is_recent_release(self, release_date: datetime, days_threshold: int = 30) -> bool:
        """
        Determine if a release is recent based on the current date.
        
        Args:
            release_date: Release datetime object
            days_threshold: Number of days to consider as "recent"
            
        Returns:
            True if release is within threshold days, False otherwise
        """
        if not release_date:
            return False
        
        current_date = datetime.now(timezone.utc)
        
        # Ensure both datetimes are timezone-aware
        if release_date.tzinfo is None:
            release_date = release_date.replace(tzinfo=timezone.utc)
        
        # Calculate difference in days
        delta = current_date - release_date
        is_recent = delta.days <= days_threshold
        
        logger.info(f"Release date: {release_date.date()}, "
                   f"Current date: {current_date.date()}, "
                   f"Age: {delta.days} days, "
                   f"Recent (<={days_threshold} days): {is_recent}")
        
        return is_recent
    
    def check_version(self, repo: str = DEFAULT_REPO, days_threshold: int = 30) -> Dict[str, Any]:
        """
        Main method to check version and determine if it's recent.
        
        Args:
            repo: GitHub repository to check
            days_threshold: Days threshold for considering a release as recent
            
        Returns:
            Dictionary with check results
        """
        result = {
            "repository": repo,
            "success": False,
            "latest_release": None,
            "is_recent": False,
            "error": None
        }
        
        # Fetch latest release from GitHub
        release_data = self.fetch_latest_release(repo)
        if not release_data:
            result["error"] = "Failed to fetch release data from GitHub API"
            return result
        
        # Parse release information
        release_info = self.parse_release_info(release_data)
        if not release_info:
            result["error"] = "Failed to parse release information"
            return result
        
        # Check if release is recent
        is_recent = False
        if release_info["release_date"]:
            is_recent = self.is_recent_release(
                release_info["release_date"], 
                days_threshold
            )
        
        result.update({
            "success": True,
            "latest_release": release_info,
            "is_recent": is_recent,
            "release_age_days": (
                (datetime.now(timezone.utc) - release_info["release_date"]).days
                if release_info["release_date"] else None
            ),
            "days_threshold": days_threshold
        })
        
        return result
    
    def print_results(self, results: Dict[str, Any]) -> None:
        """Print formatted results to console."""
        print("\n" + "="*60)
        print("DEEPSEEK VERSION CHECK RESULTS")
        print("="*60)
        
        if not results["success"]:
            print(f"❌ Error: {results['error']}")
            return
        
        release = results["latest_release"]
        
        print(f"Repository:     {results['repository']}")
        print(f"Latest Tag:     {release['tag_name']}")
        print(f"Version:        {release['version'] or 'N/A'}")
        print(f"Release Date:   {release['release_date'] or 'N/A'}")
        print(f"Pre-release:    {release['prerelease']}")
        
        if results.get('release_age_days') is not None:
            print(f"Age:            {results['release_age_days']} days")
            recent_status = "✅ RECENT" if results["is_recent"] else "⚠️  NOT RECENT"
            print(f"Status:         {recent_status} (threshold: {results['days_threshold']} days)")
        
        if release.get('html_url'):
            print(f"Release URL:    {release['html_url']}")
        
        print("="*60 + "\n")


def main():
    """Main entry point for the script."""
    parser = argparse.ArgumentParser(
        description="Check Deepseek's latest version using GitHub API"
    )
    parser.add_argument(
        "--repo",
        default="deepseek-ai/deepseek-coder",
        help="GitHub repository in format owner/repo (default: deepseek-ai/deepseek-coder)"
    )
    parser.add_argument(
        "--days",
        type=int,
        default=30,
        help="Number of days threshold for considering a release as recent (default: 30)"
    )
    parser.add_argument(
        "--token",
        help="GitHub token for higher rate limits (optional)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output results as JSON instead of human-readable format"
    )
    
    args = parser.parse_args()
    
    # Initialize checker
    checker = DeepseekVersionChecker(github_token=args.token)
    
    # Run version check
    results = checker.check_version(repo=args.repo, days_threshold=args.days)
    
    # Output results
    if args.json:
        print(json.dumps(results, default=str, indent=2))
    else:
        checker.print_results(results)
    
    # Exit with appropriate code
    if results.get("success") and results.get("is_recent"):
        sys.exit(0)  # Success, recent release
    elif results.get("success") and not results.get("is_recent"):
        sys.exit(1)  # Success, but not recent
    else:
        sys.exit(2)  # Failure


if __name__ == "__main__":
    main()