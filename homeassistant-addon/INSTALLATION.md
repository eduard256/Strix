# Home Assistant Add-on Installation Guide

This guide explains how to set up and publish the Strix Home Assistant Add-on.

## 📋 Overview

The add-on structure is ready and includes:
- ✅ `config.yaml` - Add-on configuration
- ✅ `Dockerfile` - Multi-arch Docker build
- ✅ `build.yaml` - Build configuration for aarch64/amd64/armv7
- ✅ `run.sh` - Startup script with HA integration
- ✅ `README.md` - User-facing documentation
- ✅ `DOCS.md` - Comprehensive usage guide
- ✅ `CHANGELOG.md` - Version history
- ✅ GitHub Actions workflow for automated builds

## 🚀 Quick Deployment

### Step 1: Enable GitHub Actions

The `.github/workflows/addon.yml` workflow will automatically:
1. Build binaries for all architectures (aarch64, amd64, armv7)
2. Create multi-arch Docker images
3. Push to GitHub Container Registry (ghcr.io)
4. Update version numbers on tags

No manual setup needed - just push to GitHub!

### Step 2: Create First Release

```bash
# Make sure all changes are committed
git add .
git commit -m "Add Home Assistant Add-on"

# Push to main branch (this will trigger a dev build)
git push origin main

# Create and push a version tag (this will trigger a release build)
git tag v1.0.0
git push origin v1.0.0
```

The GitHub Action will automatically build and publish Docker images to:
- `ghcr.io/eduard256/strix-addon-aarch64:latest`
- `ghcr.io/eduard256/strix-addon-amd64:latest`
- `ghcr.io/eduard256/strix-addon-armv7:latest`

### Step 3: Add Icons (Optional but Recommended)

Add these files to `homeassistant-addon/`:
- `icon.png` - 128x128px icon for the add-on store
- `logo.png` - 256x256px logo for the add-on page

Recommended: Simple owl or camera icon in Home Assistant style (blue/white theme).

### Step 4: Test Installation

After the GitHub Action completes:

1. In Home Assistant, go to **Supervisor** → **Add-on Store**
2. Click **⋮** (menu) → **Repositories**
3. Add: `https://github.com/eduard256/Strix`
4. Find "Strix Camera Discovery" in the store
5. Click **Install**
6. Configure and start the add-on
7. Click **Open Web UI**

## 🔄 Updating the Add-on

### For New Features/Fixes

```bash
# Make your changes to the codebase
git add .
git commit -m "feat: add new feature"
git push origin main
```

The dev build will automatically trigger, creating images tagged with `dev-<git-hash>`.

### For New Releases

```bash
# Update version in homeassistant-addon/config.yaml
sed -i 's/^version:.*/version: "1.1.0"/' homeassistant-addon/config.yaml

# Update CHANGELOG.md
# Add new version section

# Commit and tag
git add homeassistant-addon/config.yaml homeassistant-addon/CHANGELOG.md
git commit -m "release: v1.1.0"
git tag v1.1.0
git push origin main
git push origin v1.1.0
```

The release build will automatically create versioned images.

## 📦 Manual Build (Optional)

If you need to build locally for testing:

```bash
# Build for your architecture
cd homeassistant-addon

# Build the Go binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w" \
  -o strix \
  ../cmd/strix/main.go

# Copy required files
cp -r ../data .
cp -r ../webui .

# Build Docker image
docker build \
  --build-arg BUILD_FROM=ghcr.io/home-assistant/amd64-base:3.20 \
  -t strix-addon:test .

# Test the image
docker run --rm -p 4567:4567 strix-addon:test
```

## 🔧 Configuration Options

Users can configure the add-on through the Home Assistant UI:

### Default Configuration
```yaml
log_level: info
port: 4567
strict_validation: true
```

### Advanced Options

Edit `homeassistant-addon/config.yaml` to add more options:

```yaml
options:
  log_level: info
  port: 4567
  strict_validation: true
  # Add new options here

schema:
  log_level: list(debug|info|warn|error)
  port: port
  strict_validation: bool
  # Add new option schemas here
```

Then update `run.sh` to use the new options:

```bash
NEW_OPTION=$(bashio::config 'new_option')
export STRIX_NEW_OPTION="${NEW_OPTION}"
```

## 🌐 Publishing to Community

### Option 1: Keep as Custom Repository (Recommended for Start)

Users add your repository manually:
```
https://github.com/eduard256/Strix
```

**Pros:**
- Full control
- Faster updates
- No approval process

**Cons:**
- Users must add repository manually
- Not in official add-on store

### Option 2: Submit to Home Assistant Community Add-ons

To get listed in the official community store:

1. Follow Home Assistant Add-on guidelines:
   https://developers.home-assistant.io/docs/add-ons/

2. Submit to Community Add-ons repository:
   https://github.com/home-assistant/addons

3. Wait for review and approval

**Pros:**
- Official recognition
- Easier for users to find
- Auto-update support

**Cons:**
- Strict guidelines
- Review process
- Slower updates

## 📊 Monitoring Builds

Check GitHub Actions status:
```
https://github.com/eduard256/Strix/actions
```

View published images:
```
https://github.com/eduard256/Strix/pkgs/container/strix-addon-amd64
https://github.com/eduard256/Strix/pkgs/container/strix-addon-aarch64
https://github.com/eduard256/Strix/pkgs/container/strix-addon-armv7
```

## 🐛 Troubleshooting

### Build Fails

Check GitHub Actions logs for errors. Common issues:
- Go build errors → Fix in main codebase
- Docker build errors → Check Dockerfile
- Permission errors → Ensure GITHUB_TOKEN has required permissions

### Add-on Won't Install

- Verify config.yaml syntax
- Check Docker images are published to ghcr.io
- Ensure repository.yaml is in root directory
- Verify architecture support matches user's system

### Add-on Won't Start

Check add-on logs in Home Assistant:
- Go to add-on page → **Log** tab
- Look for startup errors
- Common issues:
  - Port already in use
  - Missing data files
  - Permission errors

## 📚 Resources

- [Home Assistant Add-on Documentation](https://developers.home-assistant.io/docs/add-ons/)
- [Add-on Configuration](https://developers.home-assistant.io/docs/add-ons/configuration)
- [Add-on Testing](https://developers.home-assistant.io/docs/add-ons/testing)
- [GitHub Actions](https://docs.github.com/en/actions)

## ✅ Checklist

Before first release:
- [ ] All code tested and working
- [ ] Version set in config.yaml
- [ ] CHANGELOG.md updated
- [ ] Icons added (icon.png, logo.png)
- [ ] README.md reviewed
- [ ] DOCS.md reviewed
- [ ] GitHub Actions workflow tested
- [ ] Repository URL correct in config files
- [ ] Git tag created (v1.0.0)
- [ ] Docker images published to ghcr.io
- [ ] Test installation in Home Assistant

## 🎉 You're Ready!

Once you've completed the checklist above, your Home Assistant Add-on is ready for users!

Share it with the community:
- Home Assistant Forums
- Reddit r/homeassistant
- GitHub Discussions
- Discord servers
