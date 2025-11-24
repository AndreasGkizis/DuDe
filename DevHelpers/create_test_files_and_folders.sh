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
mkdir -p "$BASE_DIR/greek_files"

mkdir -p "$BASE_TARGET_DIR/text_files"
mkdir -p "$BASE_TARGET_DIR/image_files"
mkdir -p "$BASE_TARGET_DIR/audio_files"
mkdir -p "$BASE_TARGET_DIR/greek_files"

create_sample_files(){
    local dir="$1"
    local prefix="$2"

    # Create text files
    echo "This is a sample text file." > "$dir/text_files/"$prefix"-has_duplicate.txt"
    cp "$dir/text_files/"$prefix"-has_duplicate.txt" "$dir/text_files/"$prefix"-has_duplicate1.txt" # Duplicate
    echo "Another unique text file." > "$dir/text_files/"$prefix"-has_no_duplicate.txt"

    # Create image files with random bytes
    openssl rand -out "$dir/image_files/"$prefix"-has_many_duplicates.jpg" 1024
    cp "$dir/image_files/"$prefix"-has_many_duplicates.jpg" "$dir/image_files/"$prefix"-has_many_duplicates1.jpg" # Duplicate
    cp "$dir/image_files/"$prefix"-has_many_duplicates.jpg" "$dir/image_files/"$prefix"-has_many_duplicates2.jpg" # Duplicate

    openssl rand -out "$dir/image_files/"$prefix"-has_no_duplicate.png" 2048

    # Create audio files with random bytes
    openssl rand -out "$dir/audio_files/"$prefix"-has_duplicate.mp3" 512
    cp "$dir/audio_files/"$prefix"-has_duplicate.mp3" "$dir/audio_files/"$prefix"-has_duplicate1.mp3" # Duplicate
    openssl rand -out "$dir/audio_files/"$prefix"-has_no_duplicate.wav" 1024

    # Create audio files with random bytes
    openssl rand -out "$dir/greek_files/"$prefix"-έχει-αντίγραφο.txt" 512
    cp "$dir/greek_files/"$prefix"-έχει-αντίγραφο.txt" "$dir/greek_files/"$prefix"-έχει-αντίγραφο1.txt" # Duplicate
    openssl rand -out "$dir/greek_files/"$prefix"-δέν-έχει-αντίγραφο.txt" 1024

    # Create UNIQUE FILES
    openssl rand -out "$dir/"$prefix"-unique1.pdf" 512
    openssl rand -out "$dir/"$prefix"-no_access_file.txt" 512
    openssl rand -out "$dir/"$prefix"-no_access_file1.txt" 512
    openssl rand -out "$dir/"$prefix"-no_access_file2.txt" 512
    chmod 000 "$dir/"$prefix"-no_access_file.txt" # no Read, Write, execute
    chmod 000 "$dir/"$prefix"-no_access_file1.txt" # no Read, Write, execute
    chmod 000 "$dir/"$prefix"-no_access_file2.txt" # no Read, Write, execute
    openssl rand -out "$dir/"$prefix"-unique2.png" 1024
    openssl rand -out "$dir/"$prefix"-unique3.zip" 2048

    echo "Sample files created in $dir"
}

create_sample_files "$BASE_DIR" "source"
create_sample_files "$BASE_TARGET_DIR" "target"

copy_files_from_to(){
    local source="$1"
    local target="$2"

    cp -r $source $target
}

copy_files_from_to "$BASE_DIR/text_files" "$BASE_TARGET_DIR/text_files"
copy_files_from_to "$BASE_DIR/image_files" "$BASE_TARGET_DIR/image_files"
copy_files_from_to "$BASE_DIR/audio_files" "$BASE_TARGET_DIR/audio_files"