export VAULT_TOKEN=root
export VAULT_ADDR="http://localhost:8201"
vault secrets enable transit
vault write -f transit/keys/key1 > /dev/null
export TRANSIT_SECRET_MSG1=$(vault write -field=ciphertext transit/encrypt/key1 plaintext=$(base64 <<< "this is first sky"))
vault write -f transit/keys/key1/rotate > /dev/null
export TRANSIT_SECRET_MSG2=$(vault write -field=ciphertext transit/encrypt/key1 plaintext=$(base64 <<< "this is second sky"))

./dist/hs-vault backup -p transit -d /tmp

export VAULT_ADDR="http://localhost:8202"
vault secrets enable transit
./dist/hs-vault restore -p transit -s /tmp/transit.transit

MSG1=$(vault write -field=plaintext transit/decrypt/key1 ciphertext=${TRANSIT_SECRET_MSG1} | base64 -d)
MSG2=$(vault write -field=plaintext transit/decrypt/key1 ciphertext=${TRANSIT_SECRET_MSG2} | base64 -d)

./e2e/verify.sh "$MSG1" "this is first sky"
./e2e/verify.sh "$MSG2" "this is second sky"

vault write -f transit/keys/key1/rotate > /dev/null
TRANSIT_SECRET_MSG3=$(vault write -field=ciphertext transit/encrypt/key1 plaintext=$(base64 <<< "this is third sky"))
MSG3=$(vault write -field=plaintext transit/decrypt/key1 ciphertext=${TRANSIT_SECRET_MSG3} | base64 -d)
./e2e/verify.sh "$MSG3" "this is third sky"