#!/usr/bin/env bash
# Tests deploy.sh's safety checks with fake terraform, aws, git and docker, so nothing reaches AWS.
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
pass=0; fail=0
T="$(mktemp -d)"
trap 'rm -rf "${T}"' EXIT
mkdir -p "${T}/bin" "${T}/infra"

# terraform output answers from FAKE_API_URL, or FAKE_API_URL_AFTER once an apply has run.
cat > "${T}/bin/terraform" <<'EOF'
#!/usr/bin/env bash
case "$*" in
  *"output -raw api_invoke_url"*)
    url="${FAKE_API_URL:-}"
    if [[ -n "${FAKE_API_URL_AFTER:-}" ]] && grep -qE "^terraform .* apply " "${FAKE_LOG}"; then url="${FAKE_API_URL_AFTER}"; fi
    echo "terraform $*" >> "${FAKE_LOG}"
    [[ -n "${url}" ]] || exit 1
    echo "${url}" ;;
  *"output -raw load_balancer_dns_name"*)
    echo "terraform $*" >> "${FAKE_LOG}"
    echo "clutter-adapter-1234.ca-central-1.elb.amazonaws.com" ;;
  *) echo "terraform $*" >> "${FAKE_LOG}" ;;
esac
EOF
cat > "${T}/bin/aws" <<'EOF'
#!/usr/bin/env bash
# ecr describe-images finds only the repository:tag pairs in FAKE_ECR_TAGS.
if [[ "$*" == *"ecr describe-images"* ]]; then
  for t in ${FAKE_ECR_TAGS:-}; do
    [[ "$*" == *"--repository-name ${t%%:*} --image-ids imageTag=${t#*:}"* ]] && exit 0
  done
  exit 1
fi
[[ -n "${FAKE_ACCOUNT:-}" ]] || exit 1
echo "${FAKE_ACCOUNT}"
EOF
cat > "${T}/bin/git" <<'EOF'
#!/usr/bin/env bash
case "$*" in
  *rev-parse*) echo "${FAKE_BRANCH:-main}" ;;
  *status*) echo "${FAKE_DIRTY:-}" ;;
esac
EOF
printf '#!/usr/bin/env bash\necho "build" >> "${FAKE_LOG}"\n' > "${T}/bin/build"
cat > "${T}/bin/adapter-images" <<'EOF'
#!/usr/bin/env bash
echo "adapter-images ${AWS_REGION} ${IMAGE_TAG}" >> "${FAKE_LOG}"
[[ -z "${FAKE_PUSH_FAILS:-}" ]]
EOF
cat > "${T}/bin/site-image" <<'EOF'
#!/usr/bin/env bash
echo "site-image ${IMAGE_REPO} ${IMAGE_TAG} ${NEXT_PUBLIC_API_ENDPOINT} ${NEXT_PUBLIC_ADAPTER_API_ENDPOINT} ${NEXT_PUBLIC_GOOGLE_CLIENT_ID} ${NEXT_PUBLIC_CONTACT_EMAIL}" >> "${FAKE_LOG}"
[[ -z "${FAKE_SITE_PUSH_FAILS:-}" ]]
EOF
cat > "${T}/bin/docker" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == login ]]; then
  echo "docker-login $*" >> "${FAKE_LOG}"
  [[ -z "${FAKE_LOGIN_FAILS:-}" ]]
fi
EOF
chmod +x "${T}/bin/"*
# A's two runner scripts, in the layout deploy.sh expects under CLUTTER_A_SCRIPTS_DIR.
for image in terraform-deployer ansible-runner; do
  mkdir -p "${T}/ecr/${image}"
  cat > "${T}/ecr/${image}/build-and-push.sh" <<EOF
#!/usr/bin/env bash
echo "a-image ${image} \${AWS_REGION} \${IMAGE_TAG}" >> "\${FAKE_LOG}"
[[ -z "\${FAKE_A_PUSH_FAILS:-}" ]]
EOF
  chmod +x "${T}/ecr/${image}/build-and-push.sh"
done

FAKE_DB_PASSWORD="fake-db-password-0001"
FAKE_JWT="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
FAKE_API="https://abc123.execute-api.ca-central-1.amazonaws.com/prod"

