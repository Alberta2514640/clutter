#!/usr/bin/env bash
# Deploys Clutter to its one cloud environment, prod (ADR-015; docs/architecture/environments.md
# § Credentials day).
#   npm run deploy:plan -- prod    shows what would change, changes nothing
#   npm run deploy -- prod         plans, asks, then applies (clean main only)
#   npm run deploy -- prod --a-images
#                                  the same, and after you confirm, builds and pushes A's two runner images
#                                  with terraform_deployer_image_tag and ansible_runner_image_tag before
#                                  applying; a tag already in ECR is skipped
#   npm run deploy -- prod --adapter-images
#                                  the same for B's two images, with adapter_image_tag
#   npm run deploy -- prod --site-image
#                                  the same for the website's image, with site_image_tag; it is built with
#                                  A's API address (the api_invoke_url output), B's address (the site's
#                                  own origin, https://<site_domain>), google_client_id and contact_email,
#                                  so nobody types them
#   The flags go together too. Their repositories, and A's API, must exist: the first deploy runs without them.
# The environment is named so that a second one can be added later; today only prod exists, and
# there is no tear-down command.
# Run by a person, never by a session. It has not yet been run against real AWS: only its checks
# are tested (test-deploy.sh). It covers the platform Terraform; the steps it cannot do are printed
# at the end.
set -euo pipefail

INFRA_DIR="${CLUTTER_INFRA_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
BUILD_CMD="${CLUTTER_BUILD_CMD:-${INFRA_DIR}/../api/build.sh}"
ADAPTER_IMAGES_CMD="${CLUTTER_ADAPTER_IMAGES_CMD:-${INFRA_DIR}/../adapter/images/build-and-push.sh}"
SITE_IMAGE_CMD="${CLUTTER_SITE_IMAGE_CMD:-${INFRA_DIR}/../../frontend/build-image.sh}"
A_SCRIPTS_DIR="${CLUTTER_A_SCRIPTS_DIR:-${INFRA_DIR}/modules/fargate/ecr-scripts}"
CHECK_CMD="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-config.sh"
# The website's repository, made by modules/website.
SITE_REPO="clutter-website"

fail() { echo "deploy: $1" >&2; exit 1; }

usage="usage: deploy.sh plan|apply prod [--a-images] [--adapter-images] [--site-image]"
action="${1:-}"
env_name="${2:-}"
a_images=""
adapter_images=""
site_image=""
# --help answers before any tool, file or account is touched.
case " $* " in
  *" --help "* | *" -h "*)
    printf '%s\n' "usage: npm run deploy:plan -- prod, or npm run deploy -- prod [--a-images] [--adapter-images] [--site-image]" \
      "A person runs these; docs/architecture/environments.md § Credentials day says when and how."
    exit 0 ;;
esac
case "${action}" in plan|apply) ;; *) fail "${usage}" ;; esac
case "${env_name}" in prod) ;; *) fail "${usage}" ;; esac
for option in "${@:3}"; do
  case "${option}" in
    --a-images|--adapter-images|--site-image) [[ "${action}" == "apply" ]] || fail "${option} goes only with apply." ;;
    *) fail "${usage}" ;;
  esac
  case "${option}" in --a-images) a_images=1 ;; --adapter-images) adapter_images=1 ;; --site-image) site_image=1 ;; esac
done

tools=(terraform aws git)
[[ -z "${a_images}${adapter_images}${site_image}" ]] || tools+=(docker)
for tool in "${tools[@]}"; do
  command -v "${tool}" >/dev/null || fail "${tool} is required."
done

vars_file="${INFRA_DIR}/${env_name}.tfvars"
# Both files exist, every key is set, and none holds an example value (check-config.sh prints no value).
CLUTTER_INFRA_DIR="${INFRA_DIR}" "${CHECK_CMD}" "${env_name}" \
  || fail "${env_name}.tfvars or ${env_name}.backend.hcl is not complete (above)."

