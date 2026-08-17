#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

mapfile -t tracked < <(git ls-files 2>/dev/null || find . -type f -not -path './.git/*' -printf '%P\n')
blocked=()
for path in "${tracked[@]}"; do
  case "$path" in
    .env|.env.*) [[ "$path" == ".env.example" ]] || blocked+=("$path") ;;
    *.tfstate|*.tfstate.*|*.tfplan|*.dump|*.backup|*.p12|*.pfx|*.pem|*.key|gha-creds-*.json|application_default_credentials.json|service-account*.json|*-service-account.json)
      blocked+=("$path") ;;
  esac
done
if ((${#blocked[@]})); then
  printf 'Sensitive or generated files must not be committed:\n' >&2
  printf '  %s\n' "${blocked[@]}" >&2
  exit 1
fi

if grep -RIlE --exclude-dir=.git --exclude='.env.example' -- \
  '-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----|"type"[[:space:]]*:[[:space:]]*"service_account"|"private_key_id"[[:space:]]*:' . | grep -q .; then
  echo "A private key or Google service-account credential appears to be present." >&2
  exit 1
fi

echo "Sensitive-file policy passed."
