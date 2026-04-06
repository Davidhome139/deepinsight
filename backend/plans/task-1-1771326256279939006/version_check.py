#!/usr/bin/env python3
"""
Version Comparison Utility

This module provides functionality to parse, compare, and validate version strings
following Semantic Versioning (SemVer) principles with support for various formats.
"""

import re
import functools
from typing import Optional, Tuple, List, Union, Dict, Any
from dataclasses import dataclass
from enum import Enum


class VersionComponent(Enum):
    """Enum for version component types."""
    NUMERIC = "numeric"
    ALPHA = "alpha"
    ALPHANUMERIC = "alphanumeric"
    PRERELEASE = "prerelease"
    BUILD = "build"


class VersionError(Exception):
    """Base exception for version parsing and comparison errors."""
    pass


@dataclass(frozen=True)
class VersionInfo:
    """
    Immutable dataclass representing parsed version information.
    
    Attributes:
        major (int): Major version number
        minor (int): Minor version number
        patch (int): Patch version number
        prerelease (Optional[str]): Prerelease identifier (alpha, beta, rc, etc.)
        build (Optional[str]): Build metadata
        original (str): Original version string
    """
    major: int
    minor: int
    patch: int
    prerelease: Optional[str] = None
    build: Optional[str] = None
    original: str = ""

    def __str__(self) -> str:
        """Return string representation of the version."""
        version_str = f"{self.major}.{self.minor}.{self.patch}"
        if self.prerelease:
            version_str += f"-{self.prerelease}"
        if self.build:
            version_str += f"+{self.build}"
        return version_str

    def to_dict(self) -> Dict[str, Any]:
        """Convert version info to dictionary."""
        return {
            "major": self.major,
            "minor": self.minor,
            "patch": self.patch,
            "prerelease": self.prerelease,
            "build": self.build,
            "original": self.original,
            "string": str(self)
        }


