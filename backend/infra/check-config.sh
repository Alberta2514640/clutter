#!/usr/bin/env bash
# Checks that an environment's config files are complete, without printing any value
# (docs/architecture/environments.md § Credentials day).
#   backend/infra/check-config.sh prod     prod.tfvars and prod.backend.hcl (deploy.sh runs this)
# The keys come from the committed examples beside each file. A key is reported when the real file
# does not set it, when it still holds an example placeholder, or when its form is wrong. Only key
# names and file names are printed. The website's build values come from these files too
# (deploy.sh --site-image), so there is no separate frontend file to check.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="${CLUTTER_INFRA_DIR:-${HERE}}"

usage="usage: check-config.sh prod"
env_name="${1:-}"
case "${env_name}" in prod) ;; *) echo "${usage}" >&2; exit 2 ;; esac
[[ $# -le 1 ]] || { echo "${usage}" >&2; exit 2; }

# The placeholders the examples use. A real value never contains one.
placeholder='REPLACE|PASSWORD@|123456789012|example\.com|NUMBER-ID|PROJECT-REF|localhost|127\.0\.0\.1'

problems=0
problem() { echo "  ${1}: ${2}"; problems=$((problems + 1)); }

# tf_value file key: the value of key in an HCL assignments file (quoted or bare), or nothing.
tf_value() { sed -nE "s/^[[:space:]]*${2}[[:space:]]*=[[:space:]]*(\"([^\"]*)\"|([^[:space:]#]+)).*/\\2\\3/p" "$1" | tail -n 1; }
tf_keys() { sed -nE 's/^[[:space:]]*([a-z_]+)[[:space:]]*=.*/\1/p' "$1"; }

# form key value: reports a value of the wrong form, by key name only.
form() {
  local key="$1" v="$2" f="$3"
  case "${key}" in
    aws_region|region) [[ "${v}" =~ ^[a-z]{2}-[a-z]+-[0-9]$ ]] || problem "${f}" "${key} is not an AWS region" ;;
    clutter_account_id) [[ "${v}" =~ ^[0-9]{12}$ ]] || problem "${f}" "${key} is not a 12-digit account id" ;;
    jwt_secret) (( ${#v} >= 32 )) || problem "${f}" "${key} is shorter than 32 characters" ;;
    psql_connection_string) [[ "${v}" =~ ^postgres(ql)?:// ]] || problem "${f}" "${key} is not a postgres:// address" ;;
    site_domain)
      [[ "${v}" =~ ^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$ && ${#v} -le 128 ]] \
        || problem "${f}" "${key} is not a lower-case DNS name (no https://, port or path)" ;;
    google_client_id)
      [[ "${v}" =~ ^[0-9]+-[a-z0-9]+\.apps\.googleusercontent\.com$ ]] || problem "${f}" "${key} is not a Google OAuth client id" ;;
    contact_email) [[ "${v}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]] || problem "${f}" "${key} is not an email address" ;;
    *_image_tag) [[ "${v}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ && "${v}" != latest ]] || problem "${f}" "${key} is not a version tag (never latest)" ;;
    certificate_arn) [[ "${v}" =~ ^arn:aws:acm:[a-z0-9-]+:[0-9]{12}:certificate/[A-Za-z0-9-]+$ ]] || problem "${f}" "${key} is not an ACM certificate ARN" ;;
    bucket) [[ "${v}" =~ ^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$ ]] || problem "${f}" "${key} is not a bucket name" ;;
  esac
}

# check real example: every key of the example is set in real, not a placeholder, and well formed.
check() {
  local real="$1" example="$2" f key v
  f="$(basename "${real}")"
  if [[ ! -f "${real}" ]]; then
    problem "${f}" "missing; copy $(basename "${example}") to it and fill it in"
    return
  fi
  while read -r key; do
    v="$(tf_value "${real}" "${key}")"
    if [[ -z "${v}" ]]; then
      problem "${f}" "${key} is not set"
    elif [[ "${v}" =~ ${placeholder} ]]; then
      problem "${f}" "${key} still holds an example value"
    else
      form "${key}" "${v}" "${f}"
    fi
  done < <(tf_keys "${example}")
}

vars="${INFRA_DIR}/${env_name}.tfvars"
backend="${INFRA_DIR}/${env_name}.backend.hcl"
echo "Checking ${env_name}'s config (values are never printed) ..."
check "${vars}" "${HERE}/prod.tfvars.example"
check "${backend}" "${HERE}/prod.backend.hcl.example"

# Values that must agree across keys.
if [[ -f "${vars}" ]]; then
  region="$(tf_value "${vars}" aws_region)"
  cert="$(tf_value "${vars}" certificate_arn)"
  if [[ -n "${region}" && "${cert}" == arn:aws:acm:* && "${cert}" != "arn:aws:acm:${region}:"* ]]; then
    problem "$(basename "${vars}")" "certificate_arn is not in aws_region; the load balancer needs it there"
  fi
  if [[ -f "${backend}" && -n "${region}" && "$(tf_value "${backend}" region)" != "${region}" ]]; then
    problem "$(basename "${backend}")" "region differs from aws_region in $(basename "${vars}")"
  fi
fi

if (( problems > 0 )); then
  echo "config: ${problems} to fix"
  exit 1
fi
echo "config: complete"
