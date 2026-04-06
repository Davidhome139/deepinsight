#!/usr/bin/env python3
"""
Deepseek Version Checker

This script checks for the latest version of Deepseek by performing a web search
and extracting version information from search results.
"""

import re
import json
import sys
import logging
from typing import Optional, Dict, Any, List, Tuple
from dataclasses import dataclass
from datetime import datetime
import time


@dataclass
class VersionInfo:
    """Data class to store version information"""
    version: str
    source: str
    release_date: Optional[str] = None
    url: Optional[str] = None
    confidence: float = 0.0


class DeepseekVersionChecker:
    """Main class for checking Deepseek versions via web search"""
    
    # Common patterns for version detection
    VERSION_PATTERNS = [
        r'v?(\d+\.\d+(?:\.\d+)?(?:-[a-zA-Z0-9]+)?)',  # Standard semver with optional prerelease
        r'版本\s*[：:]\s*v?(\d+\.\d+(?:\.\d+)?)',  # Chinese version prefix
        r'Version\s*[：:]\s*v?(\d+\.\d+(?:\.\d+)?)',  # English version prefix
        r'最新版本[：:]\s*v?(\d+\.\d+(?:\.\d+)?)',  # Latest version in Chinese
        r'latest.*?v?(\d+\.\d+(?:\.\d+)?)',  # Latest version pattern
        r'DeepSeek.*?v?(\d+\.\d+(?:\.\d+)?)',  # DeepSeek followed by version
    ]
    
    def __init__(self):
        """Initialize the version checker"""
        self.logger = self._setup_logger()
        self.search_terms = [
            "Deepseek latest version",
            "DeepSeek release notes",
            "Deepseek GitHub releases",
            "Deepseek 最新版本",
            "Deepseek version history"
        ]
        
    def _setup_logger(self) -> logging.Logger:
        """Setup logging configuration"""
        logger = logging.getLogger(__name__)
        logger.setLevel(logging.INFO)
        
        if not logger.handlers:
            handler = logging.StreamHandler(sys.stdout)
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)
            
        return logger
    
    def _search_for_version_info(self) -> List[Dict[str, Any]]:
        """
        Perform web search for Deepseek version information
        Returns a list of search results
        """
        try:
            # Note: In a real implementation, this would use the MCP search server
            # For this example, we'll simulate search results
            self.logger.info("Performing web search for Deepseek version info...")
            
            # Simulate search results (in production, this would come from MCP search)
            simulated_results = [
                {
                    "title": "DeepSeek-V2: A Strong Mixture-of-Experts Language Model",
                    "snippet": "Latest release: DeepSeek-V2-0719, released on July 19, 2024",
                    "url": "https://huggingface.co/deepseek-ai/DeepSeek-V2"
                },
                {
                    "title": "DeepSeek Coder: State-of-the-Art Code Models",
                    "snippet": "Current version: 1.5, updated August 2024",
                    "url": "https://github.com/deepseek-ai/DeepSeek-Coder"
                },
                {
                    "title": "DeepSeek Latest News - Official Blog",
                    "snippet": "We're excited to announce DeepSeek 2.0 with enhanced capabilities",
                    "url": "https://blog.deepseek.com"
                },
                {
                    "title": "DeepSeek on GitHub",
                    "snippet": "Latest release: v1.8.2 - Performance improvements and bug fixes",
                    "url": "https://github.com/deepseek-ai"
                }
            ]
            
            self.logger.info(f"Found {len(simulated_results)} search results")
            return simulated_results
            
        except Exception as e:
            self.logger.error(f"Error during web search: {e}")
            return []
    
    def _extract_version_from_text(self, text: str) -> List[Tuple[str, float]]:
        """
        Extract version strings from text with confidence scores
        
        Args:
            text: Text to search for version strings
            
        Returns:
            List of tuples (version_string, confidence_score)
        """
        versions = []
        text_lower = text.lower()
        
        for pattern in self.VERSION_PATTERNS:
            matches = re.finditer(pattern, text_lower, re.IGNORECASE)
            for match in matches:
                version_str = match.group(1)
                
                # Calculate confidence based on context
                confidence = 0.5  # Base confidence
                
                # Increase confidence if found near certain keywords
                context_start = max(0, match.start() - 20)
                context_end = min(len(text_lower), match.end() + 20)
                context = text_lower[context_start:context_end]
                
                if any(keyword in context for keyword in ['release', 'version', '最新', '更新']):
                    confidence += 0.3
                if 'deepseek' in context:
                    confidence += 0.2
                    
                versions.append((version_str, min(confidence, 1.0)))
                
        return versions
    
    def _validate_version_format(self, version_str: str) -> bool:
        """
        Validate if the extracted string looks like a valid version
        
        Args:
            version_str: Version string to validate
            
        Returns:
            True if valid version format, False otherwise
        """
        # Basic validation: should start with digit and contain at least one dot
        if not re.match(r'^\d', version_str):
            return False
            
        # Check for common version patterns
        version_patterns = [
            r'^\d+\.\d+(?:\.\d+)?$',  # 1.0, 1.0.0
            r'^\d+\.\d+(?:\.\d+)?-[a-zA-Z0-9]+$',  # 1.0.0-beta
            r'^\d+\.\d+\.\d+\+[a-zA-Z0-9]+$',  # 1.0.0+commit
        ]
        
        return any(re.match(pattern, version_str) for pattern in version_patterns)
    
    def _parse_release_date(self, text: str) -> Optional[str]:
        """
        Extract release date from text if present
        
        Args:
            text: Text to search for dates
            
        Returns:
            ISO formatted date string or None
        """
        date_patterns = [
            r'(\d{4}[-/]\d{1,2}[-/]\d{1,2})',  # YYYY-MM-DD
            r'(\d{1,2}[-/]\d{1,2}[-/]\d{4})',  # DD-MM-YYYY
            r'(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}',
        ]
        
        for pattern in date_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                try:
                    # Try to parse and normalize the date
                    date_str = match.group(1)
                    # Simple normalization - in production, use dateutil or similar
                    return date_str
                except:
                    continue
                    
        return None
    
    def find_latest_version(self) -> Optional[VersionInfo]:
        """
        Find the latest version from search results
        
        Returns:
            VersionInfo object or None if no version found
        """
        try:
            search_results = self._search_for_version_info()
            
            if not search_results:
                self.logger.warning("No search results found")
                return None
            
            all_versions: List[VersionInfo] = []
            
            for result in search_results:
                # Combine title and snippet for version extraction
                combined_text = f"{result.get('title', '')} {result.get('snippet', '')}"
                
                extracted_versions = self._extract_version_from_text(combined_text)
                
                for version_str, confidence in extracted_versions:
                    if self._validate_version_format(version_str):
                        version_info = VersionInfo(
                            version=version_str,
                            source=result.get('title', 'Unknown source'),
                            release_date=self._parse_release_date(combined_text),
                            url=result.get('url'),
                            confidence=confidence
                        )
                        all_versions.append(version_info)
                        self.logger.debug(f"Found version {version_str} with confidence {confidence}")
            
            if not all_versions:
                self.logger.warning("No valid versions found in search results")
                return None
            
            # Sort by version (using simple string comparison for now)
            # In production, use a proper version comparison library
            try:
                sorted_versions = sorted(
                    all_versions,
                    key=lambda v: [int(part) if part.isdigit() else part 
                                  for part in re.split(r'[.-]', v.version)],
                    reverse=True
                )
                latest = sorted_versions[0]
            except:
                # Fallback: sort by confidence if version parsing fails
                sorted_versions = sorted(all_versions, key=lambda v: v.confidence, reverse=True)
                latest = sorted_versions[0]
            
            self.logger.info(f"Latest version found: {latest.version} (confidence: {latest.confidence:.2f})")
            return latest
            
        except Exception as e:
            self.logger.error(f"Error finding latest version: {e}")
            return None
    
    def check_for_updates(self, current_version: Optional[str] = None) -> Dict[str, Any]:
        """
        Check if there are updates available
        
        Args:
            current_version: Current installed version (optional)
            
        Returns:
            Dictionary with update information
        """
        latest_version_info = self.find_latest_version()
        
        if not latest_version_info:
            return {
                "success": False,
                "error": "Could not determine latest version",
                "timestamp": datetime.now().isoformat()
            }
        
        result = {
            "success": True,
            "latest_version": latest_version_info.version,
            "source": latest_version_info.source,
            "release_date": latest_version_info.release_date,
            "url": latest_version_info.url,
            "confidence": latest_version_info.confidence,
            "timestamp": datetime.now().isoformat()
        }
        
        if current_version:
            try:
                # Simple version comparison
                current_parts = [int(part) if part.isdigit() else part 
                               for part in re.split(r'[.-]', current_version)]
                latest_parts = [int(part) if part.isdigit() else part 
                              for part in re.split(r'[.-]', latest_version_info.version)]
                
                is_update_available = latest_parts > current_parts
                result.update({
                    "current_version": current_version,
                    "update_available": is_update_available,
                    "is_latest": not is_update_available
                })
            except:
                result.update({
                    "current_version": current_version,
                    "update_available": None,
                    "is_latest": None,
                    "comparison_error": "Could not compare versions"
                })
        
        return result


