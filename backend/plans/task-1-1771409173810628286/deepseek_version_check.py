#!/usr/bin/env python3
"""
Script to fetch the latest version information for DeepSeek AI models and tools.
This script checks multiple sources for version information including GitHub releases
and official documentation.
"""

import argparse
import json
import sys
import time
from datetime import datetime
from typing import Dict, Optional, Any
from urllib import request, error

# Constants
GITHUB_API_BASE = "https://api.github.com"
DEEPSEEK_REPOS = [
    "deepseek-ai/DeepSeek-Coder",
    "deepseek-ai/DeepSeek-LLM",
    "deepseek-ai/DeepSeek-V2",
    "deepseek-ai/DeepSeek-Math"
]
USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
REQUEST_TIMEOUT = 30  # seconds


class VersionChecker:
    """Main class for checking DeepSeek version information."""
    
    def __init__(self, verbose: bool = False) -> None:
        """
        Initialize the version checker.
        
        Args:
            verbose: Enable verbose output
        """
        self.verbose = verbose
        self.results: Dict[str, Any] = {}
    
    def _make_request(self, url: str) -> Optional[Dict[str, Any]]:
        """
        Make an HTTP GET request to the specified URL.
        
        Args:
            url: The URL to fetch
            
        Returns:
            Parsed JSON response or None if request fails
        """
        try:
            req = request.Request(
                url,
                headers={
                    "User-Agent": USER_AGENT,
                    "Accept": "application/vnd.github.v3+json"
                }
            )
            
            with request.urlopen(req, timeout=REQUEST_TIMEOUT) as response:
                if response.status == 200:
                    data = response.read().decode("utf-8")
                    return json.loads(data)
                else:
                    if self.verbose:
                        print(f"HTTP {response.status} for {url}")
                    return None
                    
        except error.URLError as e:
            if self.verbose:
                print(f"Network error for {url}: {e}")
            return None
        except json.JSONDecodeError as e:
            if self.verbose:
                print(f"JSON decode error for {url}: {e}")
            return None
        except Exception as e:
            if self.verbose:
                print(f"Unexpected error for {url}: {e}")
            return None
    
    def check_github_repo(self, repo: str) -> Optional[Dict[str, Any]]:
        """
        Check the latest release for a GitHub repository.
        
        Args:
            repo: Repository name in format "owner/repo"
            
        Returns:
            Release information or None if not available
        """
        url = f"{GITHUB_API_BASE}/repos/{repo}/releases/latest"
        
        if self.verbose:
            print(f"Checking {repo}...")
        
        data = self._make_request(url)
        
        if data and self.verbose:
            print(f"Found release: {data.get('tag_name', 'Unknown')}")
        
        return data
    
    def check_all_repos(self) -> Dict[str, Dict[str, Any]]:
        """
        Check all configured DeepSeek repositories.
        
        Returns:
            Dictionary with repository names as keys and release info as values
        """
        results = {}
        
        print("Fetching DeepSeek version information...")
        print("-" * 50)
        
        for repo in DEEPSEEK_REPOS:
            release_info = self.check_github_repo(repo)
            if release_info:
                results[repo] = {
                    "latest_version": release_info.get("tag_name", "Unknown"),
                    "release_date": release_info.get("published_at", "Unknown"),
                    "release_name": release_info.get("name", "Unnamed"),
                    "url": release_info.get("html_url", ""),
                    "prerelease": release_info.get("prerelease", False),
                    "assets_count": len(release_info.get("assets", [])),
                    "body_preview": (release_info.get("body", "")[:200] + "...") 
                                   if release_info.get("body") else "No description"
                }
            else:
                results[repo] = {
                    "latest_version": "Error fetching data",
                    "release_date": "Unknown",
                    "release_name": "Error",
                    "url": "",
                    "prerelease": False,
                    "assets_count": 0,
                    "body_preview": "Failed to fetch release information"
                }
            
            # Be respectful to GitHub API
            time.sleep(0.5)
        
        self.results = results
        return results
    
    def format_results(self, output_format: str = "table") -> str:
        """
        Format the results in the specified output format.
        
        Args:
            output_format: One of "table", "json", or "simple"
            
        Returns:
            Formatted results as string
        """
        if not self.results:
            return "No results to display. Run check_all_repos() first."
        
        if output_format == "json":
            return json.dumps(self.results, indent=2, ensure_ascii=False)
        
        elif output_format == "simple":
            output_lines = []
            for repo, info in self.results.items():
                output_lines.append(
                    f"{repo}: {info['latest_version']} "
                    f"(Released: {info['release_date'][:10] if info['release_date'] != 'Unknown' else 'Unknown'})"
                )
            return "\n".join(output_lines)
        
        else:  # table format (default)
            output_lines = []
            header = f"{'Repository':<30} {'Latest Version':<20} {'Release Date':<15} {'Status':<10}"
            separator = "-" * 85
            output_lines.append(header)
            output_lines.append(separator)
            
            for repo, info in self.results.items():
                # Shorten repository name if too long
                repo_display = repo if len(repo) <= 29 else repo[:26] + "..."
                
                # Format date
                date_str = info['release_date']
                if date_str != 'Unknown' and len(date_str) >= 10:
                    date_display = date_str[:10]
                else:
                    date_display = date_str
                
                # Determine status
                status = "PRERELEASE" if info['prerelease'] else "STABLE"
                status_color = "🟡" if info['prerelease'] else "🟢"
                
                output_lines.append(
                    f"{repo_display:<30} {info['latest_version']:<20} "
                    f"{date_display:<15} {status_color} {status:<8}"
                )
            
            return "\n".join(output_lines)
    
    def get_summary(self) -> Dict[str, Any]:
        """
        Get a summary of all version information.
        
        Returns:
            Dictionary with summary statistics
        """
        if not self.results:
            return {}
        
        total_repos = len(self.results)
        successful_checks = sum(1 for info in self.results.values() 
                              if info['latest_version'] != "Error fetching data")
        prereleases = sum(1 for info in self.results.values() if info['prerelease'])
        
        # Find most recent release
        latest_release = None
        latest_date = None
        
        for repo, info in self.results.items():
            if info['release_date'] != 'Unknown':
                try:
                    release_date = datetime.fromisoformat(info['release_date'].replace('Z', '+00:00'))
                    if latest_date is None or release_date > latest_date:
                        latest_date = release_date
                        latest_release = repo
                except ValueError:
                    continue
        
        return {
            "total_repositories": total_repos,
            "successful_checks": successful_checks,
            "failed_checks": total_repos - successful_checks,
            "prereleases": prereleases,
            "stable_releases": successful_checks - prereleases,
            "most_recent_release": latest_release,
            "most_recent_date": latest_date.isoformat() if latest_date else None,
            "check_timestamp": datetime.now().isoformat()
        }


