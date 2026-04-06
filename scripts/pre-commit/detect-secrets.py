#!/usr/bin/env python3
import re
import sys
import argparse
from pathlib import Path
from typing import List, Tuple, Dict


SENSITIVE_PATTERNS = [
    (r"(?i)(?:api[_-]?key|apikey|secret[_-]?key|secretkey|private[_-]?key|privatekey|access[_-]?key|accesskey)[\s]*[=:]\s*['\"]?([A-Za-z0-9_\-]{16,})['\"]?", "API Key"),
    (r"(?i)(?:password|passwd|pwd)[\s]*[=:]\s*['\"]?([A-Za-z0-9_\-@#$%^&*+!]{8,})['\"]?", "Password"),
    (r"(?i)(?:token|auth[_-]?token|authtoken)[\s]*[=:]\s*['\"]?([A-Za-z0-9_\-]{20,})['\"]?", "Auth Token"),
    (r"(?i)(?:database[_-]?url|db[_-]?url|postgres|mysql|mongodb)[\s]*[=:]\s*['\"]?([a-z]+://[^\s\"']+)['\"]?", "Database URL"),
    (r"(?i)(?:jwt[_-]?secret|jwt[_-]?key)[\s]*[=:]\s*['\"]?([A-Za-z0-9_\-]{16,})['\"]?", "JWT Secret"),
    (r"(?i)(?:aws[_-]?access[_-]?key|aws[_-]?secret[_-]?access[_-]?key)[\s]*[=:]\s*['\"]?([A-Za-z0-9_\-]{16,})['\"]?", "AWS Credential"),
    (r"(?i)(?:github[_-]?token|ghp_[A-Za-z0-9]{36,})", "GitHub Token"),
    (r"(?i)(?:slack[_-]?token|xox[baprs]-[0-9]{12,}-[0-9]{12,}-[0-9]{12,}-[a-z0-9]{48,})", "Slack Token"),
    (r"(?i)(?:sk_live_[A-Za-z0-9]{24,}|pk_live_[A-Za-z0-9]{24,})", "Stripe Key"),
    (r"-----BEGIN\s+(?:RSA\s+)?PRIVATE\s+KEY-----", "Private Key"),
    (r"(?i)(?:aliyun|qwen|deepseek|minimax|tencent|baidu|openai|claude|moonshot|doubao|zhipu)[\s]*[_-]?(?:api[_-]?key|secret[_-]?key|secret[_-]?id)[\s]*[=:]\s*['\"]?([A-Za-z0-9_\-]{10,})['\"]?", "AI Provider Credential"),
]


EXCLUDED_FILES = [
    r"\.git",
    r"\.env.*",
    r"\.gitignore",
    r"node_modules",
    r"__pycache__",
    r"dist",
    r"build",
    r"vendor",
    r"\.venv",
    r"coverage",
    r"README\.md",
    r"DEVELOPMENT\.md",
]


EXCLUDED_EXTENSIONS = [
    ".md",
    ".txt",
    ".log",
    ".png",
    ".jpg",
    ".jpeg",
    ".gif",
    ".pdf",
    ".zip",
    ".tar",
    ".gz",
]


def is_excluded(file_path: Path) -> bool:
    file_str = str(file_path)
    for pattern in EXCLUDED_FILES:
        if re.search(pattern, file_str):
            return True
    if file_path.suffix in EXCLUDED_EXTENSIONS:
        if "test-samples" in file_str:
            return False
        return True
    return False


def scan_file(file_path: Path) -> List[Tuple[int, str, str]]:
    findings = []
    try:
        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
            lines = f.readlines()
        
        for line_num, line in enumerate(lines, 1):
            for pattern, pattern_name in SENSITIVE_PATTERNS:
                matches = re.finditer(pattern, line)
                for match in matches:
                    findings.append((line_num, pattern_name, match.group(0)))
    except Exception as e:
        print(f"Warning: Could not read {file_path}: {e}", file=sys.stderr)
    return findings


def scan_files(files: List[Path]) -> Dict[Path, List[Tuple[int, str, str]]]:
    results = {}
    for file_path in files:
        if not file_path.exists():
            continue
        if is_excluded(file_path):
            continue
        findings = scan_file(file_path)
        if findings:
            results[file_path] = findings
    return results


def main():
    parser = argparse.ArgumentParser(description="Detect sensitive information in files.")
    parser.add_argument('files', nargs='*', help='Files to scan (if not specified, scan all files in git)')
    parser.add_argument('--all', action='store_true', help='Scan all files in repository')
    parser.add_argument('--staged', action='store_true', help='Scan only staged files (for git pre-commit)')
    
    args = parser.parse_args()
    
    files_to_scan = []
    
    if args.staged:
        import subprocess
        try:
            result = subprocess.run(
                ['git', 'diff', '--cached', '--name-only', '--diff-filter=ACM'],
                capture_output=True,
                text=True,
                check=True
            )
            staged_files = result.stdout.strip().split('\n')
            files_to_scan = [Path(f) for f in staged_files if f]
        except Exception as e:
            print(f"Error getting staged files: {e}", file=sys.stderr)
            sys.exit(1)
    elif args.files:
        files_to_scan = [Path(f) for f in args.files]
    elif args.all:
        repo_root = Path.cwd()
        for file_path in repo_root.rglob('*'):
            if file_path.is_file():
                files_to_scan.append(file_path)
    else:
        print("Please specify --staged, --all, or provide files to scan.", file=sys.stderr)
        parser.print_help()
        sys.exit(1)
    
    results = scan_files(files_to_scan)
    
    if results:
        print("\n⚠️  Sensitive information detected!\n")
        for file_path, findings in results.items():
            print(f"📁 {file_path}:")
            for line_num, pattern_name, match in findings:
                print(f"  Line {line_num}: {pattern_name}")
                print(f"    {match[:100]}{'...' if len(match) > 100 else ''}")
            print()
        sys.exit(1)
    else:
        print("✅ No sensitive information detected.")
        sys.exit(0)


if __name__ == "__main__":
    main()
