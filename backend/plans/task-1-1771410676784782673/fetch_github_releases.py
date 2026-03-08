#!/usr/bin/env python3
"""
GitHub Release Information Fetcher

This script fetches release information from the GitHub API for a given repository.
It supports fetching latest releases, specific releases by tag, or all releases.
"""

import argparse
import json
import sys
from typing import Any, Dict, List, Optional
import urllib.error
import urllib.parse
import urllib.request


class GitHubReleaseFetcher:
    """Fetches release information from GitHub API."""
    
    BASE_URL = "https://api.github.com/repos"
    
    def __init__(self, owner: str, repo: str, token: Optional[str] = None):
        """
        Initialize the fetcher with repository details.
        
        Args:
            owner: Repository owner/organization
            repo: Repository name
            token: Optional GitHub personal access token for higher rate limits
        """
        self.owner = owner
        self.repo = repo
        self.token = token
        
    def _make_request(self, url: str) -> Dict[str, Any]:
        """
        Make HTTP request to GitHub API with proper headers and error handling.
        
        Args:
            url: Full URL to request
            
        Returns:
            JSON response as dictionary
            
        Raises:
            urllib.error.HTTPError: For HTTP errors
            ValueError: For JSON parsing errors
        """
        headers = {
            "Accept": "application/vnd.github.v3+json",
            "User-Agent": "GitHub-Release-Fetcher/1.0"
        }
        
        if self.token:
            headers["Authorization"] = f"token {self.token}"
        
        request = urllib.request.Request(url, headers=headers)
        
        try:
            with urllib.request.urlopen(request) as response:
                data = response.read().decode('utf-8')
                return json.loads(data)
        except urllib.error.HTTPError as e:
            if e.code == 404:
                raise ValueError(f"Repository {self.owner}/{self.repo} not found or no releases exist")
            elif e.code == 403:
                raise ValueError("Rate limit exceeded. Consider using a GitHub token")
            else:
                raise ValueError(f"HTTP Error {e.code}: {e.reason}")
        except json.JSONDecodeError as e:
            raise ValueError(f"Failed to parse JSON response: {e}")
        except urllib.error.URLError as e:
            raise ValueError(f"Network error: {e.reason}")
    
    def get_latest_release(self) -> Dict[str, Any]:
        """Get the latest release for the repository."""
        url = f"{self.BASE_URL}/{self.owner}/{self.repo}/releases/latest"
        return self._make_request(url)
    
    def get_release_by_tag(self, tag: str) -> Dict[str, Any]:
        """Get a specific release by tag name."""
        url = f"{self.BASE_URL}/{self.owner}/{self.repo}/releases/tags/{urllib.parse.quote(tag)}"
        return self._make_request(url)
    
    def get_all_releases(self) -> List[Dict[str, Any]]:
        """Get all releases for the repository (paginated)."""
        url = f"{self.BASE_URL}/{self.owner}/{self.repo}/releases"
        return self._make_request(url)
    
    def get_releases(self, latest_only: bool = False, tag: Optional[str] = None) -> Any:
        """
        Get releases based on specified criteria.
        
        Args:
            latest_only: If True, fetch only the latest release
            tag: If provided, fetch release with this specific tag
            
        Returns:
            Release data (single dict for specific releases, list for all)
        """
        if tag:
            return self.get_release_by_tag(tag)
        elif latest_only:
            return self.get_latest_release()
        else:
            return self.get_all_releases()


