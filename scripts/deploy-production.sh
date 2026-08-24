#!/usr/bin/env bash
set -Eeuo pipefail

cat >&2 <<'EOF'
scripts/deploy-production.sh is deprecated and intentionally does not deploy.

Production now has separate App and Judge nodes. Use the node-specific helpers:
  scripts/deploy-judge-node.sh <vX.Y.Z>
  scripts/deploy-app-node.sh <vX.Y.Z>

The GitHub Deploy production workflow orchestrates Judge first, then App, and
performs the corresponding rollback handling. Do not use this legacy combined
entrypoint because it cannot safely model the external App override or the
node-owned Judge Compose/configuration.
EOF
exit 64