# write_config environment: a complete config, made from the committed examples with fake real values.
# site_domain becomes clutter.test, and contact_email hello@clutter.test.
write_config() {
  sed -e 's/123456789012/111122223333/g' \
      -e 's/hello@example\.com/hello@clutter.test/' \
      -e "s/REPLACE_WITH_64_HEX_CHARACTERS/${FAKE_JWT}/" \
      -e "s/postgres\.PROJECT-REF:PASSWORD/postgres.abcdefghijklmnop:${FAKE_DB_PASSWORD}/" \
      -e 's/app\.example\.com/clutter.test/' \
      -e 's/NUMBER-ID/123456789-abc123/' \
      -e 's/certificate\/REPLACE/certificate\/abcd-1234/' \
      -e 's/^adapter_image_tag = "v1"/adapter_image_tag = "v7"/' \
      -e 's/^site_image_tag = "v1"/site_image_tag = "v9"/' \
      -e 's/^\(terraform_deployer_image_tag *= *\)"v1"/\1"v3"/' \
      -e 's/^\(ansible_runner_image_tag *= *\)"v1"/\1"v4"/' \
      "${HERE}/prod.tfvars.example" > "${T}/infra/$1.tfvars"
  sed -e 's/tfstate-ca-central-1-REPLACE/tfstate-ca-central-1-0a1b2c3d/' \
      "${HERE}/prod.backend.hcl.example" > "${T}/infra/$1.backend.hcl"
}

# run <input> <args...>: runs deploy.sh with the fakes; output in $out, exit code in $code.
run() {
  local input="$1"; shift
  : > "${T}/log"
  out="$(printf '%s\n' "${input}" | PATH="${T}/bin:${PATH}" FAKE_LOG="${T}/log" CLUTTER_INFRA_DIR="${T}/infra" CLUTTER_BUILD_CMD="${T}/bin/build" CLUTTER_ADAPTER_IMAGES_CMD="${T}/bin/adapter-images" CLUTTER_SITE_IMAGE_CMD="${T}/bin/site-image" CLUTTER_A_SCRIPTS_DIR="${T}/ecr" "${HERE}/deploy.sh" "$@" 2>&1)"
  code=$?
}
applied() { grep -qE "^terraform .* apply " "${T}/log"; }
planned() { grep -qE "^terraform .* plan " "${T}/log"; }
pushed() { grep -q "^adapter-images " "${T}/log"; }
a_pushed() { grep -q "^a-image " "${T}/log"; }
site_pushed() { grep -q "^site-image " "${T}/log"; }
# first_of a b: which of two log prefixes comes first.
first_of() { grep -nE "^($1|$2)" "${T}/log" | head -1 | cut -d: -f2- | grep -qE "^$1"; }

check() { # name, condition
  if eval "$2"; then pass=$((pass + 1)); else fail=$((fail + 1)); echo "FAIL: $1 (exit ${code}): ${out}"; fi
}

export FAKE_ACCOUNT=111122223333
export FAKE_API_URL="${FAKE_API}"

run "" ;                      check "no arguments is refused" '[ $code -ne 0 ]'
run "" plan --help;           check "--help prints usage and touches nothing" '[ $code -eq 0 ] && [[ "$out" == usage:* ]] && [ ! -s "${T}/log" ]'
run "" apply staging;         check "unknown environment is refused" '[ $code -ne 0 ]'
run "dev" apply dev;          check "dev does not exist (ADR-015)" '[ $code -ne 0 ] && ! applied'
run "" destroy prod;          check "there is no destroy" '[ $code -ne 0 ] && ! applied'
run "" apply prod;            check "missing config is refused" '[ $code -ne 0 ] && [[ "$out" == *"prod.tfvars: missing"* ]]'

write_config prod
printf 'jwt_secret = "REPLACE_WITH_64_HEX_CHARACTERS"\n' >> "${T}/infra/prod.tfvars"
run "" apply prod;            check "example values are refused" '[ $code -ne 0 ] && [[ "$out" == *"jwt_secret still holds an example value"* ]] && ! applied'

