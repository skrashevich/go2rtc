#!/bin/bash



# Use grep with extended regex (-E) to search for the pattern
grep -rnw './' -e '"api' --include \*.go | while read -r line; do
    filepath=$(echo "$line" | cut -d: -f1)
    linenumber=$(echo "$line" | cut -d: -f2)
    handler=$(echo "$line" | cut -d: -f3- | grep -o 'api[^)]*)' | sed 's/.*,\s*\(.*\))$/\1/' | sed 's/^[ \t]*//;s/[ \t]*$//')
    
    echo "Handler: $handler (File: $filepath, Line: $linenumber)"
    echo "---------------------------------------"
    
    python3 ./scripts/api.py "$filepath" "$handler"

done
