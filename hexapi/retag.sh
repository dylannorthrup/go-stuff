#!/bin/bash
#
# Retag the project when it's ready for release

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR
VERS=$(awk -F\" '/var programVersion/ {print $2}' hexapi.go)

echo "Tagging version ${VERS}"
git tag -d ${VERS}
git push origin :refs/tags/${VERS}
git tag ${VERS}
git push --tags

echo "Removing current and stable tags and re-pointing them to this commit"
git tag -d stable
git push origin :refs/tags/stable
git tag -d current
git push origin :refs/tags/current
git tag stable
git tag current
git push --tags
