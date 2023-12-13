export VAULT_TOKEN=root

export VAULT_ADDR="http://localhost:8201"
vault secrets enable totp
vault write totp/keys/key1 generate=true issuer=example.com account_name=admin@example.com > /dev/null
./dist/hs-vault backup -p totp -d /tmp

export VAULT_ADDR="http://localhost:8202"
vault secrets enable totp
./dist/hs-vault restore -p totp -s /tmp/totp.totp


CODE=$(VAULT_ADDR="http://localhost:8201" vault read -field=code totp/code/key1)
RESULT=$(VAULT_ADDR="http://localhost:8202" vault write -field=valid totp/code/key1 code="$CODE")

./e2e/verify.sh "$RESULT" "true"