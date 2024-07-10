#!/bin/bash
# based on: https://github.com/skx/github-action-publish-binaries/blob/master/upload-script
#
# Upload binary artifacts when a new release is made.
#

set -e

# Ensure that the GITHUB_TOKEN secret is included
if [[ -z "$GITHUB_TOKEN" ]]; then
  echo "Set the GITHUB_TOKEN env variable."
  exit 1
fi

# Ensure that the file path is present
if [[ -z "$1" ]]; then
  echo "Missing file (pattern) to upload."
  exit 1
fi

# Only upload to non-draft releases
IS_DRAFT=$(jq --raw-output '.release.draft' $GITHUB_EVENT_PATH)
if [ "$IS_DRAFT" = true ]; then
  echo "This is a draft, so nothing to do!"
  exit 0
fi

# Run the build-script
make docker-build
docker container create --name rita quay.io/activecm/rita-legacy:latest
docker container cp rita:/rita ./rita

# Prepare the headers
AUTH_HEADER="Authorization: token ${GITHUB_TOKEN}"

# Build the Upload URL from the various pieces
RELEASE_ID=$(jq --raw-output '.release.id' $GITHUB_EVENT_PATH)

# For each matching file
for file in $*; do
    echo "Processing file ${file}"

    FILENAME=$(basename ${file})
    UPLOAD_URL="https://uploads.github.com/repos/${GITHUB_REPOSITORY}/releases/${RELEASE_ID}/assets?name=${FILENAME}"
    echo "$UPLOAD_URL"

    # Upload the file
    curl \
        -sSL \
        -XPOST \
        -H "${AUTH_HEADER}" \
        --upload-file "${file}" \
        --header "Content-Type:application/octet-stream" \
        "${UPLOAD_URL}"
done
