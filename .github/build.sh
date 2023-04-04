#!/bin/bash

ref=${GITHUB_REF_NAME//\//-}
image_name=$GITHUB_WORKFLOW-ci-$ref

cleanup() {
  docker rm -f ${image_name}
}

set -xe

docker build --progress=plain -f .github/Dockerfile . -t ${image_name}
docker run --name=${image_name} --rm -d $image_name sleep 300

trap "cleanup" EXIT INT TERM

rm -rf build
docker cp -q ${image_name}:/app/build build

cat build/sha256sums
ls -lh build
