#!/bin/bash

# Create a base directory for the sample files
BASE_DIR="../testfiles/WorkingCopies/Test"
mkdir -p "$BASE_DIR"

# Create subdirectories for different file types
mkdir -p "$BASE_DIR/text_files"
mkdir -p "$BASE_DIR/image_files"
mkdir -p "$BASE_DIR/audio_files"

# Create text files
echo "This is a sample text file." > "$BASE_DIR/text_files/has_duplicate.txt"
cp "$BASE_DIR/text_files/has_duplicate.txt" "$BASE_DIR/text_files/has_duplicate1.txt" # Duplicate
echo "Another unique text file." > "$BASE_DIR/text_files/has_no_duplicate.txt"

# Create image files with random bytes
openssl rand -out "$BASE_DIR/image_files/has_duplicate.jpg" 1024
cp "$BASE_DIR/image_files/has_duplicate.jpg" "$BASE_DIR/image_files/has_duplicate1.jpg" # Duplicate
openssl rand -out "$BASE_DIR/image_files/has_no_duplicate.png" 2048

# Create audio files with random bytes
openssl rand -out "$BASE_DIR/audio_files/has_duplicate.mp3" 512
cp "$BASE_DIR/audio_files/has_duplicate.mp3" "$BASE_DIR/audio_files/has_duplicate1.mp3" # Duplicate
openssl rand -out "$BASE_DIR/audio_files/has_no_duplicate.wav" 1024

echo "Sample files created in $BASE_DIR"
