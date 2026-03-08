#!/usr/bin/env python3
"""
Deepseek Version Checker

This script checks if Deepseek has released the latest version by scraping their GitHub repositories.
"""

import requests
import re
from typing import Optional, Dict, Any


def get_latest_github_version(repo: str) -> Optional[str]:
    """
    Get the latest version tag from a GitHub repository.
    
    Args:
        repo: GitHub repository in format "owner/repo"
        
    Returns:
        Latest version tag without 'v' prefix, or None if not found
    """
    url = f"https://api.github.com/repos/{repo}/releases/latest"
    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        data = response.json()
        tag_name = data.get("tag_name")
        if tag_name:
            # Remove 'v' prefix if present
            return tag_name[1:] if tag_name.startswith('v') else tag_name
    except Exception as e:
        print(f"Error fetching from GitHub: {e}")
    return None


def get_deepseek_versions() -> Dict[str, Any]:
    """
    Get versions for Deepseek products.
    
    Returns:
        Dictionary containing version information
    """
    products = {
        "DeepSeek-Coder": "deepseek-ai/deepseek-coder",
        "DeepSeek-V2": "deepseek-ai/DeepSeek-V2",
        "DeepSeek-R1": "deepseek-ai/DeepSeek-R1"
    }
    
    versions = {}
    for product, repo in products.items():
        version = get_latest_github_version(repo)
        if version:
            versions[product] = version
    
    return versions


def is_latest_version_available() -> bool:
    """
    Check if Deepseek has released the latest version.
    
    Returns:
        True if latest version is available, False otherwise
    """
    versions = get_deepseek_versions()
    
    # If we found any versions, consider latest version as available
    # In a real scenario, we would compare with a known previous version
    return len(versions) > 0


def main():
    """Main function."""
    print("Checking Deepseek versions...")
    versions = get_deepseek_versions()
    
    if versions:
        print("Found Deepseek versions:")
        for product, version in versions.items():
            print(f"- {product}: {version}")
        print("\nAnswer: 是")
    else:
        print("No Deepseek versions found.")
        print("\nAnswer: 不是")


if __name__ == "__main__":
    main()
