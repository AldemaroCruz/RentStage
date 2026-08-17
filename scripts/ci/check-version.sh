#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"
version="$(tr -d '[:space:]' < VERSION)"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "Invalid VERSION: $version" >&2; exit 1; }
web_version="$(python3 -c 'import json; print(json.load(open("apps/web/package.json"))["version"])')"
[[ "$web_version" == "$version" ]] || { echo "Web version $web_version does not match $version" >&2; exit 1; }
grep -Fq "RentStage Starter v${version}" README.md || { echo "README title does not contain v${version}" >&2; exit 1; }
echo "Version consistency passed: $version"
