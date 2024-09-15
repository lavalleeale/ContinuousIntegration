#! /usr/bin/env bash
set -e

docker build images/base -t alex95712/base
docker build . -f Dockerfile.registry -t alex95712/registry-auth
docker build . -f Dockerfile.ContinuousIntegration -t alex95712/ci
docker build services/proxy -t alex95712/proxy
docker push alex95712/base
docker push alex95712/registry-auth
docker push alex95712/ci
docker push alex95712/proxy
