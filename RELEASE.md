# Release Process Documentation

This document describes the complete release process for Strix and its Home Assistant add-on.

## Overview

Strix consists of two repositories:
- **Strix** - Main application (Go binary, Docker image)
- **hassio-strix** - Home Assistant add-on

Both repositories must be released together to maintain version consistency.

## Version Files

### Strix Repository

| File | Line | Purpose |
|------|------|---------|
| `cmd/strix/main.go` | 23 | Application version constant |
| `webui/package.json` | 3 | Frontend version |
| `CHANGELOG.md` | Top | Release notes |

### hassio-strix Repository

| File | Line | Purpose |
|------|------|---------|
| `strix/config.json` | 3 | Home Assistant add-on version |
| `strix/CHANGELOG.md` | Top | Release notes |

## Release Workflow

### Prerequisites

1. Ensure you're on the `develop` branch with all changes committed
2. All tests pass: `go test ./...`
3. Application builds successfully: `go build ./...`

### Step 1: Update Strix Repository

```bash
cd /home/user/Strix
git checkout develop

# 1. Update version in code
# Edit cmd/strix/main.go - line 23
#   Version = "1.0.X"

# 2. Update frontend version
# Edit webui/package.json - line 3
#   "version": "1.0.X"

# 3. Update CHANGELOG.md
# Add new section at the top:
#   ## [1.0.X] - YYYY-MM-DD
#   ### Fixed/Added/Changed
#   - Description of changes

# 4. Commit version bump
git add cmd/strix/main.go webui/package.json CHANGELOG.md
git commit -m "Release v1.0.X: Brief description"

# 5. Merge to main
git checkout main
git merge develop --no-ff -m "Merge develop into main for v1.0.X release"

# 6. Create tag
git tag -a v1.0.X -m "Release v1.0.X: Brief description"

# 7. Push everything
git push origin main --tags
git push origin develop
```

### Step 2: Update hassio-strix Repository

```bash
cd /home/user/hassio-strix

# 1. Update add-on version
# Edit strix/config.json - line 3
#   "version": "1.0.X"

# 2. Update CHANGELOG
# Edit strix/CHANGELOG.md - add same section as in Strix

# 3. Commit and push
git add strix/config.json strix/CHANGELOG.md
git commit -m "Release v1.0.X: Brief description"
git push origin main

# 4. Create tag
git tag -a v1.0.X -m "Release v1.0.X: Brief description"
git push origin --tags
```

### Step 3: Verify GitHub Actions

After pushing tags, GitHub Actions will automatically:

**Strix (eduard256/Strix):**
- ✅ `release.yml` - GoReleaser creates GitHub Release with binaries
- ✅ `docker.yml` - Builds and pushes Docker images to Docker Hub
  - Tags: `1.0.X`, `1.0`, `1`, `latest`
  - Platforms: `linux/amd64`, `linux/arm64`

**hassio-strix (eduard256/hassio-strix):**
- ✅ `builder.yml` - Builds and pushes HA add-on to GHCR
  - Image: `ghcr.io/eduard256/hassio-strix-{arch}`
  - Architectures: `amd64`, `aarch64`

Check status:
```bash
# Check Strix actions
gh run list --repo eduard256/Strix --limit 5

# Check hassio-strix actions
gh run list --repo eduard256/hassio-strix --limit 5
```

### Step 4: Verify Release

1. **GitHub Release**: https://github.com/eduard256/Strix/releases/tag/v1.0.X
   - Should have binaries for all platforms

2. **Docker Hub**: https://hub.docker.com/r/eduard256/strix/tags
   - Should show new version tag

3. **GHCR**: https://github.com/eduard256/hassio-strix/pkgs/container/hassio-strix-amd64
   - Should show new version

4. **Home Assistant Add-on Store**:
   - Users will see update notification automatically

## Quick Release Script

For automated release, use this script:

