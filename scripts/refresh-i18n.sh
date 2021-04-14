#!/bin/bash

# "Refresh" all i18n language files by merging missing keys in lang files
# from a base language file. In addition, sort all files by keys.

BASE_DIR=$(dirname "$0")"/../i18n" # Exclude the trailing slash.
BASE_FILE="en.json"

# Iterate through all i18n files and merge them into the base file,
# filling in missing keys.
for fpath in "$BASE_DIR/"*.json; do
	echo $(basename -- $fpath)
	echo "$( jq -s '.[0] * .[1]' -S --indent 4 "$BASE_DIR/$BASE_FILE" $fpath )" > $fpath
done
