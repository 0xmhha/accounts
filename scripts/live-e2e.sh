#!/usr/bin/env bash
#
# live-e2e.sh — one-shot reproducible live end-to-end test.
#
# Builds gstable (if needed), boots a chainbench go-stablenet network, runs the
# accounts SDK e2e (keystore decrypt -> create -> sign 0x00/0x01/0x02/0x16 ->
# submit -> verify on-chain), then tears the network down.
#
# Paths are overridable via environment variables so this works across machines.
set -euo pipefail

ACCOUNTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# --- configurable locations -------------------------------------------------
GOSTABLENET_DIR="${GOSTABLENET_DIR:-$(cd "$ACCOUNTS_DIR/../go-stablenet" 2>/dev/null && pwd || echo "")}"
CHAINBENCH_DIR="${CHAINBENCH_DIR:-/Users/kevin/work/github/0xmhha/auto-coding/chainbench}"
RPC="${RPC:-http://127.0.0.1:8505}"
PASSWORD="${KEYSTORE_PASSWORD:-1}"
# ---------------------------------------------------------------------------

fail() { echo "ERROR: $*" >&2; exit 1; }

[ -n "$GOSTABLENET_DIR" ] && [ -d "$GOSTABLENET_DIR" ] || fail "go-stablenet not found (set GOSTABLENET_DIR)"
[ -x "$CHAINBENCH_DIR/chainbench.sh" ] || fail "chainbench.sh not found (set CHAINBENCH_DIR)"

BIN="$GOSTABLENET_DIR/build/bin/gstable"
CB="$CHAINBENCH_DIR/chainbench.sh"

# 1. Build gstable if missing.
if [ ! -x "$BIN" ]; then
  echo ">> building gstable ..."
  ( cd "$GOSTABLENET_DIR" && GOTOOLCHAIN=local make gstable )
fi

# 2. Resolve a funded preset keystore.
KEYSTORE="${KEYSTORE:-$(ls "$CHAINBENCH_DIR"/keys/preset/node1/keystore/* 2>/dev/null | head -1 || true)}"
[ -n "$KEYSTORE" ] && [ -f "$KEYSTORE" ] || fail "funded keystore not found (set KEYSTORE)"

cleanup() { echo ">> stopping network ..."; "$CB" stop --quiet >/dev/null 2>&1 || true; }
trap cleanup EXIT

# 3. Boot the network with our binary.
echo ">> init + start chainbench (binary: $BIN) ..."
"$CB" init  --binary-path "$BIN" --quiet
"$CB" start --binary-path "$BIN" --quiet

# 4. Wait for RPC to answer.
echo ">> waiting for RPC $RPC ..."
for i in $(seq 1 30); do
  if curl -s -X POST "$RPC" -H 'Content-Type: application/json' \
      --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
      --max-time 2 | grep -q result; then
    echo ">> RPC ready"
    break
  fi
  [ "$i" = 30 ] && fail "RPC did not become ready"
  sleep 1
done

# 5. Run the SDK e2e.
echo ">> running accounts SDK e2e ..."
( cd "$ACCOUNTS_DIR" && GOTOOLCHAIN=local go run ./cmd/e2e -rpc "$RPC" -keystore "$KEYSTORE" -password "$PASSWORD" )

echo ">> live e2e finished"
