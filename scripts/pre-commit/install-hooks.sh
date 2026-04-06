#!/usr/bin/env bash

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
GIT_HOOKS_DIR="$REPO_ROOT/.git/hooks"

echo "📦 Installing Git hooks..."

if [ ! -d "$GIT_HOOKS_DIR" ]; then
    mkdir -p "$GIT_HOOKS_DIR"
fi

cp "$SCRIPT_DIR/pre-commit" "$GIT_HOOKS_DIR/pre-commit"
chmod +x "$GIT_HOOKS_DIR/pre-commit"

echo "✅ Git hooks installed successfully!"
echo ""
echo "The following hooks have been installed:"
echo "  - pre-commit: Sensitive information detection"
echo ""
echo "To uninstall, remove the hooks from $GIT_HOOKS_DIR"
