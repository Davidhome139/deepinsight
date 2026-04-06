#!/usr/bin/env python3
import sys
import subprocess
from pathlib import Path


def run_test():
    script_dir = Path(__file__).parent
    repo_root = script_dir.parent.parent
    test_file = script_dir / "test-samples.txt"
    
    print("=" * 60)
    print("🧪 测试敏感信息检测器")
    print("=" * 60)
    print()
    
    print("📝 测试 1: 检测测试文件中的敏感信息")
    print("-" * 60)
    
    result = subprocess.run(
        [sys.executable, str(script_dir / "detect-secrets.py"), str(test_file)],
        cwd=repo_root,
        capture_output=True,
        text=True
    )
    
    if result.returncode == 1:
        print("✅ 测试通过：检测器成功发现了敏感信息")
        print()
        print("📊 检测结果：")
        print(result.stdout)
    else:
        print("❌ 测试失败：检测器没有发现敏感信息")
        print(result.stdout)
        print(result.stderr)
        return False
    
    print()
    print("📝 测试 2: 测试普通文件（不应检测到敏感信息）")
    print("-" * 60)
    
    normal_file = script_dir / "detect-secrets.py"
    result = subprocess.run(
        [sys.executable, str(script_dir / "detect-secrets.py"), str(normal_file)],
        cwd=repo_root,
        capture_output=True,
        text=True
    )
    
    if result.returncode == 0:
        print("✅ 测试通过：普通文件没有被误报")
    else:
        print("❌ 测试失败：普通文件被误报为包含敏感信息")
        print(result.stdout)
        return False
    
    print()
    print("=" * 60)
    print("🎉 所有测试完成！")
    print("=" * 60)
    
    return True


if __name__ == "__main__":
    success = run_test()
    sys.exit(0 if success else 1)
