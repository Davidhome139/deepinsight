#!/usr/bin/env python3
"""
Web scraping module for checking Deepseek versions from official website and GitHub.
Uses MCP servers for enhanced functionality when available.
"""

import re
import json
import logging
import subprocess
import html
from typing import Optional, List, Dict, Any, Tuple
from dataclasses import dataclass
from datetime import datetime
from urllib.parse import urljoin, urlparse
import sys

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@dataclass
class ScrapedVersionInfo:
    """Data class to hold version information from web scraping."""
    version: str
    source: str
    url: str
    release_date: Optional[str] = None
    changelog: Optional[str] = None
    checksum: Optional[str] = None
    file_size: Optional[str] = None
    scraped_at: Optional[str] = None


class DeepseekWebScraper:
    """
    Web scraper for Deepseek official website and GitHub releases.
    Uses available MCP servers for enhanced functionality.
    """
    
    # URLs for scraping
    OFFICIAL_WEBSITE = "https://www.deepseek.com"
    GITHUB_RELEASES = "https://github.com/deepseek-ai/DeepSeek-V3/releases"
    GITHUB_API = "https://api.github.com/repos/deepseek-ai/DeepSeek-V3/releases"
    
    # Version patterns
    VERSION_PATTERNS = [
        r'v?(\d+\.\d+\.\d+)',  # Standard semver: 1.2.3 or v1.2.3
        r'v?(\d+\.\d+)',       # Major.minor: 1.2 or v1.2
        r'Release\s+(\d+\.\d+\.\d+)',  # Release 1.2.3
        r'DeepSeek[-\s]*v?(\d+\.\d+\.\d+)',  # DeepSeek v1.2.3
        r'version[:\s]*v?(\d+\.\d+\.\d+)',   # version: 1.2.3
    ]
    
    def __init__(self, use_mcp: bool = True):
        """Initialize the web scraper with optional MCP server usage."""
        self.use_mcp = use_mcp
        self.session = None
        self._init_mcp_clients()
        
    def _init_mcp_clients(self):
        """Initialize MCP client connections if available."""
        self.mcp_search = None
        self.mcp_terminal = None
        
        if not self.use_mcp:
            return
            
        try:
            # Try to import and initialize MCP clients
            # These would be provided by the environment
            import mcp
            
            # Attempt to connect to available MCP servers
            # In a real implementation, these would be configured properly
            logger.info("MCP servers available: attempting to use search and terminal")
            
        except ImportError:
            logger.warning("MCP client not available, using standard scraping methods")
            self.use_mcp = False
        except Exception as e:
            logger.warning(f"Failed to initialize MCP clients: {e}")
            self.use_mcp = False
    
    def _fetch_with_mcp(self, url: str, method: str = "search") -> Optional[str]:
        """
        Fetch web content using MCP servers if available.
        
        Args:
            url: URL to fetch
            method: Which MCP method to use ('search' or 'terminal' for curl)
            
        Returns:
            HTML content as string or None if failed
        """
        if not self.use_mcp:
            return None
            
        try:
            if method == "terminal" and self.mcp_terminal:
                # Use terminal MCP to run curl
                curl_command = f"curl -s -L -A 'Mozilla/5.0' '{url}'"
                # In real implementation: self.mcp_terminal.execute(curl_command)
                logger.debug(f"Would execute curl command: {curl_command}")
                
                # Fallback to subprocess for now
                return self._fetch_with_curl(url)
                
            elif method == "search" and self.mcp_search:
                # Use search MCP to fetch content
                # In real implementation: self.mcp_search.fetch(url)
                logger.debug(f"Would use MCP search to fetch: {url}")
                
        except Exception as e:
            logger.error(f"Error using MCP to fetch {url}: {e}")
            
        return None
    
    def _fetch_with_curl(self, url: str) -> Optional[str]:
        """
        Fetch URL content using curl command.
        
        Args:
            url: URL to fetch
            
        Returns:
            HTML content as string or None if failed
        """
        try:
            # Construct curl command with appropriate headers
            curl_cmd = [
                "curl", "-s", "-L", 
                "-A", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                "-H", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
                "-H", "Accept-Language: en-US,en;q=0.5",
                "-H", "Accept-Encoding: gzip, deflate, br",
                url
            ]
            
            logger.debug(f"Executing curl command for: {url}")
            result = subprocess.run(
                curl_cmd,
                capture_output=True,
                text=True,
                timeout=30
            )
            
            if result.returncode == 0:
                return result.stdout
            else:
                logger.error(f"curl failed with return code {result.returncode}: {result.stderr}")
                
        except subprocess.TimeoutExpired:
            logger.error(f"Timeout while fetching {url}")
        except Exception as e:
            logger.error(f"Error executing curl for {url}: {e}")
            
        return None
    
    def _fetch_url(self, url: str) -> Optional[str]:
        """
        Fetch URL content using best available method.
        
        Args:
            url: URL to fetch
            
        Returns:
            HTML content as string or None if failed
        """
        # Try MCP first if available
        content = self._fetch_with_mcp(url, method="terminal")
        if content:
            return content
            
        # Fallback to curl
        content = self._fetch_with_curl(url)
        if content:
            return content
            
        logger.warning(f"Failed to fetch URL using any method: {url}")
        return None
    
    def _extract_version_from_text(self, text: str) -> Optional[str]:
        """
        Extract version number from text using regex patterns.
        
        Args:
            text: Text to search for version
            
        Returns:
            Version string or None if not found
        """
        if not text:
            return None
            
        for pattern in self.VERSION_PATTERNS:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                version = match.group(1)
                # Validate it looks like a version
                if re.match(r'^\d+\.\d+(\.\d+)?$', version):
                    return version
                    
        return None
    
    def _parse_html_for_versions(self, html_content: str, base_url: str) -> List[ScrapedVersionInfo]:
        """
        Parse HTML content to find version information.
        
        Args:
            html_content: HTML to parse
            base_url: Base URL for resolving relative links
            
        Returns:
            List of ScrapedVersionInfo objects
        """
        versions = []
        
        if not html_content:
            return versions
            
        try:
            # Simple regex-based HTML parsing (avoids external dependencies)
            # Look for version-like patterns in text content
            lines = html_content.split('\n')
            
            # Extract text content (crude but works for simple cases)
            text_content = " ".join(
                re.sub(r'<[^>]+>', ' ', line) for line in lines[:1000]
            )
            
            # Look for version mentions in text
            version_matches = []
            for pattern in self.VERSION_PATTERNS:
                matches = re.finditer(pattern, text_content, re.IGNORECASE)
                version_matches.extend(matches)
            
            # Process unique versions
            seen_versions = set()
            for match in version_matches:
                version = match.group(1)
                if version not in seen_versions and re.match(r'^\d+\.\d+(\.\d+)?$', version):
                    seen_versions.add(version)
                    
                    # Try to find nearby context
                    start = max(0, match.start() - 100)
                    end = min(len(text_content), match.end() + 100)
                    context = text_content[start:end].strip()
                    
                    # Create version info
                    version_info = ScrapedVersionInfo(
                        version=version,
                        source="web_scraping",
                        url=base_url,
                        changelog=context if len(context) < 500 else context[:500] + "...",
                        scraped_at=datetime.now().isoformat()
                    )
                    versions.append(version_info)
            
        except Exception as e:
            logger.error(f"Error parsing HTML for versions: {e}")
            
        return versions
    
    def scrape_official_website(self) -> List[ScrapedVersionInfo]:
        """
        Scrape Deepseek official website for version information.
        
        Returns:
            List of ScrapedVersionInfo objects
        """
        logger.info(f"Scraping official website: {self.OFFICIAL_WEBSITE}")
        versions = []
        
        try:
            # Fetch homepage
            html_content = self._fetch_url(self.OFFICIAL_WEBSITE)
            if html_content:
                versions.extend(self._parse_html_for_versions(
                    html_content, 
                    self.OFFICIAL_WEBSITE
                ))
            
            # Try to find documentation or download pages
            # Look for links that might contain version info
            if html_content:
                # Search for links containing "release", "download", "version", etc.
                link_patterns = [
                    r'href=["\']([^"\']*release[^"\']*)["\']',
                    r'href=["\']([^"\']*download[^"\']*)["\']',
                    r'href=["\']([^"\']*version[^"\']*)["\']',
                ]
                
                for pattern in link_patterns:
                    matches = re.finditer(pattern, html_content, re.IGNORECASE)
                    for match in matches:
                        link = match.group(1)
                        if not link.startswith(('http://', 'https://')):
                            link = urljoin(self.OFFICIAL_WEBSITE, link)
                        
                        # Fetch linked page
                        linked_content = self._fetch_url(link)
                        if linked_content:
                            versions.extend(self._parse_html_for_versions(
                                linked_content,
                                link
                            ))
            
        except Exception as e:
            logger.error(f"Error scraping official website: {e}")
            
        return versions
    
    def scrape_github_releases(self, use_api: bool = True) -> List[ScrapedVersionInfo]:
        """
        Scrape GitHub releases page for version information.
        
        Args:
            use_api: Whether to try GitHub API first (recommended)
            
        Returns:
            List of ScrapedVersionInfo objects
        """
        logger.info(f"Scraping GitHub releases: {self.GITHUB_RELEASES}")
        versions = []
        
        try:
            # Try GitHub API first (more reliable, rate-limited)
            if use_api:
                api_versions = self._scrape_github_api()
                if api_versions:
                    versions.extend(api_versions)
                    logger.info(f"Found {len(api_versions)} versions via GitHub API")
                    return versions
            
            # Fallback to HTML scraping
            html_content = self._fetch_url(self.GITHUB_RELEASES)
            if not html_content:
                return versions
            
            # Parse GitHub releases HTML
            # GitHub releases page has specific structure
            versions.extend(self._parse_github_html(html_content))
            
        except Exception as e:
            logger.error(f"Error scraping GitHub releases: {e}")
            
        return versions
    
    def _scrape_github_api(self) -> List[ScrapedVersionInfo]:
        """
        Scrape GitHub API for releases.
        
        Returns:
            List of ScrapedVersionInfo objects
        """
        versions = []
        
        try:
            api_content = self._fetch_url(self.GITHUB_API)
            if not api_content:
                return versions
                
            releases = json.loads(api_content)
            
            for release in releases:
                if isinstance(release, dict):
                    # Extract version from tag_name
                    tag_name = release.get('tag_name', '')
                    version = self._extract_version_from_text(tag_name)
                    
                    if version:
                        version_info = ScrapedVersionInfo(
                            version=version,
                            source="github_api",
                            url=release.get('html_url', self.GITHUB_RELEASES),
                            release_date=release.get('published_at'),
                            changelog=release.get('body', '')[:500],
                            scraped_at=datetime.now().isoformat()
                        )
                        versions.append(version_info)
                        
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse GitHub API response: {e}")
        except Exception as e:
            logger.error(f"Error parsing GitHub API data: {e}")
            
        return versions
    
    def _parse_github_html(self, html_content: str) -> List[ScrapedVersionInfo]:
        """
        Parse GitHub releases HTML page.
        
        Args:
            html_content: HTML from GitHub releases page
            
        Returns:
            List of ScrapedVersionInfo objects
        """
        versions = []
        
        try:
            # GitHub releases page structure
            # Look for release entries
            release_patterns = [
                r'<div[^>]*class="release-entry"[^>]*>.*?</div>',
                r'<li[^>]*class="release"[^>]*>.*?</li>',
            ]
            
            for pattern in release_patterns:
                matches = re.finditer(pattern, html_content, re.DOTALL | re.IGNORECASE)
                for match in matches:
                    release_html = match.group(0)
                    
                    # Extract version/tag
                    version = None
                    
                    # Try to find version in various places
                    version_sources = [
                        r'<a[^>]*href="[^"]*/tag/([^"/]+)"',
                        r'<span[^>]*class="css-truncate-target"[^>]*>([^<]+)</span>',
                        r'<h2[^>]*>([^<]+)</h2>',
                    ]
                    
                    for source_pattern in version_sources:
                        v_match = re.search(source_pattern, release_html, re.IGNORECASE)
                        if v_match:
                            version = self._extract_version_from_text(v_match.group(1))
                            if version:
                                break
                    
                    if version:
                        # Extract date if available
                        date_match = re.search(
                            r'<relative-time[^>]*datetime="([^"]+)"',
                            release_html
                        )
                        release_date = date_match.group(1) if date_match else None
                        
                        # Extract release notes snippet
                        notes_match = re.search(
                            r'<div[^>]*class="markdown-body"[^>]*>(.*?)</div>',
                            release_html,
                            re.DOTALL
                        )
                        changelog = notes_match.group(1) if notes_match else None
                        if changelog:
                            # Clean HTML tags
                            changelog = re.sub(r'<[^>]+>', ' ', changelog)
                            changelog = ' '.join(changelog.split())[:500]
                        
                        version_info = ScrapedVersionInfo(
                            version=version,
                            source="github_web",
                            url=self.GITHUB_RELEASES,
                            release_date=release_date,
                            changelog=changelog,
                            scraped_at=datetime.now().isoformat()
                        )
                        versions.append(version_info)
            
        except Exception as e:
            logger.error(f"Error parsing GitHub HTML: {e}")
            
        return versions
    
    def scrape_all_sources(self) -> List[ScrapedVersionInfo]:
        """
        Scrape all available sources for version information.
        
        Returns:
            List of ScrapedVersionInfo objects from all sources
        """
        all_versions = []
        
        logger.info("Starting comprehensive Deepseek version scraping...")
        
        # Scrape official website
        website_versions = self.scrape_official_website()
        all_versions.extend(website_versions)
        logger.info(f"Found {len(website_versions)} versions on official website")
        
        # Scrape GitHub releases
        github_versions = self.scrape_github_releases()
        all_versions.extend(github_versions)
        logger.info(f"Found {len(github_versions)} versions on GitHub")
        
        # Remove duplicates (same version from multiple sources)
        unique_versions = []
        seen = set()
        
        for version_info in all_versions:
            key = (version_info.version, version_info.source)
            if key not in seen:
                seen.add(key)
                unique_versions.append(version_info)
        
        # Sort by version (simple semantic-ish sort)
        unique_versions.sort(
            key=lambda v: [int(x) for x in v.version.split('.')],
            reverse=True
        )
        
        return unique_versions
    
    def find_latest_version(self) -> Optional[ScrapedVersionInfo]:
        """
        Find the latest version across all sources.
        
        Returns:
            Latest ScrapedVersionInfo or None if none found
        """
        versions = self.scrape_all_sources()
        if versions:
            return versions[0]  # Already sorted with latest first
        return None


