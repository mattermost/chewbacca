#!/bin/bash
set -e
set -u

: ${GITHUB_SHA:?}

export TAG="${GITHUB_SHA:0:7}"

make build-image-with-tag
