---
name: release_strix
description: Full release of Strix -- merge develop to main, tag, build multiarch Docker image, build static binaries, push to Docker Hub, update hassio-strix, create GitHub Release with binaries attached.
disable-model-invocation: true
---

# Strix Release

You are performing a full release of Strix. Follow every step exactly. Do NOT skip steps. Do NOT ask for confirmation except where explicitly noted below.

## Repositories

- Strix: `/home/user/Strix`
- hassio-strix: `/home/user/hassio-strix`

## Versioning

Version is injected at build time via ldflags (`-X main.version=$VERSION`). There is NO hardcoded version in the source code -- `main.go` has `var version = "dev"` as default. The Dockerfile passes `--build-arg VERSION` which maps to the same ldflags. Do NOT edit `main.go` to set the version.

## Step 1: Gather information

```bash
cd /home/user/Strix
git checkout develop
git pull origin develop
git pull origin main

# Get last release tag
git tag --sort=-version:refname | head -1

# Show all commits since last release
git log main..develop --oneline

# Show changed files
git diff main..develop --stat
```

## Step 2: Ask for version (THE ONLY QUESTION)

Use AskUserQuestion to ask the user which version to release.

Show them:
- The last tag
- The list of commits from Step 1

Offer options:
- Next patch (e.g. 1.0.9 -> 1.0.10)
- Next minor (e.g. 1.0.9 -> 1.1.0)
- Next major (e.g. 1.0.9 -> 2.0.0)
- Other (user types custom version)

Wait for answer. Store the chosen version as VERSION (without "v" prefix).

## Step 3: Download latest camera database

```bash
cd /home/user/Strix
gh release download latest --repo eduard256/StrixCamDB --pattern "cameras.db" --clobber
```

Verify the database was downloaded:
```bash
ls -lh cameras.db
```

## Step 4: Verify build

```bash
cd /home/user/Strix
go test ./...
CGO_ENABLED=0 go build ./...
```

If tests or build fail -- STOP and report the error. Do not continue.

## Step 5: Update CHANGELOG.md

Read `/home/user/Strix/CHANGELOG.md`. Add a new section at the top (after the header lines), based on the commits from Step 1. Follow the existing format exactly:

```markdown
## [VERSION] - YYYY-MM-DD

### Added
- ...

### Fixed
- ...

### Changed
- ...
```

Use today's date. Categorize commits into Added/Fixed/Changed/Technical sections. Only include sections that have entries. Write clear, user-facing descriptions (not raw commit messages).

## Step 6: Git -- commit, merge, tag, push

```bash
cd /home/user/Strix
git add CHANGELOG.md
git commit -m "Release v$VERSION"

git checkout main
git merge develop --no-ff -m "Merge develop into main for v$VERSION release"
git tag v$VERSION

git push origin main --tags

git checkout develop
git merge main
git push origin develop
```

## Step 7: Build and push Docker image

```bash
cd /home/user/Strix
docker buildx build --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$VERSION \
  -t eduard256/strix:$VERSION \
  -t eduard256/strix:latest \
  -t eduard256/strix:$(echo $VERSION | cut -d. -f1-2) \
  -t eduard256/strix:$(echo $VERSION | cut -d. -f1) \
  --push .
```

## Step 8: Verify Docker Hub

```bash
curl -s "https://hub.docker.com/v2/repositories/eduard256/strix/tags/?page_size=10" | jq '.results[].name'
docker manifest inspect eduard256/strix:$VERSION | jq '.manifests[].platform'
```

Verify the new version tag exists and both amd64 and arm64 platforms are present.

## Step 9: Smoke test

```bash
docker run --rm -d --name strix-smoke-test -p 14567:4567 eduard256/strix:$VERSION
sleep 5
curl -s http://localhost:14567/api | jq '.version'
docker stop strix-smoke-test
```

Verify the health endpoint returns the correct version string.

## Step 10: Update hassio-strix

```bash
cd /home/user/hassio-strix
git pull origin main
```

Edit `/home/user/hassio-strix/strix/config.json` -- change `"version"` to the new VERSION.

Edit `/home/user/hassio-strix/strix/CHANGELOG.md` -- add the same CHANGELOG section as in Step 5.

```bash
cd /home/user/hassio-strix
git add strix/config.json strix/CHANGELOG.md
git commit -m "Release v$VERSION"
git push origin main
```

## Step 11: Build static binaries

Build static binaries for both platforms:

```bash
cd /home/user/Strix
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$VERSION" -o strix-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.version=$VERSION" -o strix-linux-arm64 .
```

Verify both binaries are statically linked:
```bash
file strix-linux-amd64 strix-linux-arm64
```

## Step 12: GitHub Release

Create the release and attach binaries:

```bash
cd /home/user/Strix
PREV_TAG=$(git tag --sort=-version:refname | sed -n '2p')
gh release create v$VERSION \
  --title "v$VERSION" \
  --notes "$(git log --oneline ${PREV_TAG}..v$VERSION)" \
  strix-linux-amd64 strix-linux-arm64
```

Clean up binaries after upload:
```bash
rm -f strix-linux-amd64 strix-linux-arm64
```

## Step 13: Final report

Output a summary:

```
Release v$VERSION complete:
- Git: tag v$VERSION pushed to main
- Docker Hub: eduard256/strix:$VERSION (amd64 + arm64)
- Health check: version "$VERSION" verified
- Binaries: strix-linux-amd64, strix-linux-arm64 attached to GitHub Release
- hassio-strix: config.json updated to $VERSION, pushed to main
- GitHub Release: <URL from gh release create>
```
