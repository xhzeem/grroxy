#!/bin/bash

# Find all directories named "frontend"
find . -type d -name "frontend" | while read -r frontend_dir; do
    echo "Installing dependencies in $frontend_dir"
    
    # Change to the frontend directory and run npm install
    cd "$frontend_dir"
    npm install
    cd - > /dev/null  # Return to the original directory
    echo "----------------------------------------"
done 