write_config prod
FAKE_ACCOUNT=999999999999 run "" apply prod
check "wrong AWS account is refused" '[ $code -ne 0 ] && [[ "$out" == *"account 999999999999"* ]] && ! applied'
FAKE_ACCOUNT="" run "" apply prod
check "not logged in is refused" '[ $code -ne 0 ] && [[ "$out" == *"not logged in"* ]]'

run "" plan prod
check "plan changes nothing" '[ $code -eq 0 ] && planned && ! applied'
check "plan uses the environment files" 'grep -q "backend-config=prod.backend.hcl" "${T}/log" && grep -q "var-file=prod.tfvars" "${T}/log"'
FAKE_BRANCH=feature run "" plan prod
check "plan works from any branch" '[ $code -eq 0 ] && ! applied'

FAKE_BRANCH=feature run "prod" apply prod
check "prod only from main" '[ $code -ne 0 ] && ! applied'
FAKE_DIRTY="M file" run "prod" apply prod
check "prod only from a clean checkout" '[ $code -ne 0 ] && ! applied'
run "no" apply prod;          check "a wrong answer applies nothing" '[ $code -ne 0 ] && ! applied'
run "yes" apply prod;         check "prod needs the word prod" '[ $code -ne 0 ] && ! applied'
run "prod" apply prod;        check "prod applies from clean main" '[ $code -eq 0 ] && applied'
check "apply without image flags pushes no image" '! pushed && ! a_pushed && ! site_pushed'
check "apply names the load balancer for both DNS names" '[[ "$out" == *"clutter-adapter-1234.ca-central-1.elb.amazonaws.com"* ]]'
FAKE_API_URL="" run "prod" apply prod
check "the first apply works before A's API exists" '[ $code -eq 0 ] && applied'

run "" plan prod --adapter-images;  check "--adapter-images with plan is refused" '[ $code -ne 0 ] && ! pushed'
run "prod" apply prod --push; check "an unknown option is refused" '[ $code -ne 0 ] && ! applied'
run "no" apply prod --adapter-images
check "--adapter-images pushes nothing unless confirmed" '[ $code -ne 0 ] && ! pushed && ! applied'
run "prod" apply prod --adapter-images
check "--adapter-images pushes the tfvars tag and region" '[ $code -eq 0 ] && grep -q "^adapter-images ca-central-1 v7$" "${T}/log"'
check "--adapter-images pushes before the apply" '[ "$(grep -nE "^adapter-images |^terraform .* apply " "${T}/log" | head -1 | cut -d: -f2 | cut -c1-14)" == "adapter-images" ] && applied'
FAKE_PUSH_FAILS=1 run "prod" apply prod --adapter-images
check "a failed push applies nothing" '[ $code -ne 0 ] && [[ "$out" == *"nothing applied"* ]] && ! applied'

run "" plan prod --a-images;  check "--a-images with plan is refused" '[ $code -ne 0 ] && ! a_pushed'
run "no" apply prod --a-images
check "--a-images pushes nothing unless confirmed" '[ $code -ne 0 ] && ! a_pushed && ! applied'
run "prod" apply prod --a-images
check "--a-images pushes each image with its tfvars tag and the region" '[ $code -eq 0 ] && grep -q "^a-image terraform-deployer ca-central-1 v3$" "${T}/log" && grep -q "^a-image ansible-runner ca-central-1 v4$" "${T}/log" && ! pushed'
check "--a-images pushes before the apply" 'first_of a-image "terraform .* apply " && applied'
FAKE_ECR_TAGS="clutter-terraform-deployer:v3" run "prod" apply prod --a-images
check "--a-images skips a tag already in ECR" '[ $code -eq 0 ] && ! grep -q "^a-image terraform-deployer" "${T}/log" && grep -q "^a-image ansible-runner ca-central-1 v4$" "${T}/log" && applied'
FAKE_A_PUSH_FAILS=1 run "prod" apply prod --a-images
check "a failed A push applies nothing" '[ $code -ne 0 ] && [[ "$out" == *"nothing applied"* ]] && ! applied'
run "prod" apply prod --a-images --adapter-images
check "both image flags push both" '[ $code -eq 0 ] && a_pushed && pushed && applied'
run "prod" apply prod --adapter-images --adapter-images --x
check "an unknown option after a known one is refused" '[ $code -ne 0 ] && ! applied'

