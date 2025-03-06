#! /usr/bin/env bash
set -e

docker buildx build --platform linux/amd64,linux/arm64 images/base -t alex95712/base --push
docker buildx build --platform linux/amd64,linux/arm64 . -f Dockerfile.registry -t alex95712/registry-auth --push
docker buildx build --platform linux/amd64,linux/arm64 . -f Dockerfile.ContinuousIntegration -t alex95712/ci --push
docker buildx build --platform linux/amd64,linux/arm64 services/proxy -t alex95712/proxy --push

docker pull alex95712/base
docker tag alex95712/base base:latest