def main():
    """Main function for command-line usage."""
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Web scraper for Deepseek version information"
    )
    parser.add_argument(
        "--source",
        choices=["all", "website", "github"],
        default="all",
        help="Source to scrape (default: all)"
    )
    parser.add_argument(
        "--format",
        choices=["json", "text", "simple"],
        default="text",
        help="Output format (default: text)"
    )
    parser.add_argument(
        "--latest",
        action="store_true",
        help="Show only the latest version"
    )
    parser.add_argument(
        "--no-mcp",
        action="store_true",
        help="Disable MCP server usage"
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Enable verbose logging"
    )
    
    args = parser.parse_args()
    
    if args.verbose:
        logger.setLevel(logging.DEBUG)
    
    scraper = DeepseekWebScraper(use_mcp=not args.no_mcp)
    
    try:
        if args.source == "website":
            versions = scraper.scrape_official_website()
        elif args.source == "github":
            versions = scraper.scrape_github_releases()
        else:  # all
            versions = scraper.scrape_all_sources()
        
        if args.latest and versions:
            versions = [versions[0]]
        
        if not versions:
            print("No versions found")
            return 1
        
        if args.format == "json":
            # Convert to JSON-serializable dict
            output = []
            for v in versions:
                v_dict = {
                    "version": v.version,
                    "source": v.source,
                    "url": v.url,
                    "release_date": v.release_date,
                    "changelog_preview": v.changelog,
                    "scraped_at": v.scraped_at
                }
                output.append(v_dict)
            print(json.dumps(output, indent=2))
            
        elif args.format == "simple":
            for v in versions:
                print(f"{v.version} ({v.source})")
                
        else:  # text format
            print("=" * 60)
            print("Deepseek Version Scraper Results")
            print("=" * 60)
            
            for i, version_info in enumerate(versions, 1):
                print(f"\n{i}. Version: {version_info.version}")
                print(f"   Source: {version_info.source}")
                print(f"   URL: {version_info.url}")
                if version_info.release_date:
                    print(f"   Release Date: {version_info.release_date}")
                if version_info.changelog:
                    print(f"   Changelog Preview: {version_info.changelog}")
                if version_info.scraped_at:
                    print(f"   Scraped: {version_info.scraped_at}")
            
            print(f"\nTotal versions found: {len(versions)}")
            latest = scraper.find_latest_version()
            if latest:
                print(f"Latest version: {latest.version} (from {latest.source})")
        
        return 0
        
    except KeyboardInterrupt:
        print("\nScraping interrupted by user")
        return 130
    except Exception as e:
        logger.error(f"Error during scraping: {e}")
        if args.verbose:
            import traceback
            traceback.print_exc()
        return 1


if __name__ == "__main__":
    sys.exit(main())
[CODE END]