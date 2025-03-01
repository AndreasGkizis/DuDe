#!/bin/bash 
# Get the directory of this script
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

# Create a base directory for the sample files
BASE_DIR="../testfiles/source"
BASE_TARGET_DIR="../testfiles/target"

# Create a base directory for the sample files relative to the script's location
BASE_DIR="$SCRIPT_DIR/../testfiles/source"
BASE_TARGET_DIR="$SCRIPT_DIR/../testfiles/target"

# Function to remove directories
remove_dir() {
    directory=$1
    if [ -d "$directory" ]; then
        rm -r "$directory"
        echo "DELETED  ===> $directory"
    else
        echo "$directory - Doesn't exist or is not a Directory"
    fi
}

remove_dir "$BASE_DIR"
remove_dir "$BASE_TARGET_DIR"
