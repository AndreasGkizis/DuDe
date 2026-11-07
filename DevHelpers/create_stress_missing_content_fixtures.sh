#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 || $# -gt 4 ]]; then
    echo "Usage: $0 OUTPUT_DIR [COVERED_COUNT] [MISSING_COUNT] [SIZE_MB]" >&2
    exit 2
fi

output_dir=$(realpath -m "$1")
covered_count=${2:-2}
missing_count=${3:-2}
size_mb=${4:-64}

if ! [[ "$covered_count" =~ ^[1-9][0-9]*$ && "$missing_count" =~ ^[1-9][0-9]*$ && "$size_mb" =~ ^[1-9][0-9]*$ ]]; then
    echo "COVERED_COUNT, MISSING_COUNT, and SIZE_MB must be positive integers." >&2
    exit 2
fi

size_bytes=$((size_mb * 1024 * 1024))
total_files=$((covered_count * 2 + missing_count))
total_bytes=$((total_files * size_bytes))
source_dir="$output_dir/source"
target_dir="$output_dir/target"

printf 'Creating coverage fixtures (%d covered, %d missing, %d MiB each, about %d MiB total) in %s\n' \
    "$covered_count" "$missing_count" "$size_mb" "$((total_bytes / 1024 / 1024))" "$output_dir"

mkdir -p "$source_dir" "$target_dir"

for ((i = 1; i <= covered_count; i++)); do
    source_file="$source_dir/source-covered-${i}.bin"
    target_file="$target_dir/renamed-target-${i}.bin"

    head -c "$size_bytes" /dev/urandom > "$source_file"
    cp "$source_file" "$target_file"
done

for ((i = 1; i <= missing_count; i++)); do
    head -c "$size_bytes" /dev/urandom > "$source_dir/source-missing-${i}.bin"
done

printf 'Created source files:\n'
find "$source_dir" -maxdepth 1 -type f -printf '  %p (%s bytes)\n' | sort
printf 'Created target files:\n'
find "$target_dir" -maxdepth 1 -type f -printf '  %p (%s bytes)\n' | sort
printf 'Remove them with: rm -rf -- %q\n' "$output_dir"
