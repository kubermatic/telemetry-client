#!/usr/bin/env bash

# Copyright 2022 The Telemetry Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

GIT_HEAD_HASH="$(git rev-parse HEAD)"
GIT_HEAD_TAG="$(git tag -l "$PULL_BASE_REF")"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
TAGS="$GIT_HEAD_HASH $GIT_HEAD_TAG"

# we only want to create the "latest" tag if we're building the main branch
if [ "$GIT_BRANCH" == "main" ]; then
  TAGS="$TAGS latest"
fi

if [ -z "$TAGS" ]; then
  echo "Found no tags to build for, cannot proceed. Did you run this script on a tagged revision?"
  exit 1
fi

apt install time -y

echo "Logging into Quay"
start-docker.sh
docker login -u "$QUAY_IO_USERNAME" -p "$QUAY_IO_PASSWORD" quay.io
echo "Successfully logged into Quay"

export DOCKER_REPO="${DOCKER_REPO:-quay.io/kubermatic/telemetry-agent}"
export GOOS="${GOOS:-linux}"

# build Docker image
PRIMARY_TAG=localbuild
make build docker-build TAGS="${PRIMARY_TAG}"

# for each given tag, tag and push the image
for TAG in $TAGS; do
  if [ -z "$TAG" ]; then
    continue
  fi

  echo "Pushing ${DOCKER_REPO}:${TAG} ..."
  docker tag "${DOCKER_REPO}:${PRIMARY_TAG}" "${DOCKER_REPO}:${TAG}"
  docker push "${DOCKER_REPO}:${TAG}"
done
