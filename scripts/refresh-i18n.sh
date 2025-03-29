#!/bin/bash

# "Refresh" all i18n language files by merging and syncing keys with the base file.
BASE_DIR=$(dirname "$0")"/../i18n" # Exclude the trailing slash.
BASE_FILE="en.json"

# Iterate through all i18n files and sync them with the base file.
for fpath in "$BASE_DIR/"*.json; do
    if [ "$(basename -- "$fpath")" = "$BASE_FILE" ]; then
        continue  # Skip the base file itself
    fi
    echo "$(basename -- "$fpath")"
    jq -s --indent 4 --sort-keys \
        '.[0] as $base | .[1] as $target |
        $base | with_entries(.value = ($target[.key] // .value))' \
        "$BASE_DIR/$BASE_FILE" "$fpath" > "$fpath.tmp" && mv "$fpath.tmp" "$fpath"
done
