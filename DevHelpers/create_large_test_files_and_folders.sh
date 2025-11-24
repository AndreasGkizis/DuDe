#!/bin/bash

# Get the directory of this script
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

# Create a base directory for the sample files
BASE_DIR="../test_files/source"
BASE_TARGET_DIR="../test_files/target"

# Create a base directory for the sample files relative to the script's location
BASE_DIR="$SCRIPT_DIR/../test_files/source"
BASE_TARGET_DIR="$SCRIPT_DIR/../test_files/target"

# Create subdirectories for different file types
mkdir -p "$BASE_DIR/text_files"

mkdir -p "$BASE_TARGET_DIR/text_files"

create_sample_files(){
    local dir="$1"
    local prefix="$2"
    local num_files="$3"

    local min_size_mb="$4"
    local max_size_mb="$5"

    # Convert MB to bytes
    local min_bytes=$(( min_size_mb * 1024 * 1024 ))
    local max_bytes=$(( max_size_mb * 1024 * 1024 ))

    # Validate sizes
    if (( min_bytes > max_bytes )); then
        echo "Error: Minimum size cannot be greater than maximum size."
        return 1
    fi
    # Create text files
    for ((i=1; i<=num_files; i++)); do
        # Generate random size between min_bytes and max_bytes
        local size_bytes=$(( $RANDOM % (max_bytes - min_bytes + 1) + min_bytes ))
        local random_block=$(($RANDOM % 16 + 1))
        dd if=/dev/random of="$dir/text_files/"$prefix"-text_$i.txt" bs=64M count=$random_block iflag=fullblock

        if (( $i % 2 == 0 )); then
            cp "$dir/text_files/"$prefix"-text_$i.txt" "$dir/text_files/"$prefix"-text_$i-dup.txt"
        fi
    done

    # Create UNIQUE FILES, just here to ensure they are not picked up by accident 
    openssl rand -out "$dir/"$prefix"-unique1.pdf" 512
    openssl rand -out "$dir/"$prefix"-unique2.png" 1024
    openssl rand -out "$dir/"$prefix"-unique3.zip" 2048

    echo "Sample files created in $dir"
}

create_readable_sample_test_files(){
    local dir="$1"
    local prefix="$2"
    local num_files="$3"
    local text_string="This is sample text!!"

    local min_size_mb="$4"
    local max_size_mb="$5"

    local random_size_mb=$((RANDOM % (max_size_mb - min_size_mb + 1) + min_size_mb))

    local random_size_bytes=$((random_size_mb * 1024 * 1024))

    for ((i=1; i<=num_files; i++)); do
        yes "$text_string" | head -c "$random_size_bytes" > "$dir/text_files/"$prefix"-text_read_$i.txt"
    done
}

create_readable_sample_test_files "$BASE_DIR" "source" 10 10 200
create_sample_files "$BASE_DIR" "source" 2 100 1000
create_sample_files "$BASE_TARGET_DIR" "target" 5 100 1000

copy_files_from_to(){
    local source="$1"
    local target="$2"

    cp -r $source $target
}

copy_files_from_to "$BASE_DIR/text_files" "$BASE_TARGET_DIR/text_files"