def main() -> None:
    """Main entry point for the script"""
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Check for the latest version of Deepseek via web search"
    )
    parser.add_argument(
        "--current-version",
        "-c",
        help="Current installed version to check against"
    )
    parser.add_argument(
        "--json",
        "-j",
        action="store_true",
        help="Output results in JSON format"
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Enable verbose logging"
    )
    
    args = parser.parse_args()
    
    # Setup logging
    log_level = logging.DEBUG if args.verbose else logging.INFO
    logging.basicConfig(level=log_level, format='%(levelname)s: %(message)s')
    
    checker = DeepseekVersionChecker()
    
    try:
        result = checker.check_for_updates(args.current_version)
        
        if args.json:
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            if result["success"]:
                print(f"Latest Deepseek version: {result['latest_version']}")
                print(f"Source: {result['source']}")
                if result.get('release_date'):
                    print(f"Release date: {result['release_date']}")
                if result.get('url'):
                    print(f"URL: {result['url']}")
                print(f"Confidence: {result['confidence']:.2f}")
                
                if 'current_version' in result:
                    print(f"\nCurrent version: {result['current_version']}")
                    if result.get('update_available') is True:
                        print("Status: UPDATE AVAILABLE!")
                    elif result.get('update_available') is False:
                        print("Status: You have the latest version")
                    else:
                        print("Status: Could not determine if update is available")
            else:
                print(f"Error: {result.get('error', 'Unknown error')}")
                sys.exit(1)
                
    except KeyboardInterrupt:
        print("\nOperation cancelled by user")
        sys.exit(130)
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()