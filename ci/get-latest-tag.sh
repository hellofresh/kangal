#!/bin/env bash

set -e

function get_latest_tag() {
    git fetch --tags
    # This suppress an error occurred when the repository is a complete one.
    git fetch --prune || true

    latest_tag=''

    # Get a latest tag in the shape of semver.
    for ref in $(git for-each-ref --sort=-creatordate --format '%(refname)' refs/tags); do
        tag="${ref#refs/tags/}"
        if echo "${tag}" | grep -Eq '^v?([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+)?$'; then
            latest_tag="${tag}"
        break
        fi
    done

    if [ "${latest_tag}" = '' ]; then
        latest_tag="${INPUT_INITIAL_VERSION}"
    fi

    echo "${latest_tag}"
}

get_latest_tag
