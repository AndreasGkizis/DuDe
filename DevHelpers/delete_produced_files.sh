#!/bin/bash

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

LOGS_DIR="$SCRIPT_DIR/../cmd/logs"
CMD_DIR="$SCRIPT_DIR/../cmd"

# Function to remove files
remove_file() {
    filename=$1
    if [ -e "$filename" ]; then
        rm "$filename"
        echo "DELETED ===> $filename"
    else
        echo "$filename - Does not exist"
    fi
}

# Function to remove directories
remove_dir() {
    directory=$1
    if [ -d "$directory" ]; then
        rm -r "$directory"
        echo "DELETED ===> $directory"
    else
        echo "$directory - Doesn't exist or is not a Directory"
    fi
}

remove_dir "$LOGS_DIR"
remove_file "$CMD_DIR/memory.db" 
remove_file "$CMD_DIR/memory.db-journal" 
remove_file "$CMD_DIR/results.csv" 