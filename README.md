# Cloudflare Sync

created to keep my raspberry pi sub domains updated.

## Introduction

golang application to update dns records on cloudflare, you can customize the frequency of updates (default for every hour) and if the dns record will be proxied (just for new records).

### Usage

Available environment variables:

```sh
CLOUDFLARE_API_TOKEN=<key>
CLOUDFLARE_DOMAIN=example.com
CLOUDFLARE_SUB_DOMAINS=test,home
CRON=*/2 * * * *
PROXIED=true
```

```sh
docker run --name cloudflaresync --env CLOUDFLARE_API_TOKEN=<key> --env CLOUDFLARE_SUB_DOMAINS=<app,home> --env CLOUDFLARE_DOMAIN=<example.com> --restart unless-stopped niradler/cloudflaresync:armv7
```
