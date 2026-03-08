#!/usr/bin/env python3
"""
GitHub Release Fetcher and Parser

This script fetches and parses release information from GitHub API and other sources.
It supports multiple output formats and provides rich filtering options.
"""

import argparse
import json
import sys
import time
from dataclasses import dataclass, asdict
from datetime import datetime
from enum import Enum
from pathlib import Path
from typing import Any, Dict, List, Optional, Union
from urllib.parse import urlparse

import requests
from requests.exceptions import RequestException, HTTPError, Timeout


class OutputFormat(Enum):
    """Supported output formats for release data."""
    JSON = "json"
    TABLE = "table"
    MARKDOWN = "markdown"
    CSV = "csv"
    YAML = "yaml"


@dataclass
class ReleaseAsset:
    """Represents a single asset attached to a GitHub release."""
    name: str
    download_url: str
    size: int
    download_count: int
    content_type: str

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ReleaseAsset":
        """Create ReleaseAsset from GitHub API response."""
        return cls(
            name=data.get("name", ""),
            download_url=data.get("browser_download_url", ""),
            size=data.get("size", 0),
            download_count=data.get("download_count", 0),
            content_type=data.get("content_type", "application/octet-stream")
        )


@dataclass
class Release:
    """Represents a GitHub release with all relevant information."""
    tag_name: str
    name: str
    body: str
    published_at: datetime
    prerelease: bool
    draft: bool
    author: str
    html_url: str
    assets: List[ReleaseAsset]
    tarball_url: str
    zipball_url: str

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Release":
        """Create Release from GitHub API response."""
        published_str = data.get("published_at")
        published_at = datetime.fromisoformat(published_str.replace("Z", "+00:00")) if published_str else datetime.now()
        
        author_data = data.get("author", {})
        author_name = author_data.get("login", "Unknown") if author_data else "Unknown"
        
        assets = [ReleaseAsset.from_dict(asset) for asset in data.get("assets", [])]
        
        return cls(
            tag_name=data.get("tag_name", ""),
            name=data.get("name", "") or data.get("tag_name", ""),
            body=data.get("body", ""),
            published_at=published_at,
            prerelease=data.get("prerelease", False),
            draft=data.get("draft", False),
            author=author_name,
            html_url=data.get("html_url", ""),
            assets=assets,
            tarball_url=data.get("tarball_url", ""),
            zipball_url=data.get("zipball_url", "")
        )


class GitHubAPIError(Exception):
    """Custom exception for GitHub API errors."""
    pass


