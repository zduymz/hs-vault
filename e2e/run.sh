#!/bin/bash

echo "Running e2e tests..."

echo "Start Vault 1"
docker run -d --name vault1 --cap-add=IPC_LOCK -e VAULT_DEV_ROOT_TOKEN_ID=root \
  -e VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200 -p8201:8200 \
  hashicorp/vault 2>/dev/null || docker start vault1

echo "Start Vault 2"
docker run -d --name vault2 --cap-add=IPC_LOCK -e VAULT_DEV_ROOT_TOKEN_ID=root \
  -e VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200 -p8202:8200 \
  hashicorp/vault 2>/dev/null || docker start vault2

FAILURES=""
for TEST_FILE in ./e2e/test_*.sh; do
    echo
    echo "🧪 Running test '$TEST_FILE'"

    "$TEST_FILE"
    if [[ "$?" -ne 0 ]]; then
        FAILURES="${FAILURES}  ⛔ ${TEST_FILE}\n"
    fi
done

docker stop vault1 vault2

if [[ -n "$FAILURES" ]]; then
    echo
    echo "Failed tests:"
    printf "$FAILURES"
    exit 1
fi
