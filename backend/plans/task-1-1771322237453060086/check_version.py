#!/usr/bin/env python3
"""
Check if a given GitHub release is the latest release by parsing GitHub API responses.
"""

import json
import sys
import argparse
from typing import Dict, Any, Optional
import logging

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(levelname)s: %(message)s')
logger = logging.getLogger(__name__)


def parse_release_data(release_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Parse GitHub release data and extract relevant information.
    
    Args:
        release_data: Dictionary containing GitHub API release data
        
    Returns:
        Dictionary with parsed release information
        
    Raises:
        ValueError: If required fields are missing
    """
    required_fields = ['tag_name', 'html_url', 'published_at']
    
    for field in required_fields:
        if field not in release_data:
            raise ValueError(f"Missing required field in release data: {field}")
    
    return {
        'tag_name': release_data['tag_name'],
        'html_url': release_data['html_url'],
        'published_at': release_data['published_at'],
        'is_prerelease': release_data.get('prerelease', False),
        'is_draft': release_data.get('draft', False),
        'name': release_data.get('name', ''),
        'body': release_data.get('body', '')
    }


def compare_releases(current_release: Dict[str, Any], 
                     latest_release: Dict[str, Any]) -> Dict[str, Any]:
    """
    Compare current release with latest release to determine if current is latest.
    
    Args:
        current_release: Parsed data for the current release
        latest_release: Parsed data for the latest release
        
    Returns:
        Dictionary with comparison results
    """
    is_latest = current_release['tag_name'] == latest_release['tag_name']
    
    return {
        'is_latest': is_latest,
        'current_tag': current_release['tag_name'],
        'latest_tag': latest_release['tag_name'],
        'current_url': current_release['html_url'],
        'latest_url': latest_release['html_url'],
        'current_published_at': current_release['published_at'],
        'latest_published_at': latest_release['published_at'],
        'current_is_prerelease': current_release['is_prerelease'],
        'latest_is_prerelease': latest_release['is_prerelease']
    }


def is_latest_release(current_release_json: str, 
                      latest_release_json: str) -> Dict[str, Any]:
    """
    Main function to determine if current release is the latest.
    
    Args:
        current_release_json: JSON string of the current release API response
        latest_release_json: JSON string of the latest release API response
        
    Returns:
        Dictionary with comparison results and metadata
        
    Raises:
        json.JSONDecodeError: If input is not valid JSON
        ValueError: If required data is missing
    """
    try:
        # Parse JSON inputs
        current_data = json.loads(current_release_json)
        latest_data = json.loads(latest_release_json)
        
        # Handle both single release object and list of releases
        if isinstance(current_data, list):
            if not current_data:
                raise ValueError("Current release data is an empty list")
            current_data = current_data[0]
            
        if isinstance(latest_data, list):
            if not latest_data:
                raise ValueError("Latest release data is an empty list")
            latest_data = latest_data[0]
        
        # Parse release information
        current_release = parse_release_data(current_data)
        latest_release = parse_release_data(latest_data)
        
        # Compare releases
        result = compare_releases(current_release, latest_release)
        
        return result
        
    except json.JSONDecodeError as e:
        logger.error(f"Failed to parse JSON: {e}")
        raise
    except KeyError as e:
        logger.error(f"Missing expected field in data: {e}")
        raise ValueError(f"Missing expected field: {e}")


def load_json_from_file(filepath: str) -> str:
    """
    Load JSON content from a file.
    
    Args:
        filepath: Path to the JSON file
        
    Returns:
        JSON string content
        
    Raises:
        FileNotFoundError: If file doesn't exist
        IOError: If file cannot be read
    """
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            return f.read()
    except FileNotFoundError:
        logger.error(f"File not found: {filepath}")
        raise
    except IOError as e:
        logger.error(f"Error reading file {filepath}: {e}")
        raise


def main():
    """Command-line interface for the version checker."""
    parser = argparse.ArgumentParser(
        description="Check if a GitHub release is the latest release",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Compare two JSON files
  python check_version.py --current current.json --latest latest.json
  
  # Compare with verbose output
  python check_version.py --current current.json --latest latest.json --verbose
  
  # Output only boolean result for scripting
  python check_version.py --current current.json --latest latest.json --quiet
        """
    )
    
    # Input source options (mutually exclusive group)
    input_group = parser.add_mutually_exclusive_group(required=True)
    input_group.add_argument(
        '--current',
        type=str,
        help='JSON file containing current release data'
    )
    input_group.add_argument(
        '--current-json',
        type=str,
        help='Direct JSON string of current release data'
    )
    
    # Latest release options
    latest_group = parser.add_mutually_exclusive_group(required=True)
    latest_group.add_argument(
        '--latest',
        type=str,
        help='JSON file containing latest release data'
    )
    latest_group.add_argument(
        '--latest-json',
        type=str,
        help='Direct JSON string of latest release data'
    )
    
    # Output options
    parser.add_argument(
        '--verbose', '-v',
        action='store_true',
        help='Display detailed comparison information'
    )
    parser.add_argument(
        '--quiet', '-q',
        action='store_true',
        help='Output only boolean result (true/false)'
    )
    parser.add_argument(
        '--exit-code',
        action='store_true',
        help='Exit with code 0 if latest, 1 if not latest, 2 on error'
    )
    
    args = parser.parse_args()
    
    try:
        # Load or use direct JSON input
        if args.current:
            current_json = load_json_from_file(args.current)
        else:
            current_json = args.current_json
            
        if args.latest:
            latest_json = load_json_from_file(args.latest)
        else:
            latest_json = args.latest_json
        
        # Perform the comparison
        result = is_latest_release(current_json, latest_json)
        
        # Handle output based on flags
        if args.quiet:
            print(str(result['is_latest']).lower())
        elif args.verbose:
            print("Release Comparison Results:")
            print(f"  Current tag:      {result['current_tag']}")
            print(f"  Latest tag:       {result['latest_tag']}")
            print(f"  Is latest:        {result['is_latest']}")
            print(f"  Current URL:      {result['current_url']}")
            print(f"  Latest URL:       {result['latest_url']}")
            print(f"  Current published: {result['current_published_at']}")
            print(f"  Latest published:  {result['latest_published_at']}")
            print(f"  Current prerelease: {result['current_is_prerelease']}")
            print(f"  Latest prerelease:  {result['latest_is_prerelease']}")
        else:
            # Default output
            if result['is_latest']:
                print(f"✓ Current release {result['current_tag']} is the latest release")
            else:
                print(f"✗ Current release {result['current_tag']} is NOT the latest release")
                print(f"  Latest release is: {result['latest_tag']}")
        
        # Handle exit code if requested
        if args.exit_code:
            sys.exit(0 if result['is_latest'] else 1)
            
    except (FileNotFoundError, json.JSONDecodeError, ValueError) as e:
        logger.error(f"Error: {e}")
        if args.exit_code:
            sys.exit(2)
        else:
            sys.exit(1)
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        if args.exit_code:
            sys.exit(2)
        else:
            sys.exit(1)


if __name__ == "__main__":
    main()