class ReleaseFetcher:
    """Handles fetching and parsing release information from GitHub."""
    
    BASE_URL = "https://api.github.com"
    DEFAULT_TIMEOUT = 30
    RATE_LIMIT_HEADERS = ["X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"]
    
    def __init__(self, token: Optional[str] = None, user_agent: str = "GitHub-Release-Fetcher"):
        """
        Initialize the ReleaseFetcher.
        
        Args:
            token: GitHub personal access token (optional, increases rate limits)
            user_agent: User-Agent string for API requests
        """
        self.session = requests.Session()
        self.session.headers.update({
            "Accept": "application/vnd.github.v3+json",
            "User-Agent": user_agent
        })
        
        if token:
            self.session.headers["Authorization"] = f"token {token}"
    
    def _make_request(self, url: str, params: Optional[Dict] = None) -> Dict[str, Any]:
        """
        Make an HTTP request to the GitHub API with proper error handling.
        
        Args:
            url: The URL to request
            params: Query parameters
            
        Returns:
            JSON response as dictionary
            
        Raises:
            GitHubAPIError: If the request fails or returns an error
        """
        try:
            response = self.session.get(
                url,
                params=params,
                timeout=self.DEFAULT_TIMEOUT
            )
            response.raise_for_status()
            
            # Log rate limit information
            self._log_rate_limit_info(response)
            
            return response.json()
            
        except HTTPError as e:
            if e.response.status_code == 403:
                # Check if it's a rate limit error
                if "rate limit" in e.response.text.lower():
                    reset_time = int(e.response.headers.get("X-RateLimit-Reset", 0))
                    wait_time = max(0, reset_time - int(time.time()))
                    raise GitHubAPIError(
                        f"Rate limit exceeded. Reset in {wait_time} seconds. "
                        f"Consider using a GitHub token for higher limits."
                    )
            raise GitHubAPIError(f"GitHub API error: {e}")
        except Timeout:
            raise GitHubAPIError("Request timed out")
        except RequestException as e:
            raise GitHubAPIError(f"Network error: {e}")
        except json.JSONDecodeError as e:
            raise GitHubAPIError(f"Failed to parse JSON response: {e}")
    
    def _log_rate_limit_info(self, response: requests.Response) -> None:
        """Log rate limit information from response headers."""
        rate_info = []
        for header in self.RATE_LIMIT_HEADERS:
            if header in response.headers:
                rate_info.append(f"{header}: {response.headers[header]}")
        
        if rate_info:
            print(f"Rate limit info: {', '.join(rate_info)}", file=sys.stderr)
    
    def parse_repo_url(self, repo_identifier: str) -> tuple[str, str]:
        """
        Parse repository identifier into owner and repo name.
        
        Args:
            repo_identifier: Repository in format "owner/repo" or full GitHub URL
            
        Returns:
            Tuple of (owner, repo_name)
        """
        # Check if it's a URL
        if repo_identifier.startswith(("http://", "https://")):
            parsed = urlparse(repo_identifier)
            path_parts = parsed.path.strip("/").split("/")
            if len(path_parts) >= 2:
                return path_parts[0], path_parts[1]
        
        # Assume it's in owner/repo format
        parts = repo_identifier.split("/")
        if len(parts) != 2:
            raise ValueError(
                f"Invalid repository identifier: {repo_identifier}. "
                f"Expected format: 'owner/repo' or full GitHub URL"
            )
        
        return parts[0], parts[1]
    
    def fetch_releases(
        self,
        repo_identifier: str,
        per_page: int = 30,
        page: int = 1,
        include_drafts: bool = False,
        include_prereleases: bool = True
    ) -> List[Release]:
        """
        Fetch releases from a GitHub repository.
        
        Args:
            repo_identifier: Repository in format "owner/repo" or full GitHub URL
            per_page: Number of releases per page (max 100)
            page: Page number to fetch
            include_drafts: Whether to include draft releases
            include_prereleases: Whether to include prereleases
            
        Returns:
            List of Release objects
            
        Raises:
            GitHubAPIError: If fetching fails
            ValueError: If repo_identifier is invalid
        """
        owner, repo = self.parse_repo_url(repo_identifier)
        url = f"{self.BASE_URL}/repos/{owner}/{repo}/releases"
        
        params = {
            "per_page": min(per_page, 100),  # GitHub max is 100
            "page": page
        }
        
        try:
            data = self._make_request(url, params)
            releases = [Release.from_dict(item) for item in data]
            
            # Apply filters
            filtered_releases = []
            for release in releases:
                if release.draft and not include_drafts:
                    continue
                if release.prerelease and not include_prereleases:
                    continue
                filtered_releases.append(release)
            
            return filtered_releases
            
        except GitHubAPIError:
            raise
        except Exception as e:
            raise GitHubAPIError(f"Unexpected error fetching releases: {e}")
    
    def fetch_latest_release(self, repo_identifier: str) -> Optional[Release]:
        """
        Fetch the latest release from a GitHub repository.
        
        Args:
            repo_identifier: Repository in format "owner/repo" or full GitHub URL
            
        Returns:
            Release object or None if no releases found
        """
        owner, repo = self.parse_repo_url(repo_identifier)
        url = f"{self.BASE_URL}/repos/{owner}/{repo}/releases/latest"
        
        try:
            data = self._make_request(url)
            return Release.from_dict(data)
        except HTTPError as e:
            if e.response.status_code == 404:
                return None
            raise GitHubAPIError(f"GitHub API error: {e}")
    
    def search_releases(
        self,
        query: str,
        per_page: int = 30,
        page: int = 1
    ) -> List[Dict[str, Any]]:
        """
        Search for releases across GitHub using the search API.
        
        Args:
            query: Search query (supports GitHub search syntax)
            per_page: Results per page
            page: Page number
            
        Returns:
            List of search results
            
        Note: This requires authentication for more than 60 requests/hour
        """
        url = f"{self.BASE_URL}/search/repositories"
        params = {
            "q": query,
            "per_page": per_page,
            "page": page,
            "sort": "updated",
            "order": "desc"
        }
        
        try:
            data = self._make_request(url, params)
            return data.get("items", [])
        except GitHubAPIError:
            # Fallback to basic search without auth if rate limited
            print("Warning: Search may be rate limited without authentication", file=sys.stderr)
            return []


