#!/usr/bin/env bash
# Builds A's Terraform deployer image and pushes it to ECR under one version tag.
#   AWS_REGION=ca-central-1 IMAGE_TAG=v1 backend/infra/modules/fargate/ecr-scripts/terraform-deployer/build-and-push.sh
# The tag must be new: the repository is immutable. Set terraform_deployer_image_tag in the
# environment's tfvars to the same tag; npm run deploy -- prod --a-images runs this with both values.
# Run by a person, never by a session: it pushes an image.
set -euo pipefail

fail() { echo "build-and-push: $1" >&2; exit 1; }

AWS_REGION="${AWS_REGION:?Set AWS_REGION to the platform region, for example ca-central-1}"
IMAGE_TAG="${IMAGE_TAG:?Set IMAGE_TAG to a new version tag, for example v1}"
ECR_REPO_NAME="clutter-terraform-deployer"
[[ "${IMAGE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]] || fail "IMAGE_TAG is not a valid tag."
[[ "${IMAGE_TAG}" != "latest" ]] || fail "use a version tag, never latest."

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for tool in aws docker; do
  command -v "${tool}" >/dev/null || fail "${tool} is required."
done

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
ECR_URI="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO_NAME}"

echo "Logging into ECR ..."
aws ecr get-login-password --region "${AWS_REGION}" \
  | docker login --username AWS --password-stdin "${ECR_URI}"

echo "Building ${ECR_URI}:${IMAGE_TAG} ..."
docker build --platform linux/amd64 --provenance false -t "${ECR_URI}:${IMAGE_TAG}" "${HERE}"

echo "Pushing ${ECR_URI}:${IMAGE_TAG} ..."
docker push "${ECR_URI}:${IMAGE_TAG}"

echo "Image pushed:"
echo "   ${ECR_URI}:${IMAGE_TAG}"
