#!/bin/bash -e
#
# Usage:
#   ./release.sh v1.0.0 "Comment for release"

TAG="$1"
if [ "$TAG" == "" ]
then
  echo "ERROR: You must specify a tag parameter, e.g. v1.0.0" >&2
  exit 1
fi

COMMENT="$2"
if [ "$COMMENT" == "" ]
then
  echo "ERROR: You must specify a comment parameter" >&2
  exit 1
fi

if [ "$GITHUB_TOKEN" == "" ]
then
  echo "ERROR: You must set GITHUB_TOKEN to make a release" >&2
  exit 1
fi

CURRENT_BRANCH=$(git branch | grep "^\*" | awk '{print $2;}')
if [ "$CURRENT_BRANCH" != "master" ]
then
  echo "ERROR: You must be on the master branch locally" >&2
  exit 1
fi

# check clone has same commit point as upstream master
git fetch upstream master

CLONE_MASTER_COMMIT=$(git show-ref refs/heads/master | awk '{print $1;}')
UPSTREAM_MASTER_COMMIT=$(git ls-remote upstream master | awk '{print $1;}')

if [ "$CLONE_MASTER_COMMIT" != "$UPSTREAM_MASTER_COMMIT" ]
then
  echo "ERROR: Your clone master must be at the same commit point as upstream master" >&2
  exit 1
fi

echo "Logging in to dockerhub"
docker login || exit 2
echo "Building goethite/goswim:$TAG image"
docker build -t goethite/goswim:$TAG . || exit 2
echo "Pushing goethite/goswim:$TAG to dockerhub"
docker push goethite/goswim:$TAG || exit 2

echo "Tagging master as $TAG"
git tag -a $TAG -m "$COMMENT"

echo "Pushing tag $TAG upstream"
git push upstream $TAG

echo "Releasing to github..."
goreleaser --rm-dist
