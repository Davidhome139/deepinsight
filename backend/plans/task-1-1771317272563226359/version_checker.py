#!/usr/bin/env python3
"""
Version Checker - Analyze software versions and determine if they're the latest.

This module provides functionality to check version strings against various sources
to determine if a given version is the latest available version.
"""

import re
import logging
import sys
from typing import Optional, Tuple, Dict, Any, List
from dataclasses import dataclass
from enum import Enum
import json
import requests
from packaging import version
from urllib.parse import urljoin
from datetime import datetime

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class VersionSource(Enum):
    """Enum representing different sources for version checking."""
    PYPI = "pypi"
    GITHUB = "github"
    NPM = "npm"
    DOCKER = "docker"
    CUSTOM = "custom"


@dataclass
class VersionInfo:
    """Data class to store version information."""
    current_version: str
    latest_version: Optional[str] = None
    is_latest: Optional[bool] = None
    last_checked: Optional[datetime] = None
    source: Optional[VersionSource] = None
    release_date: Optional[datetime] = None
    release_notes: Optional[str] = None
    error: Optional[str] = None


class VersionChecker:
    """
    Main class for checking if a version is the latest available.
    
    This class provides methods to check versions from various sources
    including PyPI, GitHub, NPM, Docker Hub, and custom endpoints.
    """
    
    def __init__(self, timeout: int = 10, verify_ssl: bool = True):
        """
        Initialize the VersionChecker.
        
        Args:
            timeout: Request timeout in seconds
            verify_ssl: Whether to verify SSL certificates
        """
        self.timeout = timeout
        self.verify_ssl = verify_ssl
        self.session = requests.Session()
        
    def check_version(
        self,
        current_version: str,
        source: VersionSource,
        package_name: Optional[str] = None,
        api_url: Optional[str] = None,
        version_parser: Optional[callable] = None
    ) -> VersionInfo:
        """
        Check if the current version is the latest available.
        
        Args:
            current_version: The version to check (e.g., "1.2.3")
            source: The source to check against
            package_name: Name of the package (required for some sources)
            api_url: Custom API URL (required for CUSTOM source)
            version_parser: Custom function to parse version from API response
            
        Returns:
            VersionInfo object with comparison results
            
        Raises:
            ValueError: If required parameters are missing
        """
        version_info = VersionInfo(
            current_version=current_version,
            source=source,
            last_checked=datetime.now()
        )
        
        try:
            if source == VersionSource.PYPI:
                latest_version = self._get_latest_pypi_version(package_name)
            elif source == VersionSource.GITHUB:
                latest_version = self._get_latest_github_version(package_name)
            elif source == VersionSource.NPM:
                latest_version = self._get_latest_npm_version(package_name)
            elif source == VersionSource.DOCKER:
                latest_version = self._get_latest_docker_version(package_name)
            elif source == VersionSource.CUSTOM:
                if not api_url:
                    raise ValueError("api_url is required for CUSTOM source")
                latest_version = self._get_latest_custom_version(api_url, version_parser)
            else:
                raise ValueError(f"Unsupported source: {source}")
            
            version_info.latest_version = latest_version
            version_info.is_latest = self._compare_versions(current_version, latest_version)
            
        except Exception as e:
            error_msg = f"Error checking version from {source.value}: {str(e)}"
            logger.error(error_msg)
            version_info.error = error_msg
            
        return version_info
    
    def _get_latest_pypi_version(self, package_name: str) -> str:
        """
        Get the latest version from PyPI.
        
        Args:
            package_name: Name of the Python package
            
        Returns:
            Latest version string
            
        Raises:
            requests.RequestException: If API request fails
            KeyError: If response format is unexpected
        """
        if not package_name:
            raise ValueError("package_name is required for PyPI source")
            
        url = f"https://pypi.org/pypi/{package_name}/json"
        response = self.session.get(url, timeout=self.timeout, verify=self.verify_ssl)
        response.raise_for_status()
        
        data = response.json()
        return data["info"]["version"]
    
    def _get_latest_github_version(self, repo_path: str) -> str:
        """
        Get the latest release version from GitHub.
        
        Args:
            repo_path: Repository path in format "owner/repo"
            
        Returns:
            Latest version string
            
        Raises:
            requests.RequestException: If API request fails
            KeyError: If response format is unexpected
        """
        if not repo_path:
            raise ValueError("repo_path is required for GitHub source")
            
        url = f"https://api.github.com/repos/{repo_path}/releases/latest"
        headers = {"Accept": "application/vnd.github.v3+json"}
        
        response = self.session.get(url, headers=headers, timeout=self.timeout, verify=self.verify_ssl)
        response.raise_for_status()
        
        data = response.json()
        # Remove 'v' prefix if present
        version_str = data["tag_name"]
        return version_str.lstrip('v')
    
    def _get_latest_npm_version(self, package_name: str) -> str:
        """
        Get the latest version from NPM registry.
        
        Args:
            package_name: Name of the NPM package
            
        Returns:
            Latest version string
            
        Raises:
            requests.RequestException: If API request fails
            KeyError: If response format is unexpected
        """
        if not package_name:
            raise ValueError("package_name is required for NPM source")
            
        url = f"https://registry.npmjs.org/{package_name}/latest"
        response = self.session.get(url, timeout=self.timeout, verify=self.verify_ssl)
        response.raise_for_status()
        
        data = response.json()
        return data["version"]
    
    def _get_latest_docker_version(self, image_name: str) -> str:
        """
        Get the latest version tag from Docker Hub.
        
        Args:
            image_name: Name of the Docker image
            
        Returns:
            Latest version string
            
        Raises:
            requests.RequestException: If API request fails
            RuntimeError: If no valid version tags found
        """
        if not image_name:
            raise ValueError("image_name is required for Docker source")
            
        # Docker Hub API endpoint for tags
        url = f"https://hub.docker.com/v2/repositories/{image_name}/tags/"
        params = {"page_size": 50, "ordering": "last_updated"}
        
        response = self.session.get(url, params=params, timeout=self.timeout, verify=self.verify_ssl)
        response.raise_for_status()
        
        data = response.json()
        results = data.get("results", [])
        
        # Filter for version-like tags
        version_pattern = re.compile(r'^\d+\.\d+\.\d+$')
        version_tags = []
        
        for tag in results:
            tag_name = tag.get("name", "")
            if version_pattern.match(tag_name):
                version_tags.append(tag_name)
        
        if not version_tags:
            raise RuntimeError(f"No version tags found for {image_name}")
        
        # Sort by version and return latest
        version_tags.sort(key=lambda x: version.parse(x))
        return version_tags[-1]
    
    def _get_latest_custom_version(
        self,
        api_url: str,
        version_parser: Optional[callable] = None
    ) -> str:
        """
        Get the latest version from a custom API endpoint.
        
        Args:
            api_url: URL of the custom API endpoint
            version_parser: Function to extract version from API response
            
        Returns:
            Latest version string
            
        Raises:
            requests.RequestException: If API request fails
            ValueError: If version cannot be parsed
        """
        response = self.session.get(api_url, timeout=self.timeout, verify=self.verify_ssl)
        response.raise_for_status()
        
        data = response.json()
        
        if version_parser:
            return version_parser(data)
        
        # Try to find version in common response formats
        for key in ["version", "latest_version", "tag_name", "release"]:
            if key in data:
                version_str = str(data[key])
                # Remove 'v' prefix if present
                return version_str.lstrip('v')
        
        raise ValueError("Could not extract version from API response")
    
    def _compare_versions(self, version1: str, version2: str) -> bool:
        """
        Compare two version strings.
        
        Args:
            version1: First version string
            version2: Second version string
            
        Returns:
            True if version1 >= version2 (version1 is same or newer)
            
        Raises:
            version.InvalidVersion: If version strings are invalid
        """
        try:
            v1 = version.parse(version1)
            v2 = version.parse(version2)
            return v1 >= v2
        except version.InvalidVersion as e:
            logger.warning(f"Invalid version format: {e}")
            # Fall back to string comparison for non-standard versions
            return version1 == version2
    
    def check_multiple_versions(
        self,
        versions: List[Tuple[str, Dict[str, Any]]]
    ) -> List[VersionInfo]:
        """
        Check multiple versions in batch.
        
        Args:
            versions: List of tuples (current_version, config_dict)
            
        Returns:
            List of VersionInfo objects
        """
        results = []
        for current_version, config in versions:
            try:
                source = VersionSource(config.get("source", "custom"))
                result = self.check_version(
                    current_version=current_version,
                    source=source,
                    package_name=config.get("package_name"),
                    api_url=config.get("api_url"),
                    version_parser=config.get("version_parser")
                )
                results.append(result)
            except Exception as e:
                logger.error(f"Error checking version {current_version}: {e}")
                results.append(VersionInfo(
                    current_version=current_version,
                    error=str(e)
                ))
        
        return results
    
    def __enter__(self):
        """Context manager entry."""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit - close session."""
        self.session.close()
    
    def close(self):
        """Close the HTTP session."""
        self.session.close()


def parse_version_string(version_str: str) -> Dict[str, Any]:
    """
    Parse a version string into its components.
    
    Args:
        version_str: Version string to parse
        
    Returns:
        Dictionary with parsed components
    """
    # Common version patterns
    patterns = [
        r'^(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[a-zA-Z0-9.-]+))?(?:\+(?P<build>[a-zA-Z0-9.-]+))?$',
        r'^(?P<major>\d+)\.(?P<minor>\d+)$',
        r'^v?(?P<full>\d+(?:\.\d+)*)$',
    ]
    
    for pattern in patterns:
        match = re.match(pattern, version_str)
        if match:
            return {k: v for k, v in match.groupdict().items() if v is not None}
    
    return {"raw": version_str}


def main():
    """Command-line interface for version checking."""
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Check if a software version is the latest available"
    )
    parser.add_argument(
        "current_version",
        help="Current version to check"
    )
    parser.add_argument(
        "--source",
        choices=[s.value for s in VersionSource],
        default="pypi",
        help="Source to check against"
    )
    parser.add_argument(
        "--package",
        help="Package name (for PyPI, NPM, Docker sources)"
    )
    parser.add_argument(
        "--repo",
        help="GitHub repository in format 'owner/repo' (for GitHub source)"
    )
    parser.add_argument(
        "--api-url",
        help="Custom API URL (for custom source)"
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=10,
        help="Request timeout in seconds"
    )
    parser.add_argument(
        "--no-ssl-verify",
        action="store_true",
        help="Disable SSL certificate verification"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output results as JSON"
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Enable verbose logging"
    )
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    # Determine package name based on source
    package_name = args.package
    if args.source == VersionSource.GITHUB.value and args.repo:
        package_name = args.repo
    
    with VersionChecker(timeout=args.timeout, verify_ssl=not args.no_ssl_verify) as checker:
        try:
            result = checker.check_version(
                current_version=args.current_version,
                source=VersionSource(args.source),
                package_name=package_name,
                api_url=args.api_url
            )
            
            if args.json:
                output = {
                    "current_version": result.current_version,
                    "latest_version": result.latest_version,
                    "is_latest": result.is_latest,
                    "source": result.source.value if result.source else None,
                    "last_checked": result.last_checked.isoformat() if result.last_checked else None,
                    "error": result.error
                }
                print(json.dumps(output, indent=2))
            else:
                if result.error:
                    print(f"Error: {result.error}")
                    sys.exit(1)
                
                if result.latest_version:
                    status = "LATEST" if result.is_latest else "OUTDATED"
                    print(f"Current version: {result.current_version}")
                    print(f"Latest version:  {result.latest_version}")
                    print(f"Status: {status}")
                    
                    if not result.is_latest:
                        print(f"Update available: {result.current_version} → {result.latest_version}")
                        sys.exit(2)  # Exit code 2 for outdated version
                    else:
                        sys.exit(0)  # Exit code 0 for latest version
                else:
                    print(f"Could not determine latest version")
                    sys.exit(1)
                    
        except KeyboardInterrupt:
            print("\nOperation cancelled by user")
            sys.exit(130)
        except Exception as e:
            print(f"Error: {str(e)}")
            sys.exit(1)


if __name__ == "__main__":
    main()