def format_release(release: Dict[str, Any], detailed: bool = True) -> str:
    """
    Format release data for human-readable output.
    
    Args:
        release: Release data dictionary
        detailed: If True, include body and assets information
        
    Returns:
        Formatted string representation
    """
    output = [
        f"Release: {release.get('name', 'Unnamed')}",
        f"Tag: {release.get('tag_name', 'No tag')}",
        f"Published: {release.get('published_at', 'Not published')}",
        f"Pre-release: {release.get('prerelease', False)}",
        f"Draft: {release.get('draft', False)}",
        f"Author: {release.get('author', {}).get('login', 'Unknown') if release.get('author') else 'Unknown'}",
        f"URL: {release.get('html_url', 'No URL')}",
    ]
    
    if detailed:
        body = release.get('body', '').strip()
        if body:
            output.append(f"\nDescription:\n{body}")
        
        assets = release.get('assets', [])
        if assets:
            output.append(f"\nAssets ({len(assets)}):")
            for asset in assets:
                output.append(f"  - {asset.get('name', 'Unnamed')}: {asset.get('download_count', 0)} downloads")
                output.append(f"    Size: {asset.get('size', 0):,} bytes")
                output.append(f"    URL: {asset.get('browser_download_url', 'No URL')}")
    
    return "\n".join(output)


def format_all_releases(releases: List[Dict[str, Any]]) -> str:
    """Format list of releases for summary output."""
    if not releases:
        return "No releases found"
    
    output = [f"Found {len(releases)} release(s):\n"]
    
    for i, release in enumerate(releases, 1):
        output.append(f"{i}. {release.get('tag_name', 'No tag')} - {release.get('name', 'Unnamed')}")
        output.append(f"   Published: {release.get('published_at', 'Not published')}")
        output.append(f"   Pre-release: {release.get('prerelease', False)}")
        output.append(f"   Downloads: {sum(asset.get('download_count', 0) for asset in release.get('assets', []))}")
        output.append("")
    
    return "\n".join(output)


def main() -> None:
    """Main entry point for the script."""
    parser = argparse.ArgumentParser(
        description="Fetch release information from GitHub API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --owner microsoft --repo vscode --latest
  %(prog)s --owner python --repo cpython --tag v3.11.0
  %(prog)s --owner tensorflow --repo tensorflow --all
  %(prog)s --owner octocat --repo hello-world --token YOUR_GITHUB_TOKEN
        """
    )
    
    parser.add_argument(
        "--owner", "-o",
        required=True,
        help="Repository owner (user or organization)"
    )
    
    parser.add_argument(
        "--repo", "-r",
        required=True,
        help="Repository name"
    )
    
    parser.add_argument(
        "--latest", "-l",
        action="store_true",
        help="Fetch only the latest release"
    )
    
    parser.add_argument(
        "--all", "-a",
        action="store_true",
        help="Fetch all releases (default behavior)"
    )
    
    parser.add_argument(
        "--tag", "-t",
        help="Fetch specific release by tag name"
    )
    
    parser.add_argument(
        "--token", "-k",
        help="GitHub personal access token for higher rate limits"
    )
    
    parser.add_argument(
        "--output", "-O",
        choices=["text", "json"],
        default="text",
        help="Output format (default: text)"
    )
    
    parser.add_argument(
        "--detailed", "-d",
        action="store_true",
        help="Show detailed information including release body and assets"
    )
    
    args = parser.parse_args()
    
    # Validate mutually exclusive options
    if sum([args.latest, args.tag is not None, args.all]) > 1:
        parser.error("--latest, --tag, and --all are mutually exclusive")
    
    try:
        # Create fetcher instance
        fetcher = GitHubReleaseFetcher(args.owner, args.repo, args.token)
        
        # Determine which release(s) to fetch
        if args.latest:
            release_data = fetcher.get_latest_release()
            is_single = True
        elif args.tag:
            release_data = fetcher.get_release_by_tag(args.tag)
            is_single = True
        else:
            release_data = fetcher.get_all_releases()
            is_single = False
        
        # Handle output format
        if args.output == "json":
            if is_single:
                print(json.dumps(release_data, indent=2))
            else:
                print(json.dumps(release_data, indent=2))
        else:
            if is_single:
                print(format_release(release_data, args.detailed))
            else:
                print(format_all_releases(release_data))
        
        sys.exit(0)
        
    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nOperation cancelled by user", file=sys.stderr)
        sys.exit(130)
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()