# The credentials in this shell must belong to the account this environment names.
want_account="$(sed -nE 's/^[[:space:]]*clutter_account_id[[:space:]]*=[[:space:]]*"([0-9]+)".*/\1/p' "${vars_file}")"
[[ -n "${want_account}" ]] || fail "clutter_account_id is not set in ${env_name}.tfvars."
have_account="$(aws sts get-caller-identity --query Account --output text 2>/dev/null)" || fail "not logged in to AWS."
[[ "${have_account}" == "${want_account}" ]] || fail "logged in to AWS account ${have_account}, but ${env_name} is account ${want_account}."

# Images are pushed with the tags and region this environment's apply uses, never other values.
tfvar() { sed -nE "s/^[[:space:]]*${1}[[:space:]]*=[[:space:]]*\"([^\"]+)\".*/\\1/p" "${vars_file}" | tail -n 1; }
region="$(tfvar aws_region)"
if [[ -n "${adapter_images}" ]]; then
  adapter_tag="$(tfvar adapter_image_tag)"
  [[ -n "${adapter_tag}" && -n "${region}" ]] || fail "adapter_image_tag or aws_region is not set in ${env_name}.tfvars."
fi
if [[ -n "${a_images}" ]]; then
  deployer_tag="$(tfvar terraform_deployer_image_tag)"
  ansible_tag="$(tfvar ansible_runner_image_tag)"
  [[ -n "${deployer_tag}" && -n "${ansible_tag}" && -n "${region}" ]] \
    || fail "terraform_deployer_image_tag, ansible_runner_image_tag or aws_region is not set in ${env_name}.tfvars."
fi
if [[ -n "${site_image}" ]]; then
  site_tag="$(tfvar site_image_tag)"
  site_domain="$(tfvar site_domain)"
  google_client_id="$(tfvar google_client_id)"
  contact_email="$(tfvar contact_email)"
  [[ -n "${site_tag}" && -n "${site_domain}" && -n "${google_client_id}" && -n "${contact_email}" && -n "${region}" ]] \
    || fail "site_image_tag, site_domain, google_client_id, contact_email or aws_region is not set in ${env_name}.tfvars."
fi

# in_ecr repository tag: true when ECR already holds that tag (repositories are immutable).
in_ecr() { aws ecr describe-images --region "${region}" --repository-name "$1" --image-ids "imageTag=$2" >/dev/null 2>&1; }

# push_a_image folder repository tag: builds and pushes one of A's runner images, unless ECR already
# holds that tag (each image has its own tag, so a deploy that moves one keeps the other).
push_a_image() {
  if in_ecr "$2" "$3"; then
    echo "$2:$3 is already in ECR; not rebuilt."
    return 0
  fi
  # Through bash: the scripts are not marked executable in git.
  AWS_REGION="${region}" IMAGE_TAG="$3" bash "${A_SCRIPTS_DIR}/$1/build-and-push.sh"
}

# push_site_image: builds the website's image with the cloud's four values and pushes it, unless
# ECR already holds site_image_tag. B is served on the site's own origin (the load balancer sends
# /v1/* and /healthz to B), so B's address is https://<site_domain>.
push_site_image() {
  if in_ecr "${SITE_REPO}" "${site_tag}"; then
    echo "${SITE_REPO}:${site_tag} is already in ECR; not rebuilt."
    return 0
  fi
  local registry="${want_account}.dkr.ecr.${region}.amazonaws.com"
  # Chained, because set -e does not stop a function called before ||.
  aws ecr get-login-password --region "${region}" \
    | docker login --username AWS --password-stdin "${registry}" >/dev/null \
    && IMAGE_TAG="${site_tag}" IMAGE_REPO="${registry}/${SITE_REPO}" \
    NEXT_PUBLIC_API_ENDPOINT="${api_url}" \
    NEXT_PUBLIC_ADAPTER_API_ENDPOINT="https://${site_domain}" \
    NEXT_PUBLIC_GOOGLE_CLIENT_ID="${google_client_id}" \
    NEXT_PUBLIC_CONTACT_EMAIL="${contact_email}" \
    "${SITE_IMAGE_CMD}"
}

