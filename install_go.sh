#!/bin/bash
set -e

# 1. Fetch latest Go version (first line only)
release=$(curl -s "https://go.dev/VERSION?m=text" | head -n 1)
if [ -z "$release" ]; then
    echo "Error: Could not fetch latest Go version from go.dev"
    exit 1
fi

# 2. Check local Go version (if installed)
if command -v go >/dev/null 2>&1; then
    version=$(go version | cut -d' ' -f 3)
else
    version="none"
fi

if [[ "$version" == "$release" ]]; then
    echo "The local Go version ${version} is already up-to-date."
    exit 0
fi

echo "Local Go version: ${version}"
echo "Latest Go version: ${release}"

# 3. Create target apps directory if it doesn't exist
mkdir -p "${HOME}/apps"

# 4. Create temporary downloading folder
tmp=$(mktemp -d)
cd "$tmp"

archive_file="${release}.linux-amd64.tar.gz"
download_url="https://go.dev/dl/${archive_file}"

echo "Downloading ${download_url} ..." 
curl -L -O "${download_url}"

echo "Cleaning old installations..."
rm -rf "${HOME}/apps/go"
rm -rf "${HOME}/apps/${release}"

echo "Extracting Go to ${HOME}/apps ..."
tar -C "${HOME}/apps" -xzf "${archive_file}"

cd - > /dev/null
rm -rf "$tmp"

# 5. Setup symlinks
mv "${HOME}/apps/go" "${HOME}/apps/${release}"
ln -sf "${HOME}/apps/${release}" "${HOME}/apps/go"

# Prepend to path for check
export PATH="${HOME}/apps/go/bin:${PATH}"
new_version=$(go version | cut -d' ' -f 3)

echo "=== Go upgraded successfully! ==="
echo "Now, local Go version is ${new_version}"
echo ""
echo "IMPORTANT: Please make sure your ~/.zshrc contains the following line at the end:"
echo 'export PATH=$HOME/apps/go/bin:$PATH'