# --- --site-image ---
# B's address is the site's own origin: the load balancer sends /v1/* and /healthz to B.
site_line="site-image 111122223333.dkr.ecr.ca-central-1.amazonaws.com/clutter-website v9 ${FAKE_API} https://clutter.test 123456789-abc123.apps.googleusercontent.com hello@clutter.test"
site_built_right() { grep -qxF "${site_line}" "${T}/log"; }
run "" plan prod --site-image; check "--site-image with plan is refused" '[ $code -ne 0 ] && ! site_pushed && ! planned'
run "no" apply prod --site-image
check "--site-image pushes nothing unless confirmed" '[ $code -ne 0 ] && ! site_pushed && ! applied'
run "prod" apply prod --site-image
check "--site-image builds with A's API output, the site's origin for B, google_client_id, contact_email and site_image_tag" '[ $code -eq 0 ] && site_built_right && ! pushed && ! a_pushed'
check "--site-image logs in to the account's ECR first" 'grep -q "^docker-login login --username AWS --password-stdin 111122223333.dkr.ecr.ca-central-1.amazonaws.com$" "${T}/log" && first_of docker-login site-image'
check "--site-image pushes before the apply" 'first_of site-image "terraform .* apply " && applied'
FAKE_API_URL="" run "prod" apply prod --site-image
check "--site-image before A's API exists changes nothing" '[ $code -ne 0 ] && [[ "$out" == *"without --site-image"* ]] && ! planned && ! site_pushed && ! applied'
FAKE_API_URL="http://abc123.example/prod" run "prod" apply prod --site-image
check "--site-image refuses an API address that is not https" '[ $code -ne 0 ] && ! site_pushed && ! applied'
FAKE_ECR_TAGS="clutter-website:v9" run "prod" apply prod --site-image
check "--site-image skips a tag already in ECR" '[ $code -eq 0 ] && ! site_pushed && ! grep -q "^docker-login" "${T}/log" && [[ "$out" == *"clutter-website:v9 is already in ECR"* ]] && applied'
FAKE_SITE_PUSH_FAILS=1 run "prod" apply prod --site-image
check "a failed website push applies nothing" '[ $code -ne 0 ] && [[ "$out" == *"nothing applied"* ]] && ! applied'
FAKE_LOGIN_FAILS=1 run "prod" apply prod --site-image
check "a failed ECR log-in builds nothing and applies nothing" '[ $code -ne 0 ] && ! site_pushed && ! applied'
FAKE_API_URL_AFTER="https://new456.execute-api.ca-central-1.amazonaws.com/prod" run "prod" apply prod --site-image
check "an API address that the apply changed is named" '[ $code -eq 0 ] && [[ "$out" == *"holds the old one"* ]]'
run "prod" apply prod --a-images --adapter-images --site-image
check "all three image flags push all three, then apply" '[ $code -eq 0 ] && a_pushed && pushed && site_pushed && applied'
sed -i.bak '/^site_image_tag/d' "${T}/infra/prod.tfvars"
run "prod" apply prod --site-image
check "--site-image without site_image_tag is refused" '[ $code -ne 0 ] && ! site_pushed && ! applied'
write_config prod
sed -i.bak '/^contact_email/d' "${T}/infra/prod.tfvars"
run "prod" apply prod --site-image
check "--site-image without contact_email is refused" '[ $code -ne 0 ] && ! site_pushed && ! applied'
write_config prod

# --- check-config.sh, run directly ---

# cc <args...>: runs check-config.sh on the temp files; output in $out, exit code in $code.
cc() {
  out="$(CLUTTER_INFRA_DIR="${T}/infra" "${HERE}/check-config.sh" "$@" 2>&1)"
  code=$?
}
no_secret() { [[ "$out" != *"${FAKE_DB_PASSWORD}"* && "$out" != *"${FAKE_JWT}"* ]]; }

