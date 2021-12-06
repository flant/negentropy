# oidc provider with special endpoint (/custom_token) to build mock of jwt okta id_token, signed by key available at /keys

## Example:

GET http://localhost:9998/custom_token?uuid=xxxxxxx&aud=audience&aud=aud2&anykey=anyvalue

returns id_token, which has okta fields, and:  
uuid=xxxxxx  
aud=[audience,aud2]  
anykey=anyvalue

jwt is signed by key, exposed at http://localhost:9998/keys

## RUN:

from e2e folder:  
```CAOS_OIDC_DEV=1 go run github.com/flant/negentropy/e2e/tests/lib/oidc_mock/cmd```