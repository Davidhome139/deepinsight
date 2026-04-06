#!/usr/bin/env python3
"""
Script to check the latest Deepseek version by scraping official sources.
Uses available MCP servers for search functionality and terminal operations.
"""

import json
import re
import sys
import logging
from typing import Optional, Tuple, Dict, Any
from dataclasses import dataclass
from datetime import datetime
import subprocess
import shlex

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@dataclass
class VersionInfo:
    """Data class to hold version information."""
    version: str
    source: str
    release_date: Optional[str] = None
    release_notes_url: Optional[str] = None
    download_url: Optional[str] = None


class DeepseekVersionChecker:
    """Main class to check Deepseek version from official sources."""
    
    # Official sources to check
    OFFICIAL_SOURCES = {
        "github": "https://github.com/deepseek-ai/DeepSeek-Coder",
        "huggingface": "https://huggingface.co/deepseek-ai",
        "official_docs": "https://www.deepseek.com",
        "pypi": "https://pypi.org/project/deepseek/"
    }
    
    # Version pattern regex
    VERSION_PATTERNS = [
        r'v?(\d+\.\d+\.\d+)',  # Standard semver: 1.2.3 or v1.2.3
        r'v?(\d+\.\d+)',       # Major.minor: 1.2 or v1.2
        r'release-(\d+\.\d+\.\d+)',  # release-1.2.3
    ]
    
    def __init__(self):
        """Initialize the version checker."""
        self.version_info: Optional[VersionInfo] = None
        self._validate_environment()
    
    def _validate_environment(self) -> None:
        """Validate that required tools are available."""
        try:
            # Check if curl is available (for web scraping)
            subprocess.run(['curl', '--version'], 
                         capture_output=True, check=True, timeout=5)
        except (subprocess.SubprocessError, FileNotFoundError):
            logger.warning("curl not found. Some sources may not be accessible.")
    
    def _run_mcp_search(self, query: str) -> Optional[Dict[str, Any]]:
        """
        Run search using MCP search server.
        
        Args:
            query: Search query string
            
        Returns:
            Search results or None if search fails
        """
        try:
            # This would be replaced with actual MCP server call
            # For now, we'll simulate the search functionality
            logger.info(f"Searching for: {query}")
            
            # In a real implementation, this would call the MCP search server
            # For example: search.search(query, limit=5)
            
            # For now, we'll use fallback methods
            return None
            
        except Exception as e:
            logger.error(f"Search failed: {e}")
            return None
    
    def _fetch_web_content(self, url: str) -> Optional[str]:
        """
        Fetch web content using terminal commands.
        
        Args:
            url: URL to fetch
            
        Returns:
            Web content as string or None if fetch fails
        """
        try:
            # Use curl to fetch web content
            cmd = f"curl -s -L --max-time 10 --user-agent 'Mozilla/5.0' {shlex.quote(url)}"
            result = subprocess.run(
                cmd,
                shell=True,
                capture_output=True,
                text=True,
                timeout=15
            )
            
            if result.returncode == 0:
                return result.stdout
            else:
                logger.error(f"Failed to fetch {url}: {result.stderr}")
                return None
                
        except subprocess.TimeoutExpired:
            logger.error(f"Timeout fetching {url}")
            return None
        except Exception as e:
            logger.error(f"Error fetching {url}: {e}")
            return None
    
    def _extract_version_from_text(self, text: str, source: str) -> Optional[str]:
        """
        Extract version from text using regex patterns.
        
        Args:
            text: Text to search for version
            source: Source identifier for logging
            
        Returns:
            Extracted version string or None
        """
        if not text:
            return None
            
        # Try each version pattern
        for pattern in self.VERSION_PATTERNS:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                # Get the most likely version (often the first match in important sections)
                # Filter out very small numbers that might not be versions
                valid_matches = [
                    m for m in matches 
                    if len(m) >= 3 and m[0].isdigit() and int(m.split('.')[0]) > 0
                ]
                
                if valid_matches:
                    # Sort by version number (newest first)
                    try:
                        valid_matches.sort(
                            key=lambda x: tuple(map(int, x.split('.'))),
                            reverse=True
                        )
                        version = valid_matches[0]
                        logger.info(f"Found version {version} in {source}")
                        return version
                    except (ValueError, IndexError):
                        return valid_matches[0]
        
        return None
    
    def check_github_releases(self) -> Optional[VersionInfo]:
        """
        Check GitHub releases for latest version.
        
        Returns:
            VersionInfo or None if not found
        """
        try:
            # Try GitHub API first
            api_url = "https://api.github.com/repos/deepseek-ai/DeepSeek-Coder/releases/latest"
            content = self._fetch_web_content(api_url)
            
            if content:
                data = json.loads(content)
                version = data.get('tag_name', '').lstrip('v')
                if version:
                    return VersionInfo(
                        version=version,
                        source="GitHub Releases API",
                        release_date=data.get('published_at'),
                        release_notes_url=data.get('html_url'),
                        download_url=data.get('zipball_url')
                    )
            
            # Fallback: scrape GitHub releases page
            releases_url = "https://github.com/deepseek-ai/DeepSeek-Coder/releases"
            content = self._fetch_web_content(releases_url)
            
            if content:
                # Look for latest release tag
                version_patterns = [
                    r'href="/deepseek-ai/DeepSeek-Coder/releases/tag/(v?\d+\.\d+\.\d+)"',
                    r'<span class="ml-1 wb-break-all">Release (v?\d+\.\d+\.\d+)</span>'
                ]
                
                for pattern in version_patterns:
                    match = re.search(pattern, content)
                    if match:
                        version = match.group(1).lstrip('v')
                        return VersionInfo(
                            version=version,
                            source="GitHub Releases Page",
                            release_notes_url=f"{releases_url}/tag/{match.group(1)}"
                        )
                        
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse GitHub API response: {e}")
        except Exception as e:
            logger.error(f"Error checking GitHub: {e}")
            
        return None
    
    def check_pypi(self) -> Optional[VersionInfo]:
        """
        Check PyPI for Deepseek package version.
        
        Returns:
            VersionInfo or None if not found
        """
        try:
            # Try PyPI JSON API
            api_url = "https://pypi.org/pypi/deepseek/json"
            content = self._fetch_web_content(api_url)
            
            if content:
                data = json.loads(content)
                version = data.get('info', {}).get('version')
                if version:
                    return VersionInfo(
                        version=version,
                        source="PyPI",
                        release_date=data.get('releases', {}).get(version, [{}])[0].get('upload_time'),
                        download_url=f"https://pypi.org/project/deepseek/{version}/"
                    )
                    
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse PyPI response: {e}")
        except Exception as e:
            logger.error(f"Error checking PyPI: {e}")
            
        return None
    
    def check_huggingface(self) -> Optional[VersionInfo]:
        """
        Check HuggingFace for Deepseek models.
        
        Returns:
            VersionInfo or None if not found
        """
        try:
            url = "https://huggingface.co/deepseek-ai"
            content = self._fetch_web_content(url)
            
            if content:
                # Look for version information in model cards
                version = self._extract_version_from_text(content, "HuggingFace")
                if version:
                    return VersionInfo(
                        version=version,
                        source="HuggingFace",
                        download_url=url
                    )
                    
        except Exception as e:
            logger.error(f"Error checking HuggingFace: {e}")
            
        return None
    
    def check_official_docs(self) -> Optional[VersionInfo]:
        """
        Check official Deepseek documentation.
        
        Returns:
            VersionInfo or None if not found
        """
        try:
            # Try different documentation URLs
            doc_urls = [
                "https://www.deepseek.com",
                "https://docs.deepseek.com",
                "https://deepseek.com/docs"
            ]
            
            for url in doc_urls:
                content = self._fetch_web_content(url)
                if content:
                    version = self._extract_version_from_text(content, "Official Docs")
                    if version:
                        return VersionInfo(
                            version=version,
                            source="Official Website",
                            download_url=url
                        )
                        
        except Exception as e:
            logger.error(f"Error checking official docs: {e}")
            
        return None
    
    def check_all_sources(self) -> VersionInfo:
        """
        Check all available sources and return the latest version found.
        
        Returns:
            Latest VersionInfo found across all sources
            
        Raises:
            RuntimeError: If no version could be found from any source
        """
        sources = [
            ("GitHub", self.check_github_releases),
            ("PyPI", self.check_pypi),
            ("HuggingFace", self.check_huggingface),
            ("Official Docs", self.check_official_docs)
        ]
        
        found_versions = []
        
        for source_name, check_func in sources:
            try:
                logger.info(f"Checking {source_name}...")
                version_info = check_func()
                if version_info:
                    found_versions.append(version_info)
                    logger.info(f"✓ Found version {version_info.version} from {source_name}")
                else:
                    logger.warning(f"✗ No version found from {source_name}")
            except Exception as e:
                logger.error(f"Error checking {source_name}: {e}")
        
        if not found_versions:
            raise RuntimeError("Could not determine Deepseek version from any official source")
        
        # Sort by version number (newest first)
        def version_key(v: VersionInfo) -> Tuple[int, ...]:
            try:
                return tuple(map(int, v.version.split('.')))
            except (ValueError, AttributeError):
                return (0, 0, 0)
        
        found_versions.sort(key=version_key, reverse=True)
        self.version_info = found_versions[0]
        
        return self.version_info
    
    def format_output(self, version_info: VersionInfo) -> str:
        """
        Format version information for display.
        
        Args:
            version_info: VersionInfo object
            
        Returns:
            Formatted string
        """
        output = [
            "=" * 60,
            "DEEPSEEK VERSION INFORMATION",
            "=" * 60,
            f"Latest Version: {version_info.version}",
            f"Source: {version_info.source}",
        ]
        
        if version_info.release_date:
            output.append(f"Release Date: {version_info.release_date}")
        
        if version_info.release_notes_url:
            output.append(f"Release Notes: {version_info.release_notes_url}")
        
        if version_info.download_url:
            output.append(f"Download URL: {version_info.download_url}")
        
        output.append("")
        output.append("Official Sources:")
        for name, url in self.OFFICIAL_SOURCES.items():
            output.append(f"  {name}: {url}")
        
        output.append("=" * 60)
        
        return "\n".join(output)


def main() -> int:
    """
    Main function to check Deepseek version.
    
    Returns:
        Exit code (0 for success, non-zero for errors)
    """
    try:
        logger.info("Starting Deepseek version check...")
        
        checker = DeepseekVersionChecker()
        version_info = checker.check_all_sources()
        
        # Output results
        print(checker.format_output(version_info))
        
        # Log success
        logger.info(f"Successfully found Deepseek version: {version_info.version}")
        
        return 0
        
    except RuntimeError as e:
        logger.error(f"Failed to determine version: {e}")
        print(f"ERROR: {e}", file=sys.stderr)
        print("\nTroubleshooting tips:")
        print("1. Check your internet connection")
        print("2. Verify the official sources are accessible")
        print("3. Try running with debug logging: export LOG_LEVEL=DEBUG")
        return 1
        
    except KeyboardInterrupt:
        logger.info("Version check interrupted by user")
        return 130
        
    except Exception as e:
        logger.exception(f"Unexpected error: {e}")
        print(f"CRITICAL ERROR: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())