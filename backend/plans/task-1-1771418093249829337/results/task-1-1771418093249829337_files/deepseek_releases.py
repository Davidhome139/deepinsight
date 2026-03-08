#!/usr/bin/env python3
"""
GitHub Releases Checker for Deepseek Repositories

This script checks the GitHub API for Deepseek repository releases,
providing detailed information about available releases with proper
error handling and rate limiting considerations.
"""

import json
import logging
import sys
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict, field
from enum import Enum
from urllib.parse import urljoin
import requests
from requests.exceptions import RequestException, Timeout, HTTPError


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('deepseek_releases.log'),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)


class ReleaseType(str, Enum):
    """Types of GitHub releases"""
    RELEASE = "release"
    PRERELEASE = "prerelease"
    DRAFT = "draft"
    ALL = "all"


class AssetType(str, Enum):
    """Types of release assets"""
    SOURCE_CODE = "source"
    BINARY = "binary"
    INSTALLER = "installer"
    DOCUMENTATION = "documentation"
    OTHER = "other"


@dataclass
class GitHubAsset:
    """Data class to structure GitHub release asset information"""
    name: str
    download_url: str
    size: int
    download_count: int
    content_type: str
    asset_type: AssetType = AssetType.OTHER
    created_at: datetime = field(default_factory=datetime.now)
    updated_at: datetime = field(default_factory=datetime.now)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "name": self.name,
            "download_url": self.download_url,
            "size": self.size,
            "download_count": self.download_count,
            "content_type": self.content_type,
            "asset_type": self.asset_type.value,
            "created_at": self.created_at.isoformat(),
            "updated_at": self.updated_at.isoformat()
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'GitHubAsset':
        """Create instance from dictionary"""
        return cls(
            name=data.get("name", ""),
            download_url=data.get("browser_download_url", ""),
            size=data.get("size", 0),
            download_count=data.get("download_count", 0),
            content_type=data.get("content_type", ""),
            created_at=datetime.fromisoformat(data.get("created_at", datetime.now().isoformat())),
            updated_at=datetime.fromisoformat(data.get("updated_at", datetime.now().isoformat()))
        )


@dataclass
class GitHubRelease:
    """Data class to structure GitHub release information"""
    tag_name: str
    name: str
    draft: bool
    prerelease: bool
    published_at: datetime
    html_url: str
    body: str
    assets: List[GitHubAsset] = field(default_factory=list)
    author: str = ""
    target_commitish: str = "main"
    created_at: datetime = field(default_factory=datetime.now)
    tarball_url: str = ""
    zipball_url: str = ""
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "tag_name": self.tag_name,
            "name": self.name,
            "draft": self.draft,
            "prerelease": self.prerelease,
            "published_at": self.published_at.isoformat(),
            "html_url": self.html_url,
            "body": self.body[:200] + "..." if len(self.body) > 200 else self.body,
            "assets": [asset.to_dict() for asset in self.assets],
            "author": self.author,
            "target_commitish": self.target_commitish,
            "created_at": self.created_at.isoformat(),
            "tarball_url": self.tarball_url,
            "zipball_url": self.zipball_url
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'GitHubRelease':
        """Create instance from dictionary"""
        assets = [GitHubAsset.from_dict(asset) for asset in data.get("assets", [])]
        
        # Determine asset types based on filename patterns
        for asset in assets:
            if asset.name.endswith(('.tar.gz', '.zip', '.rar')):
                asset.asset_type = AssetType.SOURCE_CODE
            elif asset.name.endswith(('.exe', '.dmg', '.app', '.bin')):
                asset.asset_type = AssetType.BINARY
            elif asset.name.endswith(('.msi', '.pkg', '.deb', '.rpm')):
                asset.asset_type = AssetType.INSTALLER
            elif asset.name.endswith(('.pdf', '.md', '.txt', '.html')):
                asset.asset_type = AssetType.DOCUMENTATION
        
        return cls(
            tag_name=data.get("tag_name", ""),
            name=data.get("name", data.get("tag_name", "")),
            draft=data.get("draft", False),
            prerelease=data.get("prerelease", False),
            published_at=datetime.fromisoformat(data.get("published_at", datetime.now().isoformat()).replace('Z', '+00:00')),
            html_url=data.get("html_url", ""),
            body=data.get("body", ""),
            assets=assets,
            author=data.get("author", {}).get("login", "") if isinstance(data.get("author"), dict) else "",
            target_commitish=data.get("target_commitish", "main"),
            created_at=datetime.fromisoformat(data.get("created_at", datetime.now().isoformat()).replace('Z', '+00:00')),
            tarball_url=data.get("tarball_url", ""),
            zipball_url=data.get("zipball_url", "")
        )