def main() -> None:
    """Main function to parse arguments and run the version check."""
    parser = argparse.ArgumentParser(
        description="Check for latest DeepSeek AI versions",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                    # Check all repos with table output
  %(prog)s --format json      # Output in JSON format
  %(prog)s --format simple    # Simple one-line per repo output
  %(prog)s --verbose          # Show detailed progress information
  %(prog)s --summary-only     # Show only summary statistics
        """
    )
    
    parser.add_argument(
        "-f", "--format",
        choices=["table", "json", "simple"],
        default="table",
        help="Output format (default: table)"
    )
    
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Enable verbose output"
    )
    
    parser.add_argument(
        "-s", "--summary-only",
        action="store_true",
        help="Show only summary statistics"
    )
    
    args = parser.parse_args()
    
    # Create version checker
    checker = VersionChecker(verbose=args.verbose)
    
    try:
        # Check all repositories
        results = checker.check_all_repos()
        
        if not results:
            print("Error: No results obtained. Check your internet connection.")
            sys.exit(1)
        
        # Display results based on arguments
        if args.summary_only:
            summary = checker.get_summary()
            print("\n📊 SUMMARY")
            print("-" * 40)
            print(f"Total repositories checked: {summary['total_repositories']}")
            print(f"Successful checks: {summary['successful_checks']}")
            print(f"Failed checks: {summary['failed_checks']}")
            print(f"Stable releases: {summary['stable_releases']}")
            print(f"Pre-releases: {summary['prereleases']}")
            
            if summary['most_recent_release']:
                print(f"\nMost recent release: {summary['most_recent_release']}")
                print(f"Released on: {summary['most_recent_date'][:10]}")
            
            print(f"\nCheck performed at: {summary['check_timestamp'][:19]}")
        else:
            # Display main results
            formatted_output = checker.format_results(args.format)
            print(formatted_output)
            
            # Always show summary at the end (except for JSON output)
            if args.format != "json":
                summary = checker.get_summary()
                print("\n" + "-" * 50)
                print(f"✓ Checked {summary['successful_checks']}/{summary['total_repositories']} repositories")
                if summary['failed_checks'] > 0:
                    print(f"⚠ {summary['failed_checks']} repositories failed to fetch")
    
    except KeyboardInterrupt:
        print("\n\n⚠ Operation cancelled by user.")
        sys.exit(130)
    except Exception as e:
        print(f"\n❌ Error: {e}")
        if args.verbose:
            import traceback
            traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()