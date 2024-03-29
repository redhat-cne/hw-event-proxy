repo_root=$(dirname $0)/..

BUILDCMD=${BUILDCMD:-podman build}

REPO=${REPO:-hw-event-proxy}
if [ -z ${VERSION+a} ]; then
	VERSION=$(git describe --abbrev=8 --dirty --always)
fi
NAME=${REPO}:${VERSION}

${BUILDCMD} -f Dockerfile -t "${NAME}" $(dirname $0)/..