class GitHubReleasesChecker:
    """Main class for checking GitHub releases"""
    
    def __init__(self, 
                 repo_owner: str = "deepseek-ai", 
                 repo_name: str = "DeepSeek-Coder",
                 github_token: Optional[str] = None):
        """
        Initialize the GitHub Releases Checker
        
        Args:
            repo_owner: Repository owner/organization
            repo_name: Repository name
            github_token: Optional GitHub API token for higher rate limits
        """
        self.repo_owner = repo_owner
        self.repo_name = repo_name
        self.github_token = github_token
        self.base_url = "https://api.github.com"
        self.repo_url = f"/repos/{repo_owner}/{repo_name}"
        self.session = requests.Session()
        
        # Set up headers
        self.headers = {
            "Accept": "application/vnd.github.v3+json",
            "User-Agent": "DeepSeek-Releases-Checker/1.0"
        }
        
        if github_token:
            self.headers["Authorization"] = f"token {github_token}"
        
        self.session.headers.update(self.headers)
    
    def _handle_rate_limit(self, response: requests.Response) -> None:
        """Handle GitHub API rate limiting"""
        if response.status_code == 403 and 'X-RateLimit-Remaining' in response.headers:
            remaining = int(response.headers.get('X-RateLimit-Remaining', 0))
            reset_time = int(response.headers.get('X-RateLimit-Reset', 0))
            
            if remaining == 0:
                reset_datetime = datetime.fromtimestamp(reset_time)
                wait_seconds = max(1, (reset_datetime - datetime.now()).total_seconds())
                
                logger.warning(f"Rate limit exceeded. Waiting {wait_seconds:.0f} seconds until {reset_datetime}")
                time.sleep(wait_seconds)
    
    def _make_request(self, endpoint: str, params: Optional[Dict] = None) -> Optional[Dict]:
        """
        Make a request to GitHub API with error handling
        
        Args:
            endpoint: API endpoint
            params: Query parameters
            
        Returns:
            Response data as dictionary or None if error
        """
        url = urljoin(self.base_url, endpoint)
        
        try:
            response = self.session.get(url, params=params, timeout=30)
            
            # Handle rate limiting
            self._handle_rate_limit(response)
            
            response.raise_for_status()
            
            return response.json()
            
        except Timeout:
            logger.error(f"Request to {url} timed out")
            return None
        except HTTPError as e:
            logger.error(f"HTTP error for {url}: {e}")
            
            # Try to get error details from response
            if e.response is not None:
                try:
                    error_data = e.response.json()
                    logger.error(f"GitHub API error: {error_data.get('message', 'Unknown error')}")
                except json.JSONDecodeError:
                    logger.error(f"Response text: {e.response.text}")
            
            return None
        except RequestException as e:
            logger.error(f"Request failed for {url}: {e}")
            return None
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse JSON response from {url}: {e}")
            return None
    
    def get_releases(self, 
                    release_type: ReleaseType = ReleaseType.ALL,
                    per_page: int = 30,
                    page: int = 1) -> List[GitHubRelease]:
        """
        Get releases from the repository
        
        Args:
            release_type: Type of releases to fetch
            per_page: Number of releases per page
            page: Page number
            
        Returns:
            List of GitHubRelease objects
        """
        endpoint = f"{self.repo_url}/releases"
        
        params = {
            "per_page": per_page,
            "page": page
        }
        
        logger.info(f"Fetching releases from {self.repo_owner}/{self.repo_name}...")
        
        releases_data = self._make_request(endpoint, params)
        
        if not releases_data:
            logger.warning(f"No releases found or error occurred for {self.repo_owner}/{self.repo_name}")
            return []
        
        releases = []
        
        for release in releases_data:
            # Filter by release type
            if release_type == ReleaseType.ALL:
                pass
            elif release_type == ReleaseType.DRAFT and not release.get("draft"):
                continue
            elif release_type == ReleaseType.PRERELEASE and not release.get("prerelease"):
                continue
            elif release_type == ReleaseType.RELEASE and (release.get("draft") or release.get("prerelease")):
                continue
            
            github_release = GitHubRelease.from_dict(release)
            releases.append(github_release)
        
        logger.info(f"Found {len(releases)} releases")
        return releases
    
    def get_latest_release(self) -> Optional[GitHubRelease]:
        """
        Get the latest release from the repository
        
        Returns:
            Latest GitHubRelease or None if not found
        """
        endpoint = f"{self.repo_url}/releases/latest"
        
        logger.info(f"Fetching latest release from {self.repo_owner}/{self.repo_name}...")
        
        release_data = self._make_request(endpoint)
        
        if not release_data:
            logger.warning(f"No latest release found for {self.repo_owner}/{self.repo_name}")
            return None
        
        return GitHubRelease.from_dict(release_data)
    
    def get_release_by_tag(self, tag: str) -> Optional[GitHubRelease]:
        """
        Get a specific release by tag name
        
        Args:
            tag: Release tag name
            
        Returns:
            GitHubRelease or None if not found
        """
        endpoint = f"{self.repo_url}/releases/tags/{tag}"
        
        logger.info(f"Fetching release with tag '{tag}' from {self.repo_owner}/{self.repo_name}...")
        
        release_data = self._make_request(endpoint)
        
        if not release_data:
            logger.warning(f"Release with tag '{tag}' not found for {self.repo_owner}/{self.repo_name}")
            return None
        
        return GitHubRelease.from_dict(release_data)
    
    def get_repository_info(self) -> Optional[Dict]:
        """
        Get basic repository information
        
        Returns:
            Repository information dictionary or None
        """
        logger.info(f"Fetching repository info for {self.repo_owner}/{self.repo_name}...")
        
        return self._make_request(self.repo_url)
    
    def check_release_activity(self, days: int = 30) -> Dict[str, Any]:
        """
        Check release activity for the past N days
        
        Args:
            days: Number of days to look back
            
        Returns:
            Dictionary with activity statistics
        """
        cutoff_date = datetime.now() - timedelta(days=days)
        
        all_releases = self.get_releases()
        
        recent_releases = []
        for release in all_releases:
            if release.published_at >= cutoff_date and not release.draft:
                recent_releases.append(release)
        
        total_downloads = 0
        for release in recent_releases:
            for asset in release.assets:
                total_downloads += asset.download_count
        
        return {
            "total_releases": len(all_releases),
            "recent_releases": len(recent_releases),
            "days_analyzed": days,
            "cutoff_date": cutoff_date.isoformat(),
            "total_downloads_recent": total_downloads,
            "most_recent_release": recent_releases[0].tag_name if recent_releases else None
        }
    
    def print_release_summary(self, release: GitHubRelease) -> None:
        """Print a formatted summary of a release"""
        print(f"\n{'='*60}")
        print(f"Release: {release.name} ({release.tag_name})")
        print(f"{'='*60}")
        print(f"Published: {release.published_at.strftime('%Y-%m-%d %H:%M:%S')}")
        print(f"Type: {'Pre-release' if release.prerelease else 'Stable release'}")
        print(f"Draft: {'Yes' if release.draft else 'No'}")
        print(f"Author: {release.author}")
        print(f"Target: {release.target_commitish}")
        print(f"\nDescription:")
        print(f"{'-'*40}")
        print(release.body[:500] + "..." if len(release.body) > 500 else release.body)
        
        if release.assets:
            print(f"\nAssets ({len(release.assets)}):")
            print(f"{'-'*40}")
            for asset in release.assets:
                size_mb = asset.size / (1024 * 1024)
                print(f"  • {asset.name}")
                print(f"    Type: {asset.asset_type.value}")
                print(f"    Size: {size_mb:.2f} MB")
                print(f"    Downloads: {asset.download_count}")
                print(f"    URL: {asset.download_url[:80]}..." if len(asset.download_url) > 80 else f"    URL: {asset.download_url}")
                print()
        
        print(f"\nLinks:")
        print(f"  • GitHub: {release.html_url}")
        if release.tarball_url:
            print(f"  • Tarball: {release.tarball_url}")
        if release.zipball_url:
            print(f"  • Zipball: {release.zipball_url}")
        print()
    
    def save_releases_to_file(self, releases: List[GitHubRelease], filename: str = "releases.json") -> bool:
        """
        Save releases to a JSON file
        
        Args:
            releases: List of releases to save
            filename: Output filename
            
        Returns:
            True if successful, False otherwise
        """
        try:
            data = {
                "repository": f"{self.repo_owner}/{self.repo_name}",
                "fetched_at": datetime.now().isoformat(),
                "total_releases": len(releases),
                "releases": [release.to_dict() for release in releases]
            }
            
            with open(filename, 'w', encoding='utf-8') as f:
                json.dump(data, f, indent=2, ensure_ascii=False)
            
            logger.info(f"Saved {len(releases)} releases to {filename}")
            return True
            
        except (IOError, OSError) as e:
            logger.error(f"Failed to save releases to {filename}: {e}")
            return False


