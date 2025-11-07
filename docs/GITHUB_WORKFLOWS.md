# GitHub Workflows Documentation

This document describes all GitHub Actions workflows configured for the Podling project.

## Workflows Overview

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:** Push and PR to `main` and `development` branches

**Jobs:**

- **Lint**: Runs `golangci-lint` to check code quality
- **Test**: Runs full test suite with PostgreSQL service
    - Includes race detection
    - Generates coverage report
    - Enforces 60% minimum coverage for internal packages
    - Uploads coverage to Codecov

**Status:** ✅ Already configured

---

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers:**

- Tags matching `v*.*.*` (e.g., `v1.0.0`)
- Manual trigger via `workflow_dispatch`

**Jobs:**

- **Build**: Builds binaries for multiple platforms
    - Linux (amd64, arm64)
    - macOS (amd64, arm64)
    - Windows (amd64)
    - Strips debug symbols for smaller binaries
    - Embeds version info via ldflags

- **Release**: Creates GitHub release with:
    - Auto-generated changelog from commits
    - Installation instructions
    - Compressed archives (`.tar.gz` for Unix, `.zip` for Windows)

- **Checksums**: Generates SHA256 checksums for all artifacts

**Usage:**

```bash
# Create and push a tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Workflow automatically builds and creates release
```

**Artifacts:**

- `podling-v1.0.0-linux-amd64.tar.gz`
- `podling-v1.0.0-darwin-arm64.tar.gz`
- `podling-v1.0.0-windows-amd64.zip`
- `SHA256SUMS`

---

### 3. Security Scanning (`.github/workflows/security.yml`)

**Triggers:**

- Push and PR to `main` and `development`
- Weekly schedule (Mondays at 9:00 AM UTC)
- Manual trigger

**Jobs:**

- **govulncheck**: Go vulnerability scanner using official Go tooling
- **gosec**: Security scanner for Go code (SAST)
- **nancy**: Dependency vulnerability scanner
- **trivy**: Filesystem vulnerability scanner
- **secret-scan**: Gitleaks secret detection
- **codeql**: GitHub's advanced semantic code analysis
- **summary**: Aggregates all scan results

**Results:** Uploaded to GitHub Security tab (SARIF format)

---

### 4. Docker Build & Push (`.github/workflows/docker.yml`)

**Triggers:**

- Push to `main` and `development`
- Tags matching `v*.*.*`
- Pull requests (build only, no push)
- Manual trigger

**Jobs:**

- **build-master**: Builds multi-arch master image
    - Platforms: `linux/amd64`, `linux/arm64`
    - Uses BuildKit cache for faster builds

- **build-worker**: Builds multi-arch worker image
    - Same platforms as master

- **scan-images**: Runs Trivy vulnerability scan on images

**Image Tags:**

```
ghcr.io/<username>/podling/master:latest        # latest from main
ghcr.io/<username>/podling/master:development   # latest from development
ghcr.io/<username>/podling/master:v1.0.0        # version tag
ghcr.io/<username>/podling/master:sha-abc123    # commit SHA
```

**Usage:**

```bash
# Pull and run master
docker pull ghcr.io/<username>/podling/master:latest
docker run -p 8080:8080 -e STORE_TYPE=memory ghcr.io/<username>/podling/master:latest

# Pull and run worker (requires Docker socket)
docker pull ghcr.io/<username>/podling/worker:latest
docker run -p 8081:8081 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/<username>/podling/worker:latest \
  -node-id=worker-1 -port=8081
```

**Dockerfiles:**

- `deployments/docker/Dockerfile.master` - Master controller
- `deployments/docker/Dockerfile.worker` - Worker agent

---

### 5. PR Automation (`.github/workflows/pr-automation.yml`)

**Triggers:** Pull request events (opened, synchronized, reopened)

**Jobs:**

- **auto-label**: Automatically labels PRs based on:
    - Changed files (component, area, type)
    - PR size (xs, s, m, l, xl)
    - Configuration: `.github/labeler.yml`

- **coverage-comment**: Posts detailed coverage report as PR comment
    - Shows total coverage
    - Breaks down by package
    - Updates on each commit
    - Uses sticky comments (updates same comment)

- **pr-checklist**: Posts checklist for new PRs
    - Testing reminders
    - Code quality checks
    - Documentation requirements

- **pr-title-check**: Validates PR title follows semantic convention
    - Format: `type: description`
    - Types: `feat`, `fix`, `docs`, `chore`, `ci`, etc.
    - Subject must start with uppercase

- **conflict-check**: Detects merge conflicts and comments

**Example Labels Applied:**

