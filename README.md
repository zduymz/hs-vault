# hs-vault

**You don't need it in most cases. Use it with your own risk.**

## Prequsites
+ Enabled [raw](https://www.vaultproject.io/api-docs/system/raw) endpoint
+ Must use `root` token
## Features
+ backup most of the Vault engines using sys/raw endpoints (except Transit and SecretV2)
+ backup file was encoded by base64

## Build
```
make build
```

## Test
Run e2e tests:

```
make e2e
```
