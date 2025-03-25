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
mkdir -p "$BASE_DIR/image_files"
mkdir -p "$BASE_DIR/audio_files"

mkdir -p "$BASE_TARGET_DIR/text_files"
mkdir -p "$BASE_TARGET_DIR/image_files"
mkdir -p "$BASE_TARGET_DIR/audio_files"

create_sample_files(){
    local dir="$1"
    local prefix="$2"
    local num_of_files="$3"

    for i in $(seq 1 $num_of_files); do
    # Create text files
    echo "This is a sample text file." > "$dir/text_files/"$prefix"-"$i"-has_duplicate.txt"
    cp "$dir/text_files/"$prefix"-"$i"-has_duplicate.txt" "$dir/text_files/"$prefix"-"$i"-has_duplicate1.txt" # Duplicate
    echo "Another unique text file." > "$dir/text_files/"$prefix"-"$i"-has_no_duplicate.txt"

    # Create image files with random bytes
    openssl rand -out "$dir/image_files/"$prefix"-"$i"-has_duplicate.jpg" 1024
    cp "$dir/image_files/"$prefix"-"$i"-has_duplicate.jpg" "$dir/image_files/"$prefix"-"$i"-has_duplicate1.jpg" # Duplicate
    openssl rand -out "$dir/image_files/"$prefix"-"$i"-has_no_duplicate.png" 2048

    # Create audio files with random bytes
    openssl rand -out "$dir/audio_files/"$prefix"-"$i"-has_duplicate.mp3" 512
    cp "$dir/audio_files/"$prefix"-"$i"-has_duplicate.mp3" "$dir/audio_files/"$prefix"-"$i"-has_duplicate1.mp3" # Duplicate
    openssl rand -out "$dir/audio_files/"$prefix"-"$i"-has_no_duplicate.wav" 1024


    # Create UNIQUE FILES
    openssl rand -out "$dir/"$prefix"-"$i"-unique1.doc" 201011048
    openssl rand -out "$dir/"$prefix"-"$i"-unique1.pdf" 20011048
    openssl rand -out "$dir/"$prefix"-"$i"-unique2.png" 2001048
    openssl rand -out "$dir/"$prefix"-"$i"-unique3.zip" 200048
    done

    echo "Sample files created in $dir"
}

create_sample_files "$BASE_DIR" "source" 50
create_sample_files "$BASE_TARGET_DIR" "target" 50

copy_files_from_to(){
    local source="$1"
    local target="$2"

    cp -r $source $target
}

copy_files_from_to "$BASE_DIR/text_files" "$BASE_TARGET_DIR/text_files"
copy_files_from_to "$BASE_DIR/image_files" "$BASE_TARGET_DIR/image_files"
copy_files_from_to "$BASE_DIR/audio_files" "$BASE_TARGET_DIR/audio_files"