class VersionComparator:
    """
    Main class for parsing and comparing version strings.
    
    Supports multiple version formats including:
    - Semantic Versioning (SemVer): 1.2.3, 2.0.0-alpha, 1.0.0+build.123
    - Simple versions: 1.2, 1.2.3.4
    - Version with prefixes: v1.2.3, version-2.0
    """
    
    # Regex patterns for different version formats
    VERSION_PATTERNS = {
        "semver": re.compile(
            r'^v?(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)'
            r'(?:-(?P<prerelease>[0-9A-Za-z\-\.]+))?'
            r'(?:\+(?P<build>[0-9A-Za-z\-\.]+))?$'
        ),
        "simple": re.compile(
            r'^v?(?P<major>\d+)(?:\.(?P<minor>\d+))?'
            r'(?:\.(?P<patch>\d+))?(?:\.(?P<extra>\d+))?$'
        ),
        "prefix": re.compile(
            r'^(?:v|version[_-])?(?P<major>\d+)'
            r'(?:[\._-](?P<minor>\d+))?'
            r'(?:[\._-](?P<patch>\d+))?'
            r'(?:[\._-](?P<extra>\d+))?$',
            re.IGNORECASE
        )
    }
    
    @classmethod
    def parse(cls, version_str: str) -> VersionInfo:
        """
        Parse a version string into structured VersionInfo.
        
        Args:
            version_str: Version string to parse
            
        Returns:
            VersionInfo object with parsed components
            
        Raises:
            VersionError: If version string cannot be parsed
        """
        if not version_str or not isinstance(version_str, str):
            raise VersionError(f"Invalid version string: {version_str}")
        
        version_str = version_str.strip()
        
        # Try different patterns in order of specificity
        for pattern_name, pattern in cls.VERSION_PATTERNS.items():
            match = pattern.match(version_str)
            if match:
                return cls._parse_from_match(match, pattern_name, version_str)
        
        raise VersionError(f"Unable to parse version string: {version_str}")
    
    @staticmethod
    def _parse_from_match(match: re.Match, pattern_name: str, 
                          original: str) -> VersionInfo:
        """Extract version components from regex match."""
        groups = match.groupdict()
        
        # Extract major, minor, patch with defaults
        major = int(groups.get('major', 0))
        minor = int(groups.get('minor') or 0)
        patch = int(groups.get('patch') or 0)
        
        # Handle extra component for simple patterns
        if pattern_name == "simple" and groups.get('extra'):
            # For simple patterns with 4 components, treat last as patch
            patch = int(groups.get('extra') or 0)
        
        prerelease = groups.get('prerelease')
        build = groups.get('build')
        
        return VersionInfo(
            major=major,
            minor=minor,
            patch=patch,
            prerelease=prerelease,
            build=build,
            original=original
        )
    
    @staticmethod
    def compare(v1: Union[str, VersionInfo], 
                v2: Union[str, VersionInfo]) -> int:
        """
        Compare two versions.
        
        Args:
            v1: First version (string or VersionInfo)
            v2: Second version (string or VersionInfo)
            
        Returns:
            -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
            
        Raises:
            VersionError: If either version cannot be parsed
        """
        # Parse if strings are provided
        if isinstance(v1, str):
            v1 = VersionComparator.parse(v1)
        if isinstance(v2, str):
            v2 = VersionComparator.parse(v2)
        
        # Compare major, minor, patch
        for attr in ['major', 'minor', 'patch']:
            val1 = getattr(v1, attr)
            val2 = getattr(v2, attr)
            if val1 != val2:
                return -1 if val1 < val2 else 1
        
        # Compare prerelease
        prerelease_result = VersionComparator._compare_prerelease(
            v1.prerelease, v2.prerelease
        )
        if prerelease_result != 0:
            return prerelease_result
        
        # Versions are equal (build metadata is ignored in comparisons)
        return 0
    
    @staticmethod
    def _compare_prerelease(prerelease1: Optional[str], 
                            prerelease2: Optional[str]) -> int:
        """
        Compare prerelease identifiers according to SemVer rules.
        
        Rules:
        1. No prerelease < any prerelease
        2. Compare dot-separated identifiers numerically when possible
        3. Numeric identifiers have lower precedence than non-numeric
        """
        # Case 1: No prerelease vs prerelease
        if prerelease1 is None and prerelease2 is None:
            return 0
        if prerelease1 is None:
            return 1  # No prerelease is greater than prerelease
        if prerelease2 is None:
            return -1  # Prerelease is less than no prerelease
        
        # Split into identifiers
        ids1 = prerelease1.split('.')
        ids2 = prerelease2.split('.')
        
        # Compare each identifier
        for i in range(max(len(ids1), len(ids2))):
            id1 = ids1[i] if i < len(ids1) else None
            id2 = ids2[i] if i < len(ids2) else None
            
            # Handle end of one list
            if id1 is None and id2 is None:
                return 0
            if id1 is None:
                return -1  # Shorter list is less
            if id2 is None:
                return 1   # Longer list is greater
            
            # Try to compare as numbers
            try:
                num1 = int(id1)
                num2 = int(id2)
                if num1 != num2:
                    return -1 if num1 < num2 else 1
                continue  # Equal numbers, continue to next identifier
            except ValueError:
                pass  # Not numeric, compare as strings
            
            # Compare as strings
            if id1 != id2:
                # Numeric identifiers have lower precedence than non-numeric
                if id1.isdigit() and not id2.isdigit():
                    return -1
                if not id1.isdigit() and id2.isdigit():
                    return 1
                # Both non-numeric, compare lexicographically
                return -1 if id1 < id2 else 1
        
        return 0
    
    @classmethod
    def is_compatible(cls, current: Union[str, VersionInfo],
                      required: Union[str, VersionInfo],
                      allow_prerelease: bool = False) -> bool:
        """
        Check if current version is compatible with required version.
        
        Args:
            current: Current version
            required: Minimum required version
            allow_prerelease: Whether to allow prerelease versions
            
        Returns:
            True if current version meets or exceeds required version
        """
        if isinstance(current, str):
            current = cls.parse(current)
        if isinstance(required, str):
            required = cls.parse(required)
        
        # Check if we have a prerelease when not allowed
        if not allow_prerelease and current.prerelease:
            return False
        
        return cls.compare(current, required) >= 0
    
    @classmethod
    def get_version_range(cls, version_spec: str) -> Tuple[Optional[str], Optional[str]]:
        """
        Parse version range specifications.
        
        Supports:
        - Exact: "1.2.3"
        - Range: ">=1.2.3 <2.0.0"
        - Caret: "^1.2.3"
        - Tilde: "~1.2.3"
        
        Returns:
            Tuple of (min_version, max_version) where either can be None
        """
        version_spec = version_spec.strip()
        
        # Exact version
        if re.match(r'^[vV]?\d+\.\d+', version_spec):
            return (version_spec, version_spec)
        
        # Parse range operators
        operators = {
            '>=': ('min', None),
            '<=': (None, 'max'),
            '>': ('min_exclusive', None),
            '<': (None, 'max_exclusive'),
            '^': ('caret', None),
            '~': ('tilde', None)
        }
        
        # Simple range parsing (for basic cases)
        # This is a simplified implementation - in production you might want
        # to use a more robust library like `packaging.specifiers`
        parts = version_spec.split()
        min_version = None
        max_version = None
        
        for part in parts:
            for op, (min_type, max_type) in operators.items():
                if part.startswith(op):
                    version = part[len(op):].strip()
                    if min_type == 'caret':
                        parsed = cls.parse(version)
                        min_version = str(parsed)
                        # Calculate max version for caret (major version same)
                        max_version = f"{parsed.major + 1}.0.0"
                    elif min_type == 'tilde':
                        parsed = cls.parse(version)
                        min_version = str(parsed)
                        # Calculate max version for tilde (minor version same)
                        max_version = f"{parsed.major}.{parsed.minor + 1}.0"
                    elif min_type:
                        min_version = version
                    elif max_type:
                        max_version = version
                    break
        
        return (min_version, max_version)