- `component: master`, `component: worker`, `component: cli`
- `area: api`, `area: storage`, `area: scheduler`
- `type: tests`, `type: documentation`, `type: dependencies`
- `size/s`, `size/m`, `size/l`
- `breaking change`, `needs review`

---

### 6. Dependabot (`.github/dependabot.yml`)

**Configuration:** Automatic dependency updates

**Ecosystems:**

- **Go modules** (`go.mod`): Weekly updates on Mondays
- **GitHub Actions**: Weekly updates
- **Docker**: Weekly Dockerfile base image updates

**Behavior:**

- Groups minor and patch updates together
- Creates labeled PRs automatically
- Limits open PRs to prevent spam
- Follows semantic commit convention

---

## Secrets Required

### Optional Secrets

1. **CODECOV_TOKEN** (optional)
    - For uploading coverage to Codecov
    - Get from: https://codecov.io
    - Add to: Settings → Secrets → Actions

2. **Personal Access Token** (automatically provided)
    - `GITHUB_TOKEN` is automatically available
    - No setup required

---

## Setting Up Workflows

### 1. Enable GitHub Actions

Already enabled by default for this repository.

### 2. Configure Branch Protection

Recommended settings for `main` branch:

```
Settings → Branches → Add rule for 'main':
☑ Require a pull request before merging
☑ Require status checks to pass before merging
  - CI / Lint
  - CI / Test
  - Security / govulncheck
  - PR Automation / pr-title-check
☑ Require conversation resolution before merging
☑ Do not allow bypassing the above settings
```

### 3. Enable Dependency Scanning

```
Settings → Security → Code security and analysis:
☑ Dependency graph
☑ Dependabot alerts
☑ Dependabot security updates
```

### 4. First Release

```bash
# Ensure all tests pass
make test

# Create annotated tag
git tag -a v0.1.0 -m "First release"

# Push tag to trigger release workflow
git push origin v0.1.0

# Check Actions tab for progress
# Release will appear under Releases section
```

---

## Workflow Status Badges

Add these badges to your README:

```markdown
[![CI](https://github.com/<username>/podling/actions/workflows/ci.yml/badge.svg)](https://github.com/<username>/podling/actions/workflows/ci.yml)
[![Security](https://github.com/<username>/podling/actions/workflows/security.yml/badge.svg)](https://github.com/<username>/podling/actions/workflows/security.yml)
[![Docker](https://github.com/<username>/podling/actions/workflows/docker.yml/badge.svg)](https://github.com/<username>/podling/actions/workflows/docker.yml)
[![codecov](https://codecov.io/gh/<username>/podling/branch/main/graph/badge.svg)](https://codecov.io/gh/<username>/podling)
```

---

## Troubleshooting

### Release Workflow Fails

- Ensure tag matches `v*.*.*` format (e.g., `v1.0.0`)
- Check that `go.mod` specifies Go 1.25
- Verify all tests pass before tagging

### Docker Workflow Fails

- Check Dockerfiles exist in `deployments/docker/`
- Ensure Go version matches in Dockerfile and `go.mod`
- Verify GitHub Container Registry permissions

### Coverage Comment Not Posted

- Check PR has write permissions
- Verify `GITHUB_TOKEN` has PR write access
- Ensure tests generate `coverage.out`

### PR Labels Not Applied

- Check `.github/labeler.yml` syntax
- Verify labeler action version is current
- Ensure PR modifies files matching patterns

---

## Future Enhancements

Potential workflow additions:

1. **Benchmark Workflow**: Track performance regressions
2. **Integration Tests**: End-to-end testing with Docker Compose
3. **Nightly Builds**: Daily builds with extended tests
4. **Release Notes Generator**: Automated changelog generation
5. **Deployment Workflow**: Auto-deploy to staging/production

---

## Maintenance

### Keeping Workflows Updated

Dependabot will automatically create PRs for:

- GitHub Actions version updates
- Base image updates in Dockerfiles

Review and merge these PRs regularly.

### Monitoring Workflow Usage

Check Actions tab for:

- Failed workflow runs
- Workflow execution time
- GitHub Actions minutes usage (free tier: 2,000 min/month)

### Workflow Costs

All workflows use free GitHub-hosted runners:

- Ubuntu: 1x multiplier
- Typical run time: 5-10 minutes per workflow
- Expected monthly usage: ~500 minutes (well within free tier)

---

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Configuration](https://golangci-lint.run/usage/configuration/)
- [Docker BuildKit Cache](https://docs.docker.com/build/cache/)
- [Semantic PR Titles](https://www.conventionalcommits.org/)
