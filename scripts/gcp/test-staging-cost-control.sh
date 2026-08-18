#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

fake_bin="$temp_dir/bin"
policy_file="$temp_dir/policy"
mkdir -p "$fake_bin"
printf 'ALWAYS\n' > "$policy_file"

cat > "$fake_bin/gcloud" <<'FAKE_GCLOUD'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$1 $2 $3" == 'sql instances describe' ]]; then
  case "$*" in
    *'value(state)'*)
      if [[ "$(<"$COST_TEST_POLICY_FILE")" == 'NEVER' ]]; then
        echo 'SUSPENDED'
      else
        echo 'RUNNABLE'
      fi
      ;;
    *'value(settings.activationPolicy)'*)
      cat "$COST_TEST_POLICY_FILE"
      ;;
    *)
      echo 'Unexpected describe format.' >&2
      exit 3
      ;;
  esac
elif [[ "$1 $2 $3" == 'sql instances patch' ]]; then
  case "$*" in
    *'--activation-policy=NEVER'*) printf 'NEVER\n' > "$COST_TEST_POLICY_FILE" ;;
    *'--activation-policy=ALWAYS'*) printf 'ALWAYS\n' > "$COST_TEST_POLICY_FILE" ;;
    *) echo 'Missing activation policy.' >&2; exit 3 ;;
  esac
elif [[ "$1 $2 $3" == 'run services describe' ]]; then
  echo 'https://rentstage.example.test rentstage-revision-00001'
else
  echo "Unexpected gcloud invocation: $*" >&2
  exit 3
fi
FAKE_GCLOUD
chmod +x "$fake_bin/gcloud"

run_control() {
  env \
    PATH="$fake_bin:$PATH" \
    COST_TEST_POLICY_FILE="$policy_file" \
    GCP_PROJECT_ID='rentstage-test' \
    GCP_REGION='us-east1' \
    STAGING_DEPLOY_ENABLED="${STAGING_DEPLOY_ENABLED:-false}" \
    CONFIRM_STAGING_COST_OPERATION="${CONFIRM_STAGING_COST_OPERATION:-false}" \
    bash "$root/scripts/gcp/staging-cost-control.sh" "$1"
}

expect_failure() {
  local operation="$1"
  if run_control "$operation" > "$temp_dir/failure-output" 2>&1; then
    echo "Expected $operation to fail." >&2
    exit 1
  fi
}

status_output="$(run_control status)"
grep -Fq 'Cloud SQL activation policy: ALWAYS' <<< "$status_output"
[[ "$(<"$policy_file")" == 'ALWAYS' ]]

STAGING_DEPLOY_ENABLED=true
CONFIRM_STAGING_COST_OPERATION=true
expect_failure pause
[[ "$(<"$policy_file")" == 'ALWAYS' ]]

STAGING_DEPLOY_ENABLED=false
CONFIRM_STAGING_COST_OPERATION=false
expect_failure pause
[[ "$(<"$policy_file")" == 'ALWAYS' ]]

CONFIRM_STAGING_COST_OPERATION=true
pause_output="$(run_control pause)"
grep -Fq 'Cloud SQL activation policy: NEVER' <<< "$pause_output"
[[ "$(<"$policy_file")" == 'NEVER' ]]

resume_output="$(run_control resume)"
grep -Fq 'Cloud SQL activation policy: ALWAYS' <<< "$resume_output"
[[ "$(<"$policy_file")" == 'ALWAYS' ]]

expect_failure unsupported

echo 'Staging cost-control tests passed.'