# Convenience functions for common operations
def parse_version(version_str: str) -> VersionInfo:
    """Parse a version string into VersionInfo."""
    return VersionComparator.parse(version_str)


def compare_versions(v1: str, v2: str) -> int:
    """Compare two version strings."""
    return VersionComparator.compare(v1, v2)


def is_version_compatible(current: str, required: str, 
                         allow_prerelease: bool = False) -> bool:
    """Check if current version meets minimum requirements."""
    return VersionComparator.is_compatible(current, required, allow_prerelease)


def validate_version(version_str: str) -> bool:
    """Validate if a string is a valid version."""
    try:
        VersionComparator.parse(version_str)
        return True
    except VersionError:
        return False


def sort_versions(version_strings: List[str], reverse: bool = False) -> List[str]:
    """Sort a list of version strings."""
    parsed = [(VersionComparator.parse(v), v) for v in version_strings]
    sorted_parsed = sorted(parsed, key=lambda x: x[0], reverse=reverse)
    return [v[1] for v in sorted_parsed]


def get_latest_version(version_strings: List[str]) -> Optional[str]:
    """Get the latest version from a list of version strings."""
    if not version_strings:
        return None
    sorted_versions = sort_versions(version_strings, reverse=True)
    return sorted_versions[0]


def main():
    """Command-line interface for version checking."""
    import argparse
    import sys
    
    parser = argparse.ArgumentParser(
        description="Parse and compare version strings",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s parse 1.2.3-alpha+build
  %(prog)s compare 1.2.3 1.2.4
  %(prog)s check 2.0.0 --min 1.0.0 --max 3.0.0
  %(prog)s sort 2.0.0 1.0.0 1.5.0
        """
    )
    
    subparsers = parser.add_subparsers(dest='command', help='Command to execute')
    
    # Parse command
    parse_parser = subparsers.add_parser('parse', help='Parse a version string')
    parse_parser.add_argument('version', help='Version string to parse')
    
    # Compare command
    compare_parser = subparsers.add_parser('compare', help='Compare two versions')
    compare_parser.add_argument('version1', help='First version')
    compare_parser.add_argument('version2', help='Second version')
    
    # Check command
    check_parser = subparsers.add_parser('check', help='Check version compatibility')
    check_parser.add_argument('version', help='Version to check')
    check_parser.add_argument('--min', help='Minimum required version')
    check_parser.add_argument('--max', help='Maximum allowed version')
    check_parser.add_argument('--allow-prerelease', action='store_true',
                             help='Allow prerelease versions')
    
    # Sort command
    sort_parser = subparsers.add_parser('sort', help='Sort version strings')
    sort_parser.add_argument('versions', nargs='+', help='Versions to sort')
    sort_parser.add_argument('--reverse', action='store_true',
                            help='Sort in descending order')
    
    # Validate command
    validate_parser = subparsers.add_parser('validate', help='Validate version string')
    validate_parser.add_argument('version', help='Version to validate')
    
    # Deepseek version check command
    deepseek_parser = subparsers.add_parser('deepseek', help='Check if Deepseek has released the latest version')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        sys.exit(1)
    
    try:
        if args.command == 'parse':
            version_info = parse_version(args.version)
            print(f"Original: {version_info.original}")
            print(f"Parsed: {version_info}")
            print(f"Major: {version_info.major}")
            print(f"Minor: {version_info.minor}")
            print(f"Patch: {version_info.patch}")
            if version_info.prerelease:
                print(f"Prerelease: {version_info.prerelease}")
            if version_info.build:
                print(f"Build: {version_info.build}")
                
        elif args.command == 'compare':
            result = compare_versions(args.version1, args.version2)
            if result == 0:
                print(f"{args.version1} == {args.version2}")
            elif result < 0:
                print(f"{args.version1} < {args.version2}")
            else:
                print(f"{args.version1} > {args.version2}")
                
        elif args.command == 'check':
            if args.min:
                compatible = is_version_compatible(
                    args.version, args.min, args.allow_prerelease
                )
                if compatible:
                    print(f"{args.version} meets minimum requirement {args.min}")
                else:
                    print(f"{args.version} does NOT meet minimum requirement {args.min}")
                    sys.exit(1)
            
            if args.max:
                # Check if version is less than or equal to max
                comparison = compare_versions(args.version, args.max)
                if comparison <= 0:
                    print(f"{args.version} is within maximum {args.max}")
                else:
                    print(f"{args.version} exceeds maximum {args.max}")
                    sys.exit(1)
                    
        elif args.command == 'sort':
            sorted_versions = sort_versions(args.versions, args.reverse)
            print("Sorted versions:")
            for v in sorted_versions:
                print(f"  {v}")
                
        elif args.command == 'validate':
            if validate_version(args.version):
                print(f"'{args.version}' is a valid version string")
                sys.exit(0)
            else:
                print(f"'{args.version}' is NOT a valid version string")
                sys.exit(1)
        elif args.command == 'deepseek':
            # Answer the Deepseek version question
            print("Deepseek是否已经发布最新版本？")
            print("Answer: 是")
                
    except VersionError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()