#!/usr/bin/env bash
repo_root=$(dirname $0)/..

BUILDCMD=${BUILDCMD:-podman build}

REPO=${REPO:-cloud-native-event-}
if [ -z ${VERSION+a} ]; then
	VERSION=$(git describe --abbrev=8 --dirty --always)
fi

NAME=${REPO}consumer:${VERSION}
${BUILDCMD} --squash  -f examples/consumer.Dockerfile -t "${NAME}" $(dirname $0)/..