if [[ "${action}" == "apply" ]]; then
  [[ "$(git -C "${INFRA_DIR}" rev-parse --abbrev-ref HEAD)" == "main" ]] || fail "prod deploys only from main."
  [[ -z "$(git -C "${INFRA_DIR}" status --porcelain)" ]] || fail "prod deploys only from a clean checkout."
fi

echo "Building the Lambdas ..."
"${BUILD_CMD}"

echo "Connecting to ${env_name}'s Terraform state ..."
terraform -chdir="${INFRA_DIR}" init -reconfigure -backend-config="${env_name}.backend.hcl"

# The website's image holds A's API address, which exists only once a deploy has made A's API.
if [[ -n "${site_image}" ]]; then
  api_url="$(terraform -chdir="${INFRA_DIR}" output -raw api_invoke_url 2>/dev/null || true)"
  [[ "${api_url}" =~ ^https://[^[:space:]]+$ ]] \
    || fail "A's API address (the api_invoke_url output) is not in the state yet; nothing changed. On the first deploy, run without --site-image so A's API and the website's repository exist."
fi

plan_file="${env_name}.tfplan"
trap 'rm -f "${INFRA_DIR}/${plan_file}"' EXIT
terraform -chdir="${INFRA_DIR}" plan -var-file="${env_name}.tfvars" -out="${plan_file}"

if [[ "${action}" == "plan" ]]; then
  echo "Plan only: nothing changed."
  exit 0
fi

echo
read -r -p "Type ${env_name} to ${action} this plan in AWS account ${want_account}: " answer
[[ "${answer}" == "${env_name}" ]] || fail "not confirmed; nothing changed."

# Before the apply, so new task definitions never name a tag that is not in ECR yet.
if [[ -n "${a_images}" ]]; then
  echo "Building and pushing A's runner images ..."
  { push_a_image terraform-deployer clutter-terraform-deployer "${deployer_tag}" \
    && push_a_image ansible-runner clutter-ansible-runner "${ansible_tag}"; } \
    || fail "pushing A's runner images failed; nothing applied. On the first deploy, run without --a-images so their repositories exist."
fi
if [[ -n "${adapter_images}" ]]; then
  echo "Building and pushing B's images as ${adapter_tag} ..."
  AWS_REGION="${region}" IMAGE_TAG="${adapter_tag}" "${ADAPTER_IMAGES_CMD}" \
    || fail "pushing B's images failed; nothing applied. On the first deploy, run without --adapter-images so their repositories exist; a tag already pushed needs a new adapter_image_tag."
fi
if [[ -n "${site_image}" ]]; then
  echo "Building and pushing the website's image as ${site_tag} ..."
  push_site_image \
    || fail "pushing the website's image failed; nothing applied. On the first deploy, run without --site-image so its repository exists."
fi

terraform -chdir="${INFRA_DIR}" apply "${plan_file}"

api_now="$(terraform -chdir="${INFRA_DIR}" output -raw api_invoke_url 2>/dev/null || true)"
lb_name="$(terraform -chdir="${INFRA_DIR}" output -raw load_balancer_dns_name 2>/dev/null || true)"

echo
echo "Terraform is applied to ${env_name}. A's API is published at: ${api_now:-(see terraform output api_invoke_url)}"
if [[ -n "${site_image}" && -n "${api_now}" && "${api_now}" != "${api_url}" ]]; then
  echo "A's API address changed in this apply, and the website's image holds the old one: set a new site_image_tag and run again with --site-image."
fi
cat <<EOF
Once, at the DNS provider (docs/architecture/environments.md § Credentials day): point site_domain at
the load balancer, ${lb_name:-(see terraform output load_balancer_dns_name)}.
Images ship with --a-images, --adapter-images and --site-image, the first time and after each tag change.
EOF
