#!/usr/bin/env bash
set -euo pipefail

# ===== CONFIG =====
AWS_REGION="${AWS_REGION:-us-west-2}"
ECR_REPO_NAME="${ECR_REPO_NAME:-clutter-ansible-runner}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
# ==================

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

ECR_URI="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO_NAME}"

echo "Logging into ECR..."
aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "$ECR_URI"

echo "Building Docker image..."
docker build --platform linux/amd64 --provenance false -t "${ECR_REPO_NAME}:${IMAGE_TAG}" .

echo "Tagging image..."
docker tag "${ECR_REPO_NAME}:${IMAGE_TAG}" "${ECR_URI}:${IMAGE_TAG}"

echo "Pushing image to ECR..."
docker push "${ECR_URI}:${IMAGE_TAG}"

echo "Image pushed:"
echo "   ${ECR_URI}:${IMAGE_TAG}"