class OutputFormatter:
    """Formats release data into different output formats."""
    
    @staticmethod
    def to_json(releases: List[Release]) -> str:
        """Convert releases to JSON string."""
        releases_dict = []
        for release in releases:
            release_dict = asdict(release)
            # Convert datetime to ISO string
            release_dict["published_at"] = release.published_at.isoformat()
            releases_dict.append(release_dict)
        
        return json.dumps(releases_dict, indent=2, ensure_ascii=False)
    
    @staticmethod
    def to_table(releases: List[Release]) -> str:
        """Convert releases to ASCII table."""
        if not releases:
            return "No releases found."
        
        headers = ["Tag", "Name", "Published", "Author", "Assets", "Prerelease", "Draft"]
        rows = []
        
        for release in releases:
            rows.append([
                release.tag_name,
                release.name[:40] + "..." if len(release.name) > 40 else release.name,
                release.published_at.strftime("%Y-%m-%d"),
                release.author,
                str(len(release.assets)),
                "✓" if release.prerelease else "",
                "✓" if release.draft else ""
            ])
        
        # Calculate column widths
        col_widths = [len(h) for h in headers]
        for row in rows:
            for i, cell in enumerate(row):
                col_widths[i] = max(col_widths[i], len(str(cell)))
        
        # Build table
        separator = "+" + "+".join(["-" * (w + 2) for w in col_widths]) + "+"
        header_row = "| " + " | ".join([h.ljust(w) for h, w in zip(headers, col_widths)]) + " |"
        
        lines = [separator, header_row, separator]
        for row in rows:
            line = "| " + " | ".join([str(cell).ljust(w) for cell, w in zip(row, col_widths)]) + " |"
            lines.append(line)
        lines.append(separator)
        
        return "\n".join(lines)
    
    @staticmethod
    def to_markdown(releases: List[Release]) -> str:
        """Convert releases to Markdown format."""
        if not releases:
            return "No releases found."
        
        lines = ["# Releases", ""]
        
        for release in releases:
            lines.append(f"## {release.name} ({release.tag_name})")
            lines.append("")
            lines.append(f"**Published:** {release.published_at.strftime('%Y-%m-%d %H:%M:%S')}")
            lines.append(f"**Author:** {release.author}")
            
            if release.prerelease:
                lines.append("**Prerelease:** Yes")
            if release.draft:
                lines.append("**Draft:** Yes")
            
            lines.append("")
            lines.append(f"**URL:** [{release.html_url}]({release.html_url})")
            lines.append("")
            
            if release.body:
                lines.append("### Release Notes")
                lines.append("")
                lines.append(release.body)
                lines.append("")
            
            if release.assets:
                lines.append("### Assets")
                lines.append("")
                lines.append("| File | Downloads | Size |")
                lines.append("|------|-----------|------|")
                for asset in release.assets:
                    size_mb = asset.size / (1024 * 1024)
                    lines.append(f"| [{asset.name}]({asset.download_url}) | {asset.download_count:,} | {size_mb:.1f} MB |")
                lines.append("")
            
            lines.append("---")
            lines.append("")
        
        return "\n".join(lines)
    
    @staticmethod
    def to_csv(releases: List[Release]) -> str:
        """Convert releases to CSV format."""
        if not releases:
            return ""
        
        headers = ["tag_name", "name", "published_at", "author", "prerelease", "draft", "html_url", "assets_count"]
        lines = [",".join(headers)]
        
        for release in releases:
            row = [
                release.tag_name,
                f'"{release.name.replace(\'"\', \'""\')}"',  # Quote and escape quotes
                release.published_at.isoformat(),
                release.author,
                str(release.prerelease),
                str(release.draft),
                release.html_url,
                str(len(release.assets))
            ]
            lines.append(",".join(row))
        
        return "\n".join(lines)
    
    @staticmethod
    def to_yaml(releases: List[Release]) -> str:
        """Convert releases to YAML format."""
        import yaml
        
        releases_dict = []
        for release in releases:
            release_dict = asdict(release)
            release_dict["published_at"] = release.published_at.isoformat()
            releases_dict.append(release_dict)
        
        return yaml.dump(releases_dict, default_flow_style=False, allow_unicode=True)
    
    @classmethod
    def format(
        cls,
        releases: List[Release],
        format_type: OutputFormat,
        output_file: Optional[Path] = None
    ) -> None:
        """
        Format releases and output to file or stdout.
        
        Args:
            releases: List of releases to format
            format_type: Output format
            output_file: Optional file to write output to
        """
        if format_type == OutputFormat.JSON:
            output = cls.to_json(releases)
        elif format_type == OutputFormat.TABLE:
            output = cls.to_table(releases)
        elif format_type == OutputFormat.MARKDOWN:
            output = cls.to_markdown(releases)
        elif format_type == OutputFormat.CSV:
            output = cls.to_csv(releases)
        elif format_type == OutputFormat.YAML:
            output = cls.to_yaml(releases)
        else:
            raise ValueError(f"Unsupported format: {format_type}")
        
        if output_file:
            try:
                output_file.write_text(output, encoding="utf-8")
                print(f"Output written to: {output_file}", file=sys.stderr)
            except IOError as e:
                raise IOError(f"Failed to write output file: {e}")
        else:
            print(output)


