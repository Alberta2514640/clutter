#!/usr/bin/env bash
# Real AWS from this machine (`npm run real-aws -- setup|status|teardown`):
# Clutter runs here (npm run dev), and B's AWS runs build in real AWS accounts
# (world 2) instead of LocalStack. Development only.
#
#   setup     with an administrator's profile for Clutter's own AWS account,
#             deploys real-aws/dev-account.yaml as the stack clutter-dev-real-aws
#             (the role ClutterAdapterRunnerRole and a templates bucket),
#             uploads the account-link template, and writes real-aws.conf.
#             It creates resources in AWS, so a person runs it.
#   status    checks real-aws.conf, the runner profile and the template address.
#   teardown  deletes the stack, its bucket's objects and real-aws.conf. Run it
#             before deploying the hosted platform (backend/infra) into the same
#             account: the platform makes roles of the same names.
#
# Delete real-aws.conf to go back to LocalStack; then run npm run dev again.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

usage() {
  cat <<EOF
usage: npm run real-aws -- setup --admin-profile PROFILE [--region REGION] [--runner-profile NAME]
       npm run real-aws -- status
       npm run real-aws -- teardown --admin-profile PROFILE

Lets Clutter on this machine deploy into real AWS accounts instead of LocalStack.
  setup     sets up Clutter's own AWS account for it (a role and a templates bucket,
            the stack ${REAL_AWS_STACK}) and writes backend/devserver/real-aws.conf.
            PROFILE is an administrator's profile in ~/.aws/config for Clutter's
            account. REGION defaults to ca-central-1, NAME to clutter-dev-runner.
  status    checks the settings, the runner profile and the template address.
  teardown  removes the stack and real-aws.conf.
Delete backend/devserver/real-aws.conf to go back to LocalStack. Restart npm run dev after
either change.
EOF
}

action="${1:-}"
[[ $# -gt 0 ]] && shift
admin="" region="ca-central-1" runner="clutter-dev-runner"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --admin-profile) admin="${2:-}" && shift 2 || { usage >&2; exit 2; } ;;
    --region) region="${2:-}" && shift 2 || { usage >&2; exit 2; } ;;
    --runner-profile) runner="${2:-}" && shift 2 || { usage >&2; exit 2; } ;;
    -h | --help) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done
case "${action}" in
  setup | teardown)
    if [[ ! "${admin}" =~ ^[A-Za-z0-9_.-]{1,64}$ ]]; then
      echo "${action} needs --admin-profile, a profile in ~/.aws/config for Clutter's own AWS account." >&2
      exit 2
    fi
    ;;
  status) ;;
  -h | --help) usage; exit 0 ;;
  *) usage >&2; exit 2 ;;
esac
if [[ ! "${region}" =~ ^[a-z]{2}(-[a-z]+)+-[0-9]$ || ! "${runner}" =~ ^[A-Za-z0-9_.-]{1,64}$ ]]; then
  usage >&2
  exit 2
fi
command -v aws >/dev/null || { echo "Missing the AWS CLI: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html" >&2; exit 1; }

# admin_aws ARGS: the AWS CLI as the administrator, in the region.
admin_aws() {
  aws --profile "${admin}" --region "${region}" "$@"
}

# stack_output KEY: one output of the stack.
stack_output() {
  admin_aws cloudformation describe-stacks --stack-name "${REAL_AWS_STACK}" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue" --output text
}

