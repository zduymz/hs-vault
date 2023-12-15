# hs-vault

**You don't need it in most cases. Use it with your own risk.**

## Prequsites
+ Some Engines require [raw](https://www.vaultproject.io/api-docs/system/raw) enabled to fully backup
+ Use `root` token
## Features
+ backup and restore secret engines
+ base64 encoded output

## Limits
+ SecretV2 "deleted" value will be treated as "destroyed"  
+ 
| Engine   | /sys/raw access required |
|----------|:------------------------:|
| TOTP     |            ✅             |
| SecretV1 |            ❌             |
| SecretV2 |            ❌             |
| Transit  |            ❌             |
| Database |            ⚠️            |
| PKI      |            ✅             |
| AWS      |            ⚠️             |
| SSH      |⚠️|

⚠️ Require /sys/raw access to backup private key or password in configuration

## Build
```
make build
```

## Test
Run e2e tests:

```
make e2e
```
