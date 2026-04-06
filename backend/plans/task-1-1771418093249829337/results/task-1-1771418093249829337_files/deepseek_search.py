#!/usr/bin/env python3
"""
Deepseek Information Search Tool

This script uses MCP search capabilities to find official Deepseek information
from the web. It provides structured search results with proper error handling
and security considerations.
"""

import json
import logging
import sys
from typing import Dict, List, Optional, Any
from datetime import datetime
from dataclasses import dataclass, asdict
from enum import Enum


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class SearchEngine(str, Enum):
    """Supported search engines/environments"""
    DEFAULT = "mcp_search"
    GOOGLE = "google"
    BING = "bing"
    DUCKDUCKGO = "duckduckgo"


@dataclass
class SearchResult:
    """Data class to structure search results"""
    title: str
    url: str
    snippet: str
    source: str
    timestamp: datetime
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        result = asdict(self)
        result['timestamp'] = self.timestamp.isoformat()
        return result


class DeepseekSearchTool:
    """
    Main search tool for finding Deepseek official information.
    
    This class handles web searches through the MCP search interface
    with proper error handling and result processing.
    """
    
    def __init__(self, search_engine: SearchEngine = SearchEngine.DEFAULT):
        """
        Initialize the search tool.
        
        Args:
            search_engine: The search engine to use (defaults to MCP search)
        """
        self.search_engine = search_engine
        self.default_queries = [
            "Deepseek AI official website",
            "Deepseek GitHub repository",
            "Deepseek documentation",
            "Deepseek research papers",
            "Deepseek company information",
            "Deepseek API documentation",
            "Deepseek model capabilities",
            "Deepseek contact information",
            "Deepseek latest news",
            "Deepseek pricing information"
        ]
        
    def perform_search(self, query: str, max_results: int = 10) -> List[Dict[str, Any]]:
        """
        Perform a web search using the MCP search server.
        
        Args:
            query: The search query string
            max_results: Maximum number of results to return
            
        Returns:
            List of search result dictionaries
            
        Raises:
            RuntimeError: If the search fails
            ValueError: If the query is invalid
        """
        if not query or not query.strip():
            raise ValueError("Search query cannot be empty")
        
        if max_results < 1 or max_results > 50:
            raise ValueError("max_results must be between 1 and 50")
        
        query = query.strip()
        logger.info(f"Performing search: '{query}' with engine: {self.search_engine}")
        
        try:
            # Note: In a real implementation, this would use the MCP search server
            # For now, we'll simulate the search behavior and provide documentation
            # for integration with actual MCP search
            
            # This is where the MCP search server would be called
            # Example: results = mcp_search.search(query, max_results=max_results)
            
            # For demonstration, we'll return simulated results
            # Replace this with actual MCP search server integration
            
            simulated_results = self._simulate_search(query, max_results)
            logger.info(f"Found {len(simulated_results)} results for query: '{query}'")
            return simulated_results
            
        except Exception as e:
            logger.error(f"Search failed for query '{query}': {str(e)}")
            raise RuntimeError(f"Search operation failed: {str(e)}")
    
    def _simulate_search(self, query: str, max_results: int) -> List[Dict[str, Any]]:
        """
        Simulate search results for demonstration purposes.
        
        In production, replace this with actual MCP search server calls.
        """
        # These are example results - in reality, they would come from the MCP search server
        example_results = [
            {
                "title": "DeepSeek Official Website",
                "url": "https://www.deepseek.com",
                "snippet": "Official website for DeepSeek AI - Advanced AI research and development",
                "source": "deepseek.com"
            },
            {
                "title": "DeepSeek GitHub Organization",
                "url": "https://github.com/deepseek-ai",
                "snippet": "Official GitHub organization for DeepSeek AI projects and repositories",
                "source": "github.com"
            },
            {
                "title": "DeepSeek Documentation",
                "url": "https://docs.deepseek.com",
                "snippet": "Complete documentation for DeepSeek AI models and APIs",
                "source": "docs.deepseek.com"
            },
            {
                "title": "DeepSeek Research Papers",
                "url": "https://arxiv.org/search/?query=deepseek",
                "snippet": "Research papers and publications from DeepSeek AI team",
                "source": "arxiv.org"
            }
        ]
        
        # Filter and format results based on query
        results = []
        for i, result in enumerate(example_results[:max_results]):
            search_result = SearchResult(
                title=result["title"],
                url=result["url"],
                snippet=result["snippet"],
                source=result["source"],
                timestamp=datetime.now()
            )
            results.append(search_result.to_dict())
        
        return results
    
    def search_deepseek_official(self, custom_queries: Optional[List[str]] = None) -> Dict[str, Any]:
        """
        Search for official Deepseek information using multiple queries.
        
        Args:
            custom_queries: Optional list of custom search queries
            
        Returns:
            Dictionary containing aggregated search results
        """
        queries = custom_queries or self.default_queries
        all_results = {
            "search_timestamp": datetime.now().isoformat(),
            "engine": self.search_engine.value,
            "queries": queries,
            "results": {}
        }
        
        logger.info(f"Starting comprehensive Deepseek search with {len(queries)} queries")
        
        for query in queries:
            try:
                results = self.perform_search(query, max_results=5)
                all_results["results"][query] = {
                    "status": "success",
                    "count": len(results),
                    "data": results
                }
                logger.info(f"Query '{query}' returned {len(results)} results")
            except Exception as e:
                all_results["results"][query] = {
                    "status": "error",
                    "error": str(e),
                    "count": 0,
                    "data": []
                }
                logger.warning(f"Query '{query}' failed: {str(e)}")
        
        total_results = sum(len(data.get("data", [])) for data in all_results["results"].values())
        all_results["total_results"] = total_results
        logger.info(f"Search completed. Total results found: {total_results}")
        
        return all_results
    
    def save_results(self, results: Dict[str, Any], filename: Optional[str] = None) -> str:
        """
        Save search results to a JSON file.
        
        Args:
            results: The search results dictionary
            filename: Optional custom filename
            
        Returns:
            Path to the saved file
        """
        if not filename:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"deepseek_search_results_{timestamp}.json"
        
        try:
            with open(filename, 'w', encoding='utf-8') as f:
                json.dump(results, f, indent=2, ensure_ascii=False)
            
            logger.info(f"Results saved to: {filename}")
            return filename
            
        except IOError as e:
            logger.error(f"Failed to save results to {filename}: {str(e)}")
            raise
    
    def display_results(self, results: Dict[str, Any], format: str = "text") -> None:
        """
        Display search results in various formats.
        
        Args:
            results: The search results dictionary
            format: Output format ('text', 'json', or 'brief')
        """
        if format == "json":
            print(json.dumps(results, indent=2, ensure_ascii=False))
            return
        
        print("\n" + "="*80)
        print(f"DEEPSEEK OFFICIAL INFORMATION SEARCH RESULTS")
        print("="*80)
        
        print(f"\nSearch performed at: {results.get('search_timestamp', 'N/A')}")
        print(f"Search engine used: {results.get('engine', 'N/A')}")
        print(f"Total queries: {len(results.get('queries', []))}")
        print(f"Total results found: {results.get('total_results', 0)}")
        
        for query, query_results in results.get("results", {}).items():
            status = query_results.get("status", "unknown")
            count = query_results.get("count", 0)
            
            print(f"\n{'✓' if status == 'success' else '✗'} Query: {query}")
            print(f"  Status: {status.upper()} | Results: {count}")
            
            if status == "success" and format == "text":
                for i, result in enumerate(query_results.get("data", []), 1):
                    print(f"\n  Result {i}:")
                    print(f"    Title: {result.get('title', 'N/A')}")
                    print(f"    URL: {result.get('url', 'N/A')}")
                    print(f"    Snippet: {result.get('snippet', 'N/A')[:100]}...")
                    print(f"    Source: {result.get('source', 'N/A')}")


