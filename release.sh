#!/usr/bin/env sh
set -e

NAME="ghbackup"
# from args
VERSION="${1}"
test ${VERSION} || (echo "Usage: ./release.sh <version>" && exit 1)

mkdir -p releases
cd releases

build () {
  BUILD="${1}"
  OS="${2}"
  ARCH="${3}"
  ZIP="${NAME}-${VERSION}-${BUILD}.zip"
  echo "Building ${BUILD}"
  GOOS=${OS} GOARCH=${ARCH} go build -v -o="${NAME}" -ldflags="-s -w" ..
  echo "Creating zip file ${ZIP}"
  zip ${ZIP} ${NAME} ../readme.md
  rm ${NAME}
}


build Windows-64bit windows amd64
build MacOS-64bit   darwin  amd64
build Linux-64bit   linux   amd64