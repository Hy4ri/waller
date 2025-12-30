# GitHub Actions Setup for Waller

This document explains how to use the GitHub Actions workflows to speed up your build process.

## 🚀 Quick Start

Once you push these workflows to GitHub, builds will automatically run on:

- Every push to `main`, `master`, or `dev` branches
- Every pull request
- Manual trigger (workflow_dispatch)

### Download Pre-built Binaries

1. Go to the **Actions** tab in your GitHub repo
2. Click on the latest successful **Build Waller** workflow run
3. Scroll to **Artifacts** section
4. Download `waller-<commit-hash>` artifact
5. Extract and run!

```bash
# Example download using GitHub CLI
gh run download --name waller-<commit-sha>
chmod +x waller
./waller
```

## 📦 Workflows Included

### 1. `build.yml` - Main Build Workflow

**Triggers:** Push to main branches, PRs, manual
**What it does:**

- Builds Waller using Nix
- Caches Nix store and Go modules for faster builds
- Runs basic linting checks
- Uploads binary artifacts (available for 30 days)

**First build:** ~5-10 minutes  
**Cached builds:** ~1-3 minutes ⚡

### 2. `release.yml` - Automated Releases

**Triggers:** Version tags (e.g., `v0.1.0`), manual

**What it does:**

- Builds release binary
- Creates GitHub release with downloadable tarball
- Generates checksums
- Auto-generates release notes

**Create a release:**

```bash
git tag v0.1.0
git push origin v0.1.0
```

### 3. `cache-clean.yml` - Cache Maintenance

**Triggers:** Weekly (Sundays 2 AM UTC), manual

Cleans up old caches to prevent bloat.

## ⚡ Speed Optimizations

### Current Setup

- ✅ Nix store caching
- ✅ Go module caching
- ✅ Parallel jobs (build + lint)
- ✅ Incremental builds

### Optional: Enable Cachix (Recommended!)

**Cachix** provides shared binary caches across all your machines and CI runs.

#### Setup

1. **Create free Cachix account:**

   ```bash
   # Visit https://cachix.org and sign up
   ```

2. **Create cache named "waller":**

   ```bash
   cachix create waller
   ```

3. **Get auth token:**
   - Go to <https://app.cachix.org>
   - Navigate to your cache settings
   - Copy the auth token

4. **Add to GitHub Secrets:**
   - Go to your repo → Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `CACHIX_AUTH_TOKEN`
   - Value: (paste your token)

5. **Uncomment Cachix step in `.github/workflows/build.yml`:**

   ```yaml
   # Remove the # symbols from these lines:
   - name: Setup Cachix
     uses: cachix/cachix-action@v14
     with:
       name: waller
       authToken: '${{ secrets.CACHIX_AUTH_TOKEN }}'
   ```

6. **Use Cachix locally too:**

   ```bash
   cachix use waller
   
   # When building locally:
   nix build
   cachix push waller result
   ```

**Result:** All builds (local + CI) share the same cache! 🎉

## 🔧 Local Development Workflow

You don't need to wait for CI during development:

```bash
# Fast iteration during development
nix develop
go build -o waller .
./waller

# Full Nix build (for final testing)
nix build
./result/bin/waller
```

## 📊 Expected Build Times

| Scenario | Without Cache | With GitHub Cache | With Cachix |
|----------|---------------|-------------------|-------------|
| First build | 8-12 min | 8-12 min | 8-12 min |
| Rebuild (no changes) | 8-12 min | 1-3 min | 30-60 sec |
| Small code change | 8-12 min | 2-4 min | 1-2 min |
| Dependency update | 8-12 min | 5-7 min | 2-3 min |

## 🎯 CI/CD Best Practices

### Branch Protection

Require successful builds before merging:

1. Go to Settings → Branches
2. Add rule for `main`
3. Enable "Require status checks to pass"
4. Select "build" and "lint"

### Manual Triggers

Run builds manually:

1. Go to Actions tab
2. Select "Build Waller" workflow
3. Click "Run workflow"

### Download Latest Build

```bash
# Using GitHub CLI
gh run list --workflow=build.yml --limit 1
gh run download <run-id>

# Or use the GitHub web UI
```

## 🐛 Troubleshooting

### Build fails on first run

- Check `flake.lock` is committed
- Ensure `vendorHash` in `flake.nix` is correct
- Try rebuilding locally first

### Cache not working

- Check cache key in workflow matches your setup
- GitHub has 10GB cache limit per repo (old caches auto-deleted)
- Use cache-clean.yml to manage cache size

### Artifacts not appearing

- Check workflow succeeded (green checkmark)
- Artifacts expire after retention period (default 30 days)
- Download within retention window

## 📝 Maintenance

### Update dependencies

```bash
# Update Go dependencies
go get -u ./...
go mod tidy
go mod vendor

# Update Nix flake
nix flake update

# Update vendor hash if needed
nix build  # Will show new hash if changed
```

### Monitor cache usage

```bash
# View cache size
gh cache list

# Delete specific cache
gh cache delete <cache-id>
```

## 🎓 Next Steps

1. ✅ Push workflows to GitHub
2. ⬜ Set up Cachix (optional but recommended)
3. ⬜ Enable branch protection
4. ⬜ Create your first release tag
5. ⬜ Configure notification preferences

---

**Questions?** Check the [GitHub Actions documentation](https://docs.github.com/en/actions) or open an issue!
