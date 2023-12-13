export VAULT_TOKEN=root

export VAULT_ADDR="http://localhost:8201"
vault secrets enable -path=kv2 kv-v2

vault kv put kv2/key1 v=1
vault kv put kv2/key1 v=2
vault kv put kv2/key1 v=3
vault kv put kv2/key1 v=4
vault kv put kv2/key1 v=5
vault kv destroy -versions=1 kv2/key1
vault kv delete -versions=3 kv2/key1
vault kv put kv2/key1 v=6
vault kv metadata put -delete-version-after=3h kv2/key1

vault kv put kv2/key2 v=1
vault kv put kv2/key2 v=2
vault kv put kv2/key2 v=3
vault kv put kv2/key2 v=4
vault kv put kv2/key2 v=5
vault kv put kv2/key2 v=6
vault kv metadata put -custom-metadata=h1=abc -custom-metadata=h2=123 -max-versions=3 kv2/key2
vault kv put kv2/key2 v=7
vault kv put kv2/key2 v=8
vault kv destroy -versions=6,8 kv2/key2

./dist/hs-vault backup -p kv2 -d /tmp

export VAULT_ADDR="http://localhost:8202"
vault secrets enable -path=kv2 kv-v2
# run restore command
./dist/hs-vault restore -p kv2 -s /tmp/kv2.kv2

# version 1 destroy, return should be null
RESULT=$(vault kv get -format=json -version=1 kv2/key1 | jq -r '.data.data')
./e2e/verify.sh "$RESULT" "null"

# version 3 deleted, return should be null
RESULT=$(vault kv get -format=json -version=3 kv2/key1 | jq -r '.data.data')
./e2e/verify.sh "$RESULT" "null"

# version 5, should return 5
RESULT=$(vault kv get -format=json -version=5 kv2/key1 | jq -r '.data.data.v')
./e2e/verify.sh "$RESULT" "5"

# metadata delete_version_after should be 3h0m0s
RESULT=$(vault kv metadata get -format=json  kv2/key1 | jq -r '.data.delete_version_after')
./e2e/verify.sh "$RESULT" "3h0m0s"

# metadata versions should return 6
RESULT=$(vault kv metadata get -format=json kv2/key1 | jq -r '.data.versions | length')
./e2e/verify.sh "$RESULT" "6"

# version 1 not existed in meatadata, so it should return "No value found at kv2/data/key2"
RESULT=$(vault kv get -format=json -version=1 kv2/key2 2>&1)
./e2e/verify.sh "$RESULT" "No value found at kv2/data/key2"

# version 6 destroyed, should return null
RESULT=$(vault kv get -format=json -version=6 kv2/key2 | jq -r '.data.data')
./e2e/verify.sh "$RESULT" "null"

# version 7 should return 7
RESULT=$(vault kv get -format=json -version=7 kv2/key2 | jq -r '.data.data.v')
./e2e/verify.sh "$RESULT" "7"

# metadata versions should return 3
RESULT=$(vault kv metadata get -format=json kv2/key2 | jq -r '.data.versions | length')
./e2e/verify.sh "$RESULT" "3"

# metadata custom_metadata.h1 should return abc
RESULT=$(vault kv metadata get -format=json kv2/key2 | jq -r '.data.custom_metadata.h1')
./e2e/verify.sh "$RESULT" "abc"

# metadata custom_metadata.h2 should return 123
RESULT=$(vault kv metadata get -format=json kv2/key2 | jq -r '.data.custom_metadata.h2')
./e2e/verify.sh "$RESULT" "123"