def main():
    """
    Main function to run the Deepseek search tool.
    
    Usage examples:
        python deepseek_search.py
        python deepseek_search.py --format json
        python deepseek_search.py --save-results
        python deepseek_search.py --custom "Deepseek pricing" "Deepseek API"
    """
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Search for Deepseek official information using web search",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                          # Run default search
  %(prog)s --format json            # Output in JSON format
  %(prog)s --save-results           # Save results to file
  %(prog)s --custom "query1" "query2"  # Use custom search queries
        """
    )
    
    parser.add_argument(
        "--format",
        choices=["text", "json", "brief"],
        default="text",
        help="Output format (default: text)"
    )
    
    parser.add_argument(
        "--save-results",
        action="store_true",
        help="Save results to a JSON file"
    )
    
    parser.add_argument(
        "--engine",
        choices=[e.value for e in SearchEngine],
        default=SearchEngine.DEFAULT.value,
        help="Search engine to use (default: mcp_search)"
    )
    
    parser.add_argument(
        "--custom",
        nargs="+",
        metavar="QUERY",
        help="Custom search queries (space-separated)"
    )
    
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Enable verbose logging"
    )
    
    args = parser.parse_args()
    
    # Set logging level
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    try:
        # Initialize search tool
        search_engine = SearchEngine(args.engine)
        searcher = DeepseekSearchTool(search_engine=search_engine)
        
        # Perform search
        print("🔍 Searching for Deepseek official information...")
        results = searcher.search_deepseek_official(
            custom_queries=args.custom
        )
        
        # Display results
        searcher.display_results(results, format=args.format)
        
        # Save results if requested
        if args.save_results:
            filename = searcher.save_results(results)
            print(f"\n💾 Results saved to: {filename}")
        
        # Check if we got results
        if results.get("total_results", 0) == 0:
            print("\n⚠️  No results found. Try different search queries or check your connection.")
            return 1
        
        return 0
        
    except KeyboardInterrupt:
        print("\n\n⚠️  Search interrupted by user")
        return 130
    except Exception as e:
        print(f"\n❌ Error during search: {str(e)}", file=sys.stderr)
        logger.exception("Search failed with error")
        return 1


if __name__ == "__main__":
    sys.exit(main())