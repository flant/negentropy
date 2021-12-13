# oidc provider with special endpoints to build mock of okta jwts, signed by key available at /keys

## Endpoints:

### /custom_id_token

#### Example :

```bash
curl  --request  GET http://localhost:9998/custom_id_token?uuid=xxxxxxx&aud=audience&aud=aud2&anykey=anyvalue
```

returns id_token, which has okta id_token fields, and has:  
"uuid":"xxxxxx"  
"aud":["audience","aud2"]  
"anykey":"anyvalue"

jwt is signed by key, exposed at http://localhost:9998/keys

### /custom_access_token && /userinfo

#### Example :

```bash
curl  --request GET 'http://localhost:9998/custom_access_token?uuid=XXXX-ZZZZ-YYYY&anykey=anyvalue'
```

returns access token, which:

1) has: "iss": "http://localhost:9998/",  
   "sub": "subject_2021-12-09 16:38:49.326517 +0300 MSK m=+7.683920879",  
   "aud": ["aud666"],
2) can be passed to /userinfo:

```bash
curl --request GET 'http://localhost:9998/userinfo' --header 'Authorization: Bearer {{token}} 
```

returns userinfo with:  
"sub": "subject_2021-12-09 16:38:49.326517 +0300 MSK m=+7.683920879",  
"uuid": "XXXX-ZZZZ-YYYY",  
"anykey":"anyvalue"

## RUN:

from e2e folder:  
```go run github.com/flant/negentropy/e2e/tests/lib/oidc_mock/cmd```

## BUILD & RUN docker container:

build: from e2e/tests/lib/oidc_mock:

```bash
docker build -t oidcmock .
```

run:

```bash
docker run -p 9998:9998 --name oidcmock oidcmock
```