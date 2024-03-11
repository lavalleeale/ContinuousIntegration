# Custom CI

## What is this?

A complete Continuous Integration solution based in docker that can follow a Directed Acyclic Graph of containers to allow for complex multi-stage builds. Has support for github to automatically run build after push and send results back to show test status (See [a build of this project](https://github.com/lavalleeale/ContinuousIntegration/runs/22526913395)). Also supports manual builds from any git repository.

## Project Structure

List of built docker containers

### alex95712/ci

Main container that hosts the web app and communicates with the docker daemon to run containers and facilitate the build process

### alex95712/registry-auth

Handles authentication for a docker registry to allow containers to access the correct images that are owned by the same organization as the running container. Also allowes users to access images uploaded from containers

### alex95712/proxy

Proxies requests to direct them to the correct container to allow for preview deployments accessable from the web from domains such as `https://aaaaaa.ci-proxy.lavallee.one`
