#!/usr/bin/env python3
"""
Script to check for the latest DeepSeek version and release date.
Uses the GitHub API to fetch release information from the DeepSeek repository.
"""

import sys
import json
import argparse
from datetime import datetime
from typing import Optional, Dict, Any, Tuple
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError
from urllib.parse import urljoin

# Constants
GITHUB_API_BASE = "https://api.github.com/"
DEEPSEEK_REPO = "deepseek-ai/DeepSeek-Coder"  # Main DeepSeek repository
USER_AGENT = "check_deepseek/1.0"


def fetch_github_api(endpoint: str) -> Dict[str, Any]:
    """
    Fetch data from GitHub API with proper error handling.
    
    Args:
        endpoint: GitHub API endpoint (e.g., 'repos/deepseek-ai/DeepSeek-Coder/releases')
        
    Returns:
        Parsed JSON response as dictionary
        
    Raises:
        SystemExit: If the request fails or returns non-200 status
    """
    url = urljoin(GITHUB_API_BASE, endpoint)
    
    try:
        # GitHub API requires a User-Agent header
        headers = {
            "User-Agent": USER_AGENT,
            "Accept": "application/vnd.github.v3+json"
        }
        request = Request(url, headers=headers)
        
        with urlopen(request, timeout=10) as response:
            if response.status != 200:
                raise HTTPError(url, response.status, "Non-200 status code", 
                              response.headers, None)
            
            data = response.read()
            return json.loads(data)
            
    except HTTPError as e:
        if e.code == 404:
            print(f"Error: Repository or endpoint not found: {url}", file=sys.stderr)
            print("Please check if the repository path is correct.", file=sys.stderr)
        elif e.code == 403:
            print("Error: GitHub API rate limit may have been exceeded.", file=sys.stderr)
            print("Try again later or use a GitHub token for higher limits.", file=sys.stderr)
        else:
            print(f"HTTP Error {e.code}: {e.reason}", file=sys.stderr)
        sys.exit(1)
        
    except URLError as e:
        print(f"Network Error: {e.reason}", file=sys.stderr)
        print("Please check your internet connection and try again.", file=sys.stderr)
        sys.exit(1)
        
    except json.JSONDecodeError as e:
        print(f"Error: Failed to parse JSON response: {e}", file=sys.stderr)
        sys.exit(1)
        
    except TimeoutError:
        print("Error: Request timed out. Please try again later.", file=sys.stderr)
        sys.exit(1)


def get_latest_release(repo: str = DEEPSEEK_REPO) -> Tuple[Optional[str], Optional[str], Optional[str]]:
    """
    Get the latest release information from a GitHub repository.
    
    Args:
        repo: GitHub repository in format 'owner/repo'
        
    Returns:
        Tuple of (version, release_date, release_notes)
        Returns (None, None, None) if no releases found
    """
    endpoint = f"repos/{repo}/releases/latest"
    
    try:
        release_data = fetch_github_api(endpoint)
        
        version = release_data.get("tag_name")
        published_at = release_data.get("published_at")
        release_notes = release_data.get("body")
        
        # Format the date nicely if available
        if published_at:
            try:
                dt = datetime.fromisoformat(published_at.replace("Z", "+00:00"))
                release_date = dt.strftime("%Y-%m-%d %H:%M:%S UTC")
            except (ValueError, AttributeError):
                release_date = published_at
        else:
            release_date = None
            
        return version, release_date, release_notes
        
    except SystemExit:
        # fetch_github_api already printed error and exited
        raise
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        sys.exit(1)


def format_output(version: Optional[str], release_date: Optional[str], 
                  release_notes: Optional[str], verbose: bool = False) -> str:
    """
    Format the output for display.
    
    Args:
        version: Release version/tag
        release_date: Formatted release date
        release_notes: Release notes/body
        verbose: Whether to show detailed output
        
    Returns:
        Formatted output string
    """
    if not version or not release_date:
        return "No releases found or repository is empty."
    
    output = [
        "=" * 60,
        "DeepSeek Release Information",
        "=" * 60,
        f"Latest Version: {version}",
        f"Release Date:   {release_date}",
        "=" * 60,
    ]
    
    if verbose and release_notes:
        output.extend([
            "Release Notes:",
            "-" * 40,
            release_notes[:500] + "..." if len(release_notes) > 500 else release_notes,
            "=" * 60
        ])
    
    return "\n".join(output)


def parse_arguments() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Check for the latest DeepSeek version and release date",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                    # Check default DeepSeek-Coder repository
  %(prog)s --repo deepseek-ai/DeepSeek-V2  # Check different repository
  %(prog)s -v                 # Show verbose output with release notes
  %(prog)s --json             # Output in JSON format for machine parsing
        """
    )
    
    parser.add_argument(
        "--repo", "-r",
        default=DEEPSEEK_REPO,
        help=f"GitHub repository (format: owner/repo). Default: {DEEPSEEK_REPO}"
    )
    
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Show verbose output including release notes"
    )
    
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output in JSON format (machine-readable)"
    )
    
    return parser.parse_args()


def main() -> None:
    """Main function."""
    args = parse_arguments()
    
    try:
        version, release_date, release_notes = get_latest_release(args.repo)
        
        if args.json:
            # JSON output for machine parsing
            result = {
                "repository": args.repo,
                "latest_version": version,
                "release_date": release_date,
                "release_notes": release_notes if args.verbose else None,
                "timestamp": datetime.utcnow().isoformat() + "Z",
                "success": version is not None
            }
            print(json.dumps(result, indent=2))
        else:
            # Human-readable output
            print(f"Repository: {args.repo}")
            if version is None:
                print("No releases found.")
                sys.exit(1)
            else:
                output = format_output(version, release_date, release_notes, args.verbose)
                print(output)
                
    except KeyboardInterrupt:
        print("\nOperation cancelled by user.", file=sys.stderr)
        sys.exit(130)
    except Exception as e:
        print(f"An unexpected error occurred: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()