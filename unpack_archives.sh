#!/bin/bash
set -e

TARGET_DIR="${1:-.}"

# Find all archive files recursively and extract them
find "$TARGET_DIR" -type f \( \
  -iname "*.zip" -o \
  -iname "*.jar" -o \
  -iname "*.war" -o \
  -iname "*.ear" -o \
  -iname "*.rar" -o \
  -iname "*.tar.gz" -o \
  -iname "*.tgz" -o \
  -iname "*.tar" \
\) | while read -r archive; do
  echo "📦 Extracting: $archive"
  dir="${archive}_unpacked"
  mkdir -p "$dir"

  case "$archive" in
    *.zip | *.jar | *.war | *.ear)
      unzip -qq -o "$archive" -d "$dir" || echo "❌ Failed: $archive"
      ;;
    *.rar)
      unrar x -o+ "$archive" "$dir" || echo "❌ Failed: $archive"
      ;;
    *.tar.gz | *.tgz)
      tar -xzf "$archive" -C "$dir" || echo "❌ Failed: $archive"
      ;;
    *.tar)
      tar -xf "$archive" -C "$dir" || echo "❌ Failed: $archive"
      ;;
  esac
done
