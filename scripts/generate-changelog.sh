#!/bin/bash
# Auto-generate CHANGELOG.md from git tags and commits
# Usage: ./scripts/generate-changelog.sh

set -euo pipefail

OUTPUT="CHANGELOG.md"
REPO_URL="https://github.com/anonysec/panel"

echo "# KorisPanel Changelog" > "$OUTPUT"
echo "" >> "$OUTPUT"
echo "All notable changes to the project." >> "$OUTPUT"
echo "" >> "$OUTPUT"

# Get all tags sorted by date
tags=$(git tag --sort=-version:refname 2>/dev/null | head -20)

if [ -z "$tags" ]; then
  # No tags — just show recent commits
  echo "## Unreleased" >> "$OUTPUT"
  echo "" >> "$OUTPUT"
  git log --pretty=format:"- %s (%h)" --no-merges -50 >> "$OUTPUT"
  echo "" >> "$OUTPUT"
else
  prev=""
  for tag in $tags; do
    date=$(git log -1 --format=%ai "$tag" | cut -d' ' -f1)
    echo "## $tag — $date" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    
    if [ -n "$prev" ]; then
      git log --pretty=format:"- %s" --no-merges "$tag..$prev" >> "$OUTPUT"
    else
      git log --pretty=format:"- %s" --no-merges "$tag..HEAD" >> "$OUTPUT"
    fi
    echo "" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    prev="$tag"
  done
fi

echo "Generated $OUTPUT"
