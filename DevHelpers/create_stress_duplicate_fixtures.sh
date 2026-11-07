#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 || $# -gt 3 ]]; then
    echo "Usage: $0 OUTPUT_DIR [PAIR_COUNT] [SIZE_MB]" >&2
    exit 2
fi

output_dir=$(realpath -m "$1")
pair_count=${2:-2}
size_mb=${3:-64}

if ! [[ "$pair_count" =~ ^[1-9][0-9]*$ && "$size_mb" =~ ^[1-9][0-9]*$ ]]; then
    echo "PAIR_COUNT and SIZE_MB must be positive integers." >&2
    exit 2
fi

size_bytes=$((size_mb * 1024 * 1024))
total_bytes=$((pair_count * 2 * size_bytes))
printf 'Creating %d duplicate pairs (%d MiB each, about %d MiB total) in %s\n' \
    "$pair_count" "$size_mb" "$((total_bytes / 1024 / 1024))" "$output_dir"

mkdir -p "$output_dir"

for ((i = 1; i <= pair_count; i++)); do
    original="$output_dir/file-${i}.bin"
    duplicate="$output_dir/file-${i}-duplicate.bin"

    # urandom is appropriate for a manual stress fixture; unlike /dev/random,
    # it does not block waiting for entropy.
    head -c "$size_bytes" /dev/urandom > "$original"
    cp "$original" "$duplicate"
done

printf 'Created fixtures:\n'
find "$output_dir" -maxdepth 1 -type f -printf '  %p (%s bytes)\n' | sort
printf 'Remove them with: rm -rf -- %q\n' "$output_dir"