def main():
    """Main entry point for the script"""
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Check GitHub API for Deepseek repository releases",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s
  %(prog)s --repo deepseek-ai/DeepSeek-V2
  %(prog)s --latest
  %(prog)s --tag v1.0.0
  %(prog)s --all --output releases.json
  %(prog)s --activity 90
        """
    )
    
    parser.add_argument(
        "--repo", 
        default="deepseek-ai/DeepSeek-Coder",
        help="Repository in format 'owner/name' (default: deepseek-ai/DeepSeek-Coder)"
    )
    
    parser.add_argument(
        "--token",
        help="GitHub personal access token for higher rate limits (optional)"
    )
    
    parser.add_argument(
        "--latest", 
        action="store_true",
        help="Get only the latest release"
    )
    
    parser.add_argument(
        "--tag",
        help="Get a specific release by tag name"
    )
    
    parser.add_argument(
        "--all", 
        action="store_true",
        help="Get all releases"
    )
    
    parser.add_argument(
        "--activity",
        type=int,
        metavar="DAYS",
        help="Check release activity for the past N days"
    )
    
    parser.add_argument(
        "--type",
        choices=[rt.value for rt in ReleaseType],
        default="all",
        help="Type of releases to fetch (default: all)"
    )
    
    parser.add_argument(
        "--output",
        help="Save releases to JSON file"
    )
    
    parser.add_argument(
        "--limit",
        type=int,
        default=10,
        help="Limit number of releases to fetch (default: 10)"
    )
    
    args = parser.parse_args()
    
    # Parse repository owner and name
    if '/' in args.repo:
        repo_owner, repo_name = args.repo.split('/', 1)
    else:
        repo_owner = "deepseek-ai"
        repo_name = args.repo
    
    try:
        # Initialize checker
        checker = GitHubReleasesChecker(
            repo_owner=repo_owner,
            repo_name=repo_name,
            github_token=args.token
        )
        
        # Check repository exists
        repo_info = checker.get_repository_info()
        if not repo_info:
            logger.error(f"Repository {repo_owner}/{repo_name} not found or inaccessible")
            return 1
        
        print(f"\nRepository: {repo_info.get('full_name')}")
        print(f"Description: {repo_info.get('description', 'No description')}")
        print(f"Stars: {repo_info.get('stargazers_count', 0):,}")
        print(f"Forks: {repo_info.get('forks_count', 0):,}")
        print(f"Watchers: {repo_info.get('watchers_count', 0):,}")
        print(f"Open Issues: {repo_info.get('open_issues_count', 0):,}")
        
        # Perform requested action
        if args.tag:
            release = checker.get_release_by_tag(args.tag)
            if release:
                checker.print_release_summary(release)
                if args.output:
                    checker.save_releases_to_file([release], args.output)
            else:
                logger.error(f"Release with tag '{args.tag}' not found")
                return 1
                
        elif args.latest:
            release = checker.get_latest_release()
            if release:
                checker.print_release_summary(release)
                if args.output:
                    checker.save_releases_to_file([release], args.output)
            else:
                logger.error("No releases found")
                return 1
                
        elif args.activity:
            activity = checker.check_release_activity(days=args.activity)
            print(f"\nRelease Activity (last {args.activity} days):")
            print(f"{'='*40}")
            for key, value in activity.items():
                if key not in ['cutoff_date']:
                    print(f"{key.replace('_', ' ').title()}: {value}")
                    
        else:
            release_type = ReleaseType(args.type)
            releases = checker.get_releases(
                release_type=release_type,
                per_page=args.limit
            )
            
            if releases:
                print(f"\nFound {len(releases)} releases:")
                
                for i, release in enumerate(releases, 1):
                    print(f"\n{i}. {release.tag_name}: {release.name}")
                    print(f"   Published: {release.published_at.strftime('%Y-%m-%d')}")
                    print(f"   Type: {'Pre-release' if release.prerelease else 'Stable'}")
                    print(f"   Assets: {len(release.assets)}")
                    
                    total_downloads = sum(asset.download_count for asset in release.assets)
                    if total_downloads > 0:
                        print(f"   Total Downloads: {total_downloads:,}")
                
                if args.output:
                    checker.save_releases_to_file(releases, args.output)
                    
                # Print detailed summary of the first release
                if len(releases) > 0:
                    print("\n" + "="*60)
                    print("DETAILED VIEW OF LATEST RELEASE:")
                    checker.print_release_summary(releases[0])
            else:
                logger.warning("No releases found")
                return 1
        
        return 0
        
    except KeyboardInterrupt:
        logger.info("Operation cancelled by user")
        return 130
    except Exception as e:
        logger.error(f"Unexpected error: {e}", exc_info=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())