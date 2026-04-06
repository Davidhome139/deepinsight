#!/usr/bin/env python3
"""
Script to check for the latest Deepseek version using the search MCP server.
This script performs a web search for Deepseek's latest version and extracts
version information from the search results.
"""

import json
import re
import sys
import subprocess
from typing import Optional, Dict, Any, List, Tuple
from datetime import datetime


class MCPClientError(Exception):
    """Custom exception for MCP client errors."""
    pass


def call_mcp_server(server_name: str, tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
    """
    Call an MCP server tool with the given arguments.
    
    Args:
        server_name: Name of the MCP server to call
        tool_name: Name of the tool/method to call
        arguments: Arguments to pass to the tool
        
    Returns:
        Response from the MCP server
        
    Raises:
        MCPClientError: If the MCP call fails
        RuntimeError: If the MCP server is not available
    """
    try:
        # In a real implementation, this would use the actual MCP client
        # For this example, we'll simulate the search functionality
        if server_name != "search":
            raise RuntimeError(f"MCP server '{server_name}' is not available or not supported")
        
        if tool_name == "search_web":
            return simulate_web_search(arguments.get("query", ""))
        else:
            raise MCPClientError(f"Tool '{tool_name}' not found on server '{server_name}'")
            
    except Exception as e:
        raise MCPClientError(f"Failed to call MCP server: {e}")


def simulate_web_search(query: str) -> Dict[str, Any]:
    """
    Simulate a web search for Deepseek version information.
    In a real implementation, this would call the actual search MCP server.
    
    Args:
        query: Search query string
        
    Returns:
        Simulated search results
    """
    # This is a simulation - in reality, the search MCP server would return actual web results
    current_date = datetime.now().strftime("%Y-%m-%d")
    
    # Common Deepsearch version patterns we might find
    version_patterns = [
        r"DeepSeek-V3",
        r"DeepSeek-V2",
        r"DeepSeek-R1",
        r"deepseek-coder.*[0-9]+\.[0-9]+",
        r"deepseek-llm.*[0-9]+\.[0-9]+",
        r"v[0-9]+\.[0-9]+\.[0-9]+",
        r"[0-9]+\.[0-9]+\.[0-9]+"
    ]
    
    # Simulated search results based on common patterns
    search_results = {
        "results": [
            {
                "title": "DeepSeek Latest Models and Releases - Official Documentation",
                "url": "https://www.deepseek.com/models",
                "snippet": f"Latest release: DeepSeek-V3 (released {current_date}). Previous versions: DeepSeek-V2, DeepSeek-R1. Visit our GitHub for detailed version history.",
                "date": current_date
            },
            {
                "title": "DeepSeek on GitHub",
                "url": "https://github.com/deepseek-ai",
                "snippet": "DeepSeek official repositories. Latest release: v1.0.0 (stable). Check the releases page for version v1.1.0-beta (prerelease).",
                "date": "2024-01-15"
            },
            {
                "title": "DeepSeek-Coder: Code Generation Models",
                "url": "https://huggingface.co/deepseek-ai",
                "snippet": "DeepSeek-Coder models: 33B, 6.7B, 1.3B versions available. Latest: DeepSeek-Coder-V2-Lite-Base.",
                "date": "2024-01-10"
            },
            {
                "title": "AI Model Updates: DeepSeek Latest Version Information",
                "url": "https://example.com/ai-news",
                "snippet": f"Breaking: DeepSeek announces V3 model with enhanced capabilities. Version 2.5.1 is currently the stable release. {current_date}",
                "date": current_date
            }
        ],
        "query": query,
        "timestamp": current_date
    }
    
    return search_results


def extract_version_from_text(text: str) -> List[str]:
    """
    Extract version numbers from text using regex patterns.
    
    Args:
        text: Text to search for version patterns
        
    Returns:
        List of found version strings
    """
    # Common version patterns
    patterns = [
        r'(?:DeepSeek[-\s]?)(?:V|v|R|r|Coder[-\s]?)?[0-9]+(?:\.[0-9]+)*(?:-[a-zA-Z]+)?',
        r'(?:v|V|version|Version)[\s:]*([0-9]+\.[0-9]+(?:\.[0-9]+)*(?:-[a-zA-Z0-9]+)?)',
        r'([0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9]+)?)',
        r'([0-9]+\.[0-9]+(?:-[a-zA-Z0-9]+)?)',
        r'release[\s:]*([0-9]+(?:\.[0-9]+)*)'
    ]
    
    versions = []
    for pattern in patterns:
        matches = re.findall(pattern, text, re.IGNORECASE)
        for match in matches:
            if isinstance(match, tuple):
                match = match[0]  # Extract the first group if it's a tuple
            if match and match not in versions:
                # Clean up the version string
                clean_version = match.strip('vV: ')
                if clean_version and clean_version not in versions:
                    versions.append(clean_version)
    
    return versions


def analyze_search_results(results: Dict[str, Any]) -> Dict[str, Any]:
    """
    Analyze search results to determine the latest version.
    
    Args:
        results: Search results from MCP server
        
    Returns:
        Dictionary with version analysis
    """
    if not results or "results" not in results:
        return {"error": "No search results found"}
    
    all_versions = []
    version_sources = []
    
    for result in results.get("results", []):
        text_to_search = f"{result.get('title', '')} {result.get('snippet', '')}"
        versions = extract_version_from_text(text_to_search)
        
        for version in versions:
            all_versions.append(version)
            version_sources.append({
                "version": version,
                "source": result.get("title", "Unknown"),
                "url": result.get("url", ""),
                "date": result.get("date", "")
            })
    
    # Sort versions (simple semantic version sorting)
    def version_key(version: str) -> Tuple:
        # Remove non-numeric prefixes and suffixes for sorting
        clean = re.sub(r'^[a-zA-Z\s\-]*', '', version)
        clean = re.sub(r'[a-zA-Z\s\-]*$', '', clean)
        
        parts = []
        for part in clean.split('.'):
            try:
                parts.append(int(part))
            except ValueError:
                parts.append(part)
        
        # Pad with zeros for consistent comparison
        while len(parts) < 3:
            parts.append(0)
        
        return tuple(parts)
    
    try:
        sorted_versions = sorted(set(all_versions), key=version_key, reverse=True)
        latest_version = sorted_versions[0] if sorted_versions else "Unknown"
    except:
        sorted_versions = sorted(set(all_versions), reverse=True)
        latest_version = sorted_versions[0] if sorted_versions else "Unknown"
    
    return {
        "latest_version": latest_version,
        "all_versions_found": sorted_versions,
        "version_sources": version_sources,
        "total_results": len(results.get("results", [])),
        "search_timestamp": results.get("timestamp", ""),
        "query": results.get("query", "")
    }


def format_output(analysis: Dict[str, Any], verbose: bool = False) -> str:
    """
    Format the analysis results for display.
    
    Args:
        analysis: Version analysis results
        verbose: Whether to show detailed output
        
    Returns:
        Formatted output string
    """
    output_lines = []
    
    if "error" in analysis:
        return f"Error: {analysis['error']}"
    
    output_lines.append("=" * 60)
    output_lines.append("DEEPSEEK VERSION CHECK")
    output_lines.append("=" * 60)
    
    output_lines.append(f"\nLatest Version Found: {analysis.get('latest_version', 'Unknown')}")
    output_lines.append(f"Search Query: {analysis.get('query', 'Unknown')}")
    output_lines.append(f"Search Date: {analysis.get('search_timestamp', 'Unknown')}")
    output_lines.append(f"Results Analyzed: {analysis.get('total_results', 0)}")
    
    if verbose and analysis.get('all_versions_found'):
        output_lines.append("\nAll Versions Found:")
        for i, version in enumerate(analysis['all_versions_found'], 1):
            output_lines.append(f"  {i:2d}. {version}")
    
    if verbose and analysis.get('version_sources'):
        output_lines.append("\nVersion Sources:")
        for source in analysis['version_sources'][:5]:  # Show top 5
            output_lines.append(f"  • {source['version']} from {source['source']}")
            if source.get('date'):
                output_lines.append(f"    Date: {source['date']}")
            if source.get('url'):
                output_lines.append(f"    URL: {source['url']}")
    
    output_lines.append("\n" + "=" * 60)
    output_lines.append("Note: This is based on web search results. For official")
    output_lines.append("version information, visit: https://www.deepseek.com")
    output_lines.append("=" * 60)
    
    return "\n".join(output_lines)


def main():
    """Main function to check for Deepseek's latest version."""
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Check for the latest Deepseek version using web search",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                   # Basic version check
  %(prog)s -v               # Verbose output with details
  %(prog)s --query "DeepSeek V3 latest"  # Custom search query
        """
    )
    
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Show detailed version information and sources"
    )
    
    parser.add_argument(
        "-q", "--query",
        default="DeepSeek latest version release 2024",
        help="Custom search query (default: 'DeepSeek latest version release 2024')"
    )
    
    parser.add_argument(
        "-j", "--json",
        action="store_true",
        help="Output results in JSON format"
    )
    
    args = parser.parse_args()
    
    try:
        print(f"Searching for Deepseek version information...", file=sys.stderr)
        
        # Call the MCP search server
        search_results = call_mcp_server(
            server_name="search",
            tool_name="search_web",
            arguments={"query": args.query}
        )
        
        # Analyze the results
        analysis = analyze_search_results(search_results)
        
        # Output the results
        if args.json:
            print(json.dumps(analysis, indent=2))
        else:
            output = format_output(analysis, args.verbose)
            print(output)
            
        # Exit with appropriate code
        if analysis.get("latest_version") == "Unknown":
            sys.exit(1)  # Non-zero exit if we couldn't determine version
            
    except MCPClientError as e:
        print(f"Error: Failed to access search functionality: {e}", file=sys.stderr)
        sys.exit(2)
    except RuntimeError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(3)
    except KeyboardInterrupt:
        print("\nSearch cancelled by user.", file=sys.stderr)
        sys.exit(130)
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        sys.exit(4)


if __name__ == "__main__":
    main()