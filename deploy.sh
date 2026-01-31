#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

: "${IMG:=svelama/mutator:latest}"

echo "==> Deploying mutator"
(cd "${ROOT_DIR}" && IMG="${IMG}" make deploy)

echo "✅ Mutator deployment complete"