setup() {
  local account bucket url role
  echo "Checking the administrator's sign-in (profile ${admin}) ..."
  if ! account="$(admin_aws sts get-caller-identity --query Account --output text 2>&1)"; then
    echo "${account}" >&2
    echo "The profile ${admin} cannot sign in. For an SSO profile, run: aws sso login --profile ${admin}" >&2
    exit 1
  fi
  echo "Clutter's account: ${account}"
  echo "Deploying the stack ${REAL_AWS_STACK} (the role ${REAL_AWS_RUNNER_ROLE} and a templates bucket) ..."
  admin_aws cloudformation deploy --stack-name "${REAL_AWS_STACK}" \
    --template-file "${DEVSERVER_DIR}/real-aws/dev-account.yaml" \
    --capabilities CAPABILITY_NAMED_IAM --no-fail-on-empty-changeset
  bucket="$(stack_output BucketName)"
  url="$(stack_output TemplateURL)"
  role="$(stack_output RunnerRoleArn)"
  echo "Uploading the account-link template to s3://${bucket}/${REAL_AWS_TEMPLATE_KEY} ..."
  admin_aws s3 cp --only-show-errors \
    "${REPO_DIR}/backend/infra/modules/s3/uploads/client_side_terraform_deployer_role.yaml" \
    "s3://${bucket}/${REAL_AWS_TEMPLATE_KEY}"
  cat >"${REAL_AWS_CONF}.new" <<EOF
# Written by npm run real-aws -- setup. Delete this file to go back to LocalStack.
CLUTTER_AWS_ACCOUNT_ID=${account}
CLUTTER_TEMPLATE_URL=${url}
CLUTTER_RUNNER_PROFILE=${runner}
CLUTTER_AWS_REGION=${region}
EOF
  read_real_aws_conf "${REAL_AWS_CONF}.new" || { rm -f "${REAL_AWS_CONF}.new"; exit 1; }
  mv "${REAL_AWS_CONF}.new" "${REAL_AWS_CONF}"
  echo "Wrote backend/devserver/real-aws.conf."
  if aws configure get role_arn --profile "${runner}" >/dev/null 2>&1; then
    echo "The profile ${runner} exists in ~/.aws/config."
  else
    cat <<EOF

One step left: add this profile to ~/.aws/config, so B can act as the role:

[profile ${runner}]
role_arn = ${role}
source_profile = ${admin}
region = ${region}
EOF
  fi
  echo
  echo "Then: npm run real-aws -- status, and npm run dev (it restarts Clutter in real AWS mode)."
}

status() {
  local identity code bad=0
  if [[ ! -f "${REAL_AWS_CONF}" ]]; then
    echo "Real AWS is off: no backend/devserver/real-aws.conf. AWS runs use LocalStack when it runs."
    return 0
  fi
  read_real_aws_conf "${REAL_AWS_CONF}" || exit 1
  echo "Real AWS is on (backend/devserver/real-aws.conf): Clutter's account ${REAL_AWS_ACCOUNT_ID}, region ${REAL_AWS_REGION}."
  if identity="$(aws --profile "${REAL_AWS_PROFILE}" --region "${REAL_AWS_REGION}" sts get-caller-identity --query Arn --output text 2>&1)" &&
    [[ "${identity}" == "arn:aws:sts::${REAL_AWS_ACCOUNT_ID}:assumed-role/${REAL_AWS_RUNNER_ROLE}/"* ]]; then
    echo "  ok    the profile ${REAL_AWS_PROFILE} acts as ${REAL_AWS_RUNNER_ROLE}"
  else
    echo "  FAIL  the profile ${REAL_AWS_PROFILE} does not act as ${REAL_AWS_RUNNER_ROLE} in ${REAL_AWS_ACCOUNT_ID}: ${identity}"
    echo "        Add the profile npm run real-aws -- setup printed to ~/.aws/config, or sign in again (aws sso login)."
    bad=1
  fi
  code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "${REAL_AWS_TEMPLATE_URL}" || true)"
  if [[ "${code}" == 200 ]]; then
    echo "  ok    CloudFormation can read the account-link template"
  else
    echo "  FAIL  the account-link template answers ${code:-nothing}: ${REAL_AWS_TEMPLATE_URL}"
    bad=1
  fi
  return "${bad}"
}

teardown() {
  local bucket
  echo "Removing the stack ${REAL_AWS_STACK} from the account of profile ${admin} ..."
  if bucket="$(stack_output BucketName 2>/dev/null)" && [[ -n "${bucket}" && "${bucket}" != None ]]; then
    admin_aws s3 rm --only-show-errors --recursive "s3://${bucket}"
  fi
  admin_aws cloudformation delete-stack --stack-name "${REAL_AWS_STACK}"
  admin_aws cloudformation wait stack-delete-complete --stack-name "${REAL_AWS_STACK}"
  rm -f "${REAL_AWS_CONF}"
  echo "Removed. AWS runs use LocalStack again after npm run dev."
}

"${action}"