def main() -> None:
    """Main entry point for the script."""
    parser = argparse.ArgumentParser(
        description="Fetch and parse release information from GitHub repositories",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s microsoft/vscode
  %(prog)s https://github.com/python/cpython --format table
  %(prog)s tensorflow/tensorflow --latest --format json
  %(prog)s owner/repo --token $GITHUB_TOKEN --output releases.md --format markdown
  %(prog)s owner/repo --search "machine learning" --per-page 10
        """
    )
    
    # Repository argument (positional or via --repo)
    repo_group = parser.add_mutually_exclusive_group(required=True)
    repo_group.add_argument(
        "repository",
        nargs="?",
        help="GitHub repository in format 'owner/repo' or full URL"
    )
    repo_group.add_argument(
        "--repo",
        dest="repository_alt",
        help="GitHub repository in format 'owner/repo' or full URL (alternative)"
    )
    
    # Search option
    parser.add_argument(
        "--search",
        help="Search for repositories containing this term (instead of fetching from specific repo)"
    )
    
    # Output options
    parser.add_argument(
        "--format",
        choices=[fmt.value for fmt in OutputFormat],
        default=OutputFormat.TABLE.value,
        help="Output format (default: table)"
    )
    parser.add_argument(
        "--output", "-o",
        type=Path,
        help="Output file path (default: stdout)"
    )
    
    # Filtering options
    parser.add_argument(
        "--latest",
        action="store_true",
        help="Fetch only the latest release"
    )
    parser.add_argument(
        "--include-drafts",
        action="store_true",
        help="Include draft releases"
    )
    parser.add_argument(
        "--exclude-prereleases",
        action="store_true",
        help="Exclude prereleases"
    )
    
    # Pagination options
    parser.add_argument(
        "--per-page",
        type=int,
        default=30,
        help="Number of releases per page (max 100, default: 30)"
    )
    parser.add_argument(
        "--page",
        type=int,
        default=1,
        help="Page number (default: 1)"
    )
    
    # Authentication
    parser.add_argument(
        "--token",
        help="GitHub personal access token (increases rate limits)"
    )
    parser.add_argument(
        "--token-file",
        type=Path,
        help="File containing GitHub personal access token"
    )
    
    # Other options
    parser.add_argument(
        "--user-agent",
        default="GitHub-Release-Fetcher/1.0",
        help="User-Agent string for API requests"
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Enable verbose output"
    )
    
    args = parser.parse_args()
    
    # Use either positional or --repo argument
    repo_identifier = args.repository or args.repository_alt
    
    # Read token from file if specified
    token = args.token
    if args.token_file and not token:
        try:
            token = args.token_file.read_text().strip()
        except IOError as e:
            print(f"Warning: Could not read token file: {e}", file=sys.stderr)
    
    try:
        # Initialize fetcher
        fetcher = ReleaseFetcher(token=token, user_agent=args.user_agent)
        
        releases: List[Release] = []
        
        if args.search:
            # Search mode
            if args.verbose:
                print(f"Searching for repositories with: {args.search}", file=sys.stderr)
            
            results = fetcher.search_releases(
                query=args.search,
                per_page=args.per_page,
                page=args.page
            )
            
            # Convert search results to simple display format
            print(f"Found {len(results)} repositories:", file=sys.stderr)
            for result in results:
                print(f"  - {result['full_name']}: {result.get('description', 'No description')}", file=sys.stderr)
            
            if not results:
                print("No repositories found.", file=sys.stderr)
                sys.exit(0)
            
            # Ask user if they want to fetch releases from any of these
            if len(results) == 1:
                repo_identifier = results[0]["full_name"]
                if args.verbose:
                    print(f"Fetching releases from: {repo_identifier}", file=sys.stderr)
            else:
                # In a real implementation, you might add interactive selection here
                print("\nNote: Use --repo <owner/repo> to fetch releases from a specific repository", file=sys.stderr)
                sys.exit(0)
        
        # Fetch releases
        if args.latest:
            if args.verbose:
                print(f"Fetching latest release from: {repo_identifier}", file=sys.stderr)
            
            release = fetcher.fetch_latest_release(repo_identifier)
            if release:
                releases = [release]
            else:
                print(f"No releases found for {repo_identifier}", file=sys.stderr)
                sys.exit(1)
        else:
            if args.verbose:
                print(f"Fetching releases from: {repo_identifier}", file=sys.stderr)
            
            releases = fetcher.fetch_releases(
                repo_identifier=repo_identifier,
                per_page=args.per_page,
                page=args.page,
                include_drafts=args.include_drafts,
                include_prereleases=not args.exclude_prereleases
            )
        
        if not releases:
            print(f"No releases found for {repo_identifier}", file=sys.stderr)
            sys.exit(0)
        
        if args.verbose:
            print(f"Found {len(releases)} release(s)", file=sys.stderr)
        
        # Format and output
        formatter = OutputFormatter()
        format_type = OutputFormat(args.format)
        
        formatter.format(
            releases=releases,
            format_type=format_type,
            output_file=args.output
        )
        
    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except GitHubAPIError as e:
        print(f"GitHub API Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        if args.verbose:
            import traceback
            traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()