write_config prod
cc prod;                      check "config: a complete config passes" '[ $code -eq 0 ] && [[ "$out" == *"config: complete"* ]] && no_secret'
cc;                           check "config: no environment is a usage error" '[ $code -eq 2 ]'
cc prod --platform;           check "config: an extra argument is a usage error" '[ $code -eq 2 ]'

sed -i.bak '/^certificate_arn/d' "${T}/infra/prod.tfvars"
cc prod;                      check "config: a missing key is named" '[ $code -ne 0 ] && [[ "$out" == *"certificate_arn is not set"* ]]'

write_config prod
printf 'psql_connection_string = "mysql://u:%s@db"\n' "${FAKE_DB_PASSWORD}" >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: a wrong form is named, never printed" '[ $code -ne 0 ] && [[ "$out" == *"psql_connection_string is not a postgres"* ]] && no_secret'

write_config prod
printf 'site_domain = "https://clutter.test"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: a site_domain with a scheme is refused" '[ $code -ne 0 ] && [[ "$out" == *"site_domain is not a lower-case DNS name"* ]]'

write_config prod
printf 'site_domain = "clutter.test/app"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: a site_domain with a path is refused" '[ $code -ne 0 ] && [[ "$out" == *"site_domain is not a lower-case DNS name"* ]]'

write_config prod
printf 'contact_email = "hello@example.com"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: the example contact_email is refused" '[ $code -ne 0 ] && [[ "$out" == *"contact_email still holds an example value"* ]]'

write_config prod
printf 'contact_email = "hello at clutter.test"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: a contact_email that is not an address is refused" '[ $code -ne 0 ] && [[ "$out" == *"contact_email is not an email address"* ]]'

write_config prod
printf 'certificate_arn = "arn:aws:acm:us-east-1:111122223333:certificate/abcd-1234"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: a certificate in another region is refused" '[ $code -ne 0 ] && [[ "$out" == *"not in aws_region"* ]]'

write_config prod
printf 'adapter_image_tag = "latest"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: the latest tag is refused" '[ $code -ne 0 ] && [[ "$out" == *"adapter_image_tag is not a version tag"* ]]'

write_config prod
printf 'site_image_tag = "latest"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: the latest website tag is refused" '[ $code -ne 0 ] && [[ "$out" == *"site_image_tag is not a version tag"* ]]'

write_config prod
printf 'terraform_deployer_image_tag = "latest"\n' >> "${T}/infra/prod.tfvars"
cc prod;                      check "config: the latest deployer tag is refused" '[ $code -ne 0 ] && [[ "$out" == *"terraform_deployer_image_tag is not a version tag"* ]]'

write_config prod
sed -i.bak '/^terraform_deployer_image_tag/d' "${T}/infra/prod.tfvars"
cc prod;                      check "config: a missing deployer tag is named" '[ $code -ne 0 ] && [[ "$out" == *"terraform_deployer_image_tag is not set"* ]]'

write_config prod
rm -f "${T}/infra/prod.backend.hcl"
cc prod;                      check "config: a missing backend file is named" '[ $code -ne 0 ] && [[ "$out" == *"prod.backend.hcl: missing"* ]]'

# --- main.tf: A's API deployment waits for every route ---
# A deployment made while a method lacks its integration fails, so every integration and CORS module
# is in the depends_on of aws_api_gateway_deployment.clutter-api.
code=0; out=""
routes="$(awk '/^module "/{n=$2; gsub(/"/,"",n)} /source *= *"\.\/modules\/templates\/api-(lambda-integration|path-cors-compliance)"/{print n}' "${HERE}/main.tf")"
deps="$(awk '/^resource "aws_api_gateway_deployment"/{d=1} d&&/depends_on/{l=1} d&&l{print} d&&l&&/^  \]/{exit}' "${HERE}/main.tf")"
missing=""
for m in ${routes}; do
  grep -q "module\.${m}," <<<"${deps}" || missing+=" ${m}"
done
out="missing:${missing}"
check "main.tf: the API deployment waits for every route module" '[ -n "${routes}" ] && [ -z "${missing}" ]'

echo "deploy checks: ${pass} passed, ${fail} failed"
[ "${fail}" -eq 0 ]
