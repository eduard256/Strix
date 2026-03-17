---
name: release_strix_dev
description: Build and push dev Docker image for Strix, update hassio-strix dev add-on version.
disable-model-invocation: true
---

# Strix Dev Build

You are building and pushing a dev image of Strix. Follow every step exactly. Do NOT ask any questions -- this is fully automated.

## Repositories

- Strix: `/home/user/Strix`
- hassio-strix: `/home/user/hassio-strix`

## Step 1: Get commit hash

```bash
cd /home/user/Strix
git rev-parse --short HEAD
```

Store this as COMMIT_HASH (e.g. `fe93aa3`).

## Step 2: Build Docker image

```bash
cd /home/user/Strix
docker build --build-arg VERSION=dev-$COMMIT_HASH -t eduard256/strix:dev -t eduard256/strix:dev-$COMMIT_HASH .
```

## Step 3: Push to Docker Hub

```bash
docker push eduard256/strix:dev
docker push eduard256/strix:dev-$COMMIT_HASH
```

## Step 4: Update hassio-strix

```bash
cd /home/user/hassio-strix
git pull origin main
```

Edit `/home/user/hassio-strix/strix-dev/config.json` -- change `"version"` to `dev-$COMMIT_HASH`.

```bash
cd /home/user/hassio-strix
git add strix-dev/config.json
git commit -m "Dev build dev-$COMMIT_HASH"
git push origin main
```

## Step 5: Report

Output a summary:

```
Dev build complete:
- Commit: $COMMIT_HASH
- Docker Hub: eduard256/strix:dev, eduard256/strix:dev-$COMMIT_HASH (amd64)
- hassio-strix: strix-dev version updated to dev-$COMMIT_HASH
```
