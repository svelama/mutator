#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Undeploying mutator"
(cd "${ROOT_DIR}" && make undeploy)

echo "✅ Mutator undeploy complete"
