#!/bin/bash
set -eo pipefail

exit 1

git checkout --orphan gh-pages
git commit --allow-empty -m "Initial commit"
