# Cloudflare Sync

Created to keep my raspberry pi sub domains updated with my home ip address.

## Introduction

Golang application to create new/update dns records on cloudflare, you can customize the frequency of updates (default for every hour) and if the dns record will be proxied (just for new records).

### Usage

Available environment variables:

```sh
CLOUDFLARE_API_TOKEN=<key>
CLOUDFLARE_DOMAIN=example.com
CLOUDFLARE_SUB_DOMAINS=test,home
CRON=*/2 * * * *
PROXIED=true
DOCKER_DAEMON=true
```

### RaspberryPI

```sh
docker run --name cloudflaresync --env CLOUDFLARE_API_TOKEN=<key> --env CLOUDFLARE_SUB_DOMAINS=<app,home> --env CLOUDFLARE_DOMAIN=<example.com> --restart unless-stopped niradler/cloudflaresync:armv7
```

### Docker labels

Automatically create dns record, set env DOCKER_DAEMON=true, and expose docker socket /var/run/docker.sock:/var/run/docker.sock

```
version: "3.5"

services:
  cloudflaresync:
    image: niradler/cloudflaresync:latest
    restart: unless-stopped
    environment:
      CLOUDFLARE_API_TOKEN: "${CLOUDFLARE_API_TOKEN}"
      CLOUDFLARE_DOMAIN: "${DOMAIN}"
      CLOUDFLARE_SUB_DOMAINS: "${SUB_DOMAINS}"
      DOCKER_DAEMON: 'true'
    labels:
      - homepage.show=true
      - homepage.description=cloudflaresync
      - homepage.title=cloudflaresync
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

```
version: "3.5"

services:
  adminer:
    image: adminer:latest
    labels:
      - cloudflaresync.name=adminer
```
