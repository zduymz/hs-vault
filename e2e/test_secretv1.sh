export VAULT_TOKEN=root
export VAULT_ADDR="http://localhost:8201"
vault secrets enable -version=1 kv
vault kv put kv/key1 k=1
vault kv put kv/key2/a k=2

./dist/hs-vault backup -p kv -d /tmp


export VAULT_ADDR="http://localhost:8202"
vault secrets enable -version=1 kv
./dist/hs-vault restore -p kv -s /tmp/kv.kv
RESULT=$(vault kv get -format=json kv/key1 | jq -r '.data.k')
./e2e/verify.sh "$RESULT" "1"
RESULT=$(vault kv get -format=json kv/key2/a | jq -r '.data.k')
./e2e/verify.sh "$RESULT" "2"

