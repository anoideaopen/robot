#!/bin/sh

# init vault
vault secrets enable -path=kv -version=1 kv & vault server -dev

# load crypto to the Vault
cd "/vault/dev-data"
for f in $(find * -path '*/crypto/*' -type f);
  do
    key="/kv/backend/${f#*/}" # exclude hlf-test-stage-xx
    base64 $f | vault kv put $key data=- # base64
#    vault write $key value=@$f # text
done
