#!/usr/bin/env bash
# Builds B's two images, B API and the run task, and pushes them to ECR under one version tag.
#   AWS_REGION=ca-central-1 IMAGE_TAG=v1 backend/adapter/images/build-and-push.sh
# The tag must be new: the repositories are immutable. Set adapter_image_tag in the environment's tfvars to
# the same tag. npm run deploy -- prod --adapter-images runs this before the apply, with both values.
# Run by a person, never by a session: it pushes images.
set -euo pipefail

fail() { echo "build-and-push: $1" >&2; exit 1; }

AWS_REGION="${AWS_REGION:?Set AWS_REGION to the platform region, for example ca-central-1}"
IMAGE_TAG="${IMAGE_TAG:?Set IMAGE_TAG to a new version tag, for example v1}"
API_REPO="${API_REPO:-clutter-adapter-api}"
RUN_REPO="${RUN_REPO:-clutter-adapter-runtask}"
[[ "${IMAGE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]] || fail "IMAGE_TAG is not a valid tag."
[[ "${IMAGE_TAG}" != "latest" ]] || fail "use a version tag, never latest."

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTEXT="$(cd "${HERE}/../.." && pwd)" # backend/, because the adapter imports backend/api

for tool in aws docker; do
  command -v "${tool}" >/dev/null || fail "${tool} is required."
done

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
REGISTRY="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

echo "Logging into ECR ..."
aws ecr get-login-password --region "${AWS_REGION}" \
  | docker login --username AWS --password-stdin "${REGISTRY}"

for image in "api:${API_REPO}" "runtask:${RUN_REPO}"; do
  dockerfile="${image%%:*}.Dockerfile"
  uri="${REGISTRY}/${image#*:}:${IMAGE_TAG}"
  echo "Building ${uri} ..."
  docker build --platform linux/amd64 --provenance false -f "${HERE}/${dockerfile}" -t "${uri}" "${CONTEXT}"
  echo "Pushing ${uri} ..."
  docker push "${uri}"
done

echo "Pushed B's images with tag ${IMAGE_TAG}."