```bash
#!/bin/bash
# release.sh - Automated release script

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: ./release.sh 1.0.X"
    exit 1
fi

echo "🚀 Releasing v$VERSION"

# Strix repo
cd /home/user/Strix
git checkout develop

# Update versions
sed -i "s/Version = \".*\"/Version = \"$VERSION\"/" cmd/strix/main.go
sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" webui/package.json

# Update CHANGELOG (manual step reminder)
echo "⚠️  Please update CHANGELOG.md manually"
read -p "Press enter when done..."

# Commit and merge
git add cmd/strix/main.go webui/package.json CHANGELOG.md
git commit -m "Release v$VERSION"
git checkout main
git merge develop --no-ff -m "Merge develop into main for v$VERSION release"
git tag -a "v$VERSION" -m "Release v$VERSION"
git push origin main --tags
git push origin develop

# hassio-strix repo
cd /home/user/hassio-strix

# Update version
sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" strix/config.json

# Update CHANGELOG (manual step reminder)
echo "⚠️  Please update strix/CHANGELOG.md manually"
read -p "Press enter when done..."

# Commit and tag
git add strix/config.json strix/CHANGELOG.md
git commit -m "Release v$VERSION"
git push origin main
git tag -a "v$VERSION" -m "Release v$VERSION"
git push origin --tags

echo "✅ Release v$VERSION complete!"
echo "Check GitHub Actions:"
echo "  - https://github.com/eduard256/Strix/actions"
echo "  - https://github.com/eduard256/hassio-strix/actions"
```

## Troubleshooting

### Docker Build Fails

**Symptom**: "Docker Hub Description" step shows `Forbidden` error

**Cause**: Docker Hub Personal Access Token expired or lacks permissions

**Solution**:
1. Go to Docker Hub → Account Settings → Security → Personal Access Tokens
2. Create new token with `Read, Write, Delete` scopes
3. Update GitHub secret: https://github.com/eduard256/Strix/settings/secrets/actions
4. Update `DOCKER_TOKEN` secret

**Note**: This only affects README updates on Docker Hub. Docker images still build successfully.

### Home Assistant Add-on Build Fails

**Symptom**: Builder workflow fails

**Solution**:
1. Check `strix/Dockerfile` for correct Go version
2. Verify `strix/build.yaml` has correct base images
3. Check logs: `gh run view <run-id> --repo eduard256/hassio-strix --log`

### Version Mismatch

**Symptom**: Strix shows different version than add-on

**Solution**:
1. Verify both `cmd/strix/main.go` and `strix/config.json` have same version
2. Ensure both repositories have matching tags: `git tag -l`
3. If needed, delete wrong tag: `git tag -d v1.0.X && git push origin :refs/tags/v1.0.X`

## Release Checklist

Before releasing:
- [ ] All changes merged to `develop`
- [ ] Tests pass: `go test ./...`
- [ ] Build succeeds: `go build ./...`
- [ ] CHANGELOG.md updated in both repos
- [ ] Version numbers match in all files
- [ ] Commit messages don't mention AI/Claude Code

After releasing:
- [ ] GitHub Release created with binaries
- [ ] Docker images pushed to Docker Hub
- [ ] HA add-on images pushed to GHCR
- [ ] Home Assistant users can see update

## Version Numbering

Strix follows [Semantic Versioning](https://semver.org/):

- **MAJOR.MINOR.PATCH** (e.g., 1.0.9)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes

## GitHub Actions Overview

### Strix Repository

**Triggers:**
- `release.yml`: On push tag `v*`
- `docker.yml`: On push tag `v*` or push to `main`
- `ci.yml`: On push to any branch

**Secrets Required:**
- `GITHUB_TOKEN` (automatic)
- `DOCKER_USERNAME`
- `DOCKER_TOKEN`

### hassio-strix Repository

**Triggers:**
- `builder.yml`: On push tag `v*` or manual workflow dispatch

**Secrets Required:**
- `GITHUB_TOKEN` (automatic)

## Notes

- Always release both repositories together
- Keep version numbers synchronized
- Test in Home Assistant Ingress mode before releasing
- Docker/direct mode should work without changes (backward compatibility)
