#!/bin/bash

read -p "GitHub username: " GITHUB_USERNAME
read -sp "GitHub token: " GITHUB_TOKEN
echo

REPO_NAME="transcode-rest"
REPO_ORGANIZATION="refcall"
DOCKER_REGISTRY="ghcr.io"
IMAGE_NAME="$DOCKER_REGISTRY/$REPO_ORGANIZATION/$REPO_NAME"

echo "$GITHUB_TOKEN" | docker login $DOCKER_REGISTRY -u $GITHUB_USERNAME --password-stdin

echo "Getting last commit hash..."
LATEST_COMMIT_HASH=$(git log -1 --format="%H")

echo "Building docker image with tag: $LATEST_COMMIT_HASH..."
docker buildx create --use
docker buildx build --platform linux/amd64 -t $IMAGE_NAME:$LATEST_COMMIT_HASH .

echo "Pushing to GitHub Packages..."
docker push $IMAGE_NAME:$LATEST_COMMIT_HASH
