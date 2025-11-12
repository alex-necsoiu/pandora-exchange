#!/bin/bash
# Setup pre-commit hooks for local development
# Usage: ./scripts/setup-pre-commit.sh

set -e

echo "🔧 Setting up pre-commit hooks..."

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is required but not installed"
    echo "   Install: brew install python3 (macOS) or apt-get install python3 (Linux)"
    exit 1
fi

# Check if pip is installed
if ! command -v pip3 &> /dev/null; then
    echo "❌ pip3 is required but not installed"
    echo "   Install: curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && python3 get-pip.py"
    exit 1
fi

# Install pre-commit
echo "📦 Installing pre-commit..."
pip3 install --user pre-commit || {
    echo "❌ Failed to install pre-commit"
    exit 1
}

# Verify installation
if ! command -v pre-commit &> /dev/null; then
    echo "⚠️  pre-commit installed but not in PATH"
    echo "   Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo "   Add to ~/.zshrc or ~/.bashrc to make permanent"
    export PATH="$HOME/.local/bin:$PATH"
fi

# Install Go tools required by hooks
echo "📦 Installing Go tools..."

go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install pre-commit hooks
echo "🪝 Installing pre-commit hooks..."
pre-commit install
pre-commit install --hook-type commit-msg

# Run pre-commit on all files to test
echo "🧪 Running pre-commit on all files (this may take a while)..."
pre-commit run --all-files || {
    echo "⚠️  Some hooks failed. This is normal on first run."
    echo "   Fix the issues and run: pre-commit run --all-files"
}

echo ""
echo "✅ Pre-commit hooks installed successfully!"
echo ""
echo "📝 Usage:"
echo "   - Hooks run automatically on git commit"
echo "   - Run manually: pre-commit run --all-files"
echo "   - Update hooks: pre-commit autoupdate"
echo "   - Skip hooks (not recommended): git commit --no-verify"
echo ""
echo "🔍 Installed hooks:"
echo "   ✓ go-fmt: Format Go code"
echo "   ✓ go-vet: Run Go vet"
echo "   ✓ go-mod-tidy: Verify go.mod is tidy"
echo "   ✓ golangci-lint: Comprehensive linting"
echo "   ✓ trailing-whitespace: Remove trailing whitespace"
echo "   ✓ check-yaml: Validate YAML files"
echo "   ✓ check-json: Validate JSON files"
echo "   ✓ gitleaks: Detect secrets"
echo "   ✓ conventional-pre-commit: Validate commit messages"
echo "   ✓ sqlc-verify: Verify sqlc generated code"
echo "   ✓ go-test: Run tests on changed files"
echo ""
echo "💡 Commit message format: <type>(<scope>): <description>"
echo "   Types: feat, fix, docs, style, refactor, test, chore"
echo "   Example: feat(auth): add JWT token rotation"
echo ""
