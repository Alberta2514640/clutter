#!/usr/bin/env bash
# Builds the website's image, and pushes it when IMAGE_REPO is set.
#   IMAGE_TAG=v1 frontend/build-image.sh                                   builds clutter-website:v1
#   IMAGE_TAG=v1 IMAGE_REPO=<registry>/clutter-website frontend/build-image.sh   also pushes it there
# Next bakes NEXT_PUBLIC_API_ENDPOINT, NEXT_PUBLIC_ADAPTER_API_ENDPOINT, NEXT_PUBLIC_GOOGLE_CLIENT_ID and
# NEXT_PUBLIC_CONTACT_EMAIL into the image. Each is read from the environment, or else from
# frontend/clutter/.env.production.local.
# npm run deploy -- prod --site-image passes all four from the platform's config and pushes to ECR
# (docs/architecture/environments.md § Credentials day); .github/workflows/docker-image.yml passes
# stand-in values and only builds. A push refuses a local, example or non-https value, so a pushed
# image never holds one. Log in to the registry first.
# Run by a person, never by a session, when it pushes.
set -euo pipefail

fail() { echo "build-image: $1" >&2; exit 1; }

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SITE_ENV="${HERE}/clutter/.env.production.local"
IMAGE_TAG="${IMAGE_TAG:?Set IMAGE_TAG to a version tag, for example v1}"
[[ "${IMAGE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ && "${IMAGE_TAG}" != latest ]] || fail "IMAGE_TAG must be a version tag, never latest."
command -v docker >/dev/null || fail "docker is required."

# file_value key: KEY=value from .env.production.local, without quotes, or nothing.
file_value() {
  [[ -f "${SITE_ENV}" ]] || return 0
  sed -nE "s/^[[:space:]]*(export[[:space:]]+)?${1}=[\"']?([^\"'[:space:]]*).*/\\2/p" "${SITE_ENV}" | tail -n 1
}

placeholder='localhost|127\.0\.0\.1|example\.com|NUMBER-ID|\.invalid'
args=()
for key in NEXT_PUBLIC_API_ENDPOINT NEXT_PUBLIC_ADAPTER_API_ENDPOINT NEXT_PUBLIC_GOOGLE_CLIENT_ID NEXT_PUBLIC_CONTACT_EMAIL; do
  value="${!key:-$(file_value "${key}")}"
  [[ -n "${value}" ]] || fail "${key} is not set, in the environment or in clutter/.env.production.local."
  if [[ -n "${IMAGE_REPO:-}" ]]; then
    [[ ! "${value}" =~ ${placeholder} ]] || fail "${key} holds a local or example value; an image that is pushed needs the cloud's."
    case "${key}" in
      *_ENDPOINT) [[ "${value}" =~ ^https://[^[:space:]]+$ ]] || fail "${key} is not an https address." ;;
      *_EMAIL) [[ "${value}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]] || fail "${key} is not an email address." ;;
      *) [[ "${value}" =~ ^[0-9]+-[a-z0-9]+\.apps\.googleusercontent\.com$ ]] || fail "${key} is not a Google OAuth client id." ;;
    esac
  fi
  args+=(--build-arg "${key}=${value}")
done

local_tag="clutter-website:${IMAGE_TAG}"
echo "Building ${local_tag} ..."
docker build --platform linux/amd64 --provenance false -f "${HERE}/Dockerfile.frontend" "${args[@]}" -t "${local_tag}" "${HERE}"

if [[ -n "${IMAGE_REPO:-}" ]]; then
  docker tag "${local_tag}" "${IMAGE_REPO}:${IMAGE_TAG}"
  echo "Pushing ${IMAGE_REPO}:${IMAGE_TAG} ..."
  docker push "${IMAGE_REPO}:${IMAGE_TAG}"
  echo "Pushed ${IMAGE_REPO}:${IMAGE_TAG}."
else
  echo "Built ${local_tag}."
fi
