# rancher-deploy

**WARN** Rancher 1.6 Only

## Prepare
Define RANCHER_ACCESS_KEY and RANCHER_SECRET_KEY environment variables

## Run
```bash
bash <( curl -sL https://git.io/rancher-deploy ) \
    -rancher-url https://cloud.example.com \
    -service some-stack/some-service \
    -docker-image username/conainer:latest \
    -env some-env
```
