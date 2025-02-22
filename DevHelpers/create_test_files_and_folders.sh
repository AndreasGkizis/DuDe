#!/bin/bash

# Create a base directory for the sample files
BASE_DIR="../testfiles/WorkingCopies/Test"
BASE_TARGET_DIR="../testfiles/WorkingCopies/Test_target"

mkdir -p "$BASE_DIR"
mkdir -p "$BASE_TARGET_DIR"

# Create subdirectories for different file types
mkdir -p "$BASE_DIR/text_files"
mkdir -p "$BASE_DIR/image_files"
mkdir -p "$BASE_DIR/audio_files"

mkdir -p "$BASE_TARGET_DIR/text_files"
mkdir -p "$BASE_TARGET_DIR/image_files"
mkdir -p "$BASE_TARGET_DIR/audio_files"

create_sample_files(){
    local dir="$1"
    # Create text files
    echo "This is a sample text file." > "$dir/text_files/has_duplicate.txt"
    cp "$dir/text_files/has_duplicate.txt" "$dir/text_files/has_duplicate1.txt" # Duplicate
    echo "Another unique text file." > "$dir/text_files/has_no_duplicate.txt"

    # Create image files with random bytes
    openssl rand -out "$dir/image_files/has_duplicate.jpg" 1024
    cp "$dir/image_files/has_duplicate.jpg" "$dir/image_files/has_duplicate1.jpg" # Duplicate
    openssl rand -out "$dir/image_files/has_no_duplicate.png" 2048

    # Create audio files with random bytes
    openssl rand -out "$dir/audio_files/has_duplicate.mp3" 512
    cp "$dir/audio_files/has_duplicate.mp3" "$dir/audio_files/has_duplicate1.mp3" # Duplicate
    openssl rand -out "$dir/audio_files/has_no_duplicate.wav" 1024

    echo "Sample files created in $dir"
}

create_sample_files "$BASE_DIR"
create_sample_files "$BASE_TARGET_DIR"

copy_files_from_to(){
    local source="$1"
    local target="$2"

    cp -r $source $target
}

copy_files_from_to "$BASE_DIR/text_files" "$BASE_TARGET_DIR/text_files"
copy_files_from_to "$BASE_DIR/image_files" "$BASE_TARGET_DIR/image_files"
copy_files_from_to "$BASE_DIR/audio_files" "$BASE_TARGET_DIR/audio_files"