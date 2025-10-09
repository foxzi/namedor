# Changelog Generation Guide

This project uses [git-cliff](https://github.com/orhun/git-cliff) to automatically generate changelogs from conventional commits.

## Commit Message Format

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `build`: Build system changes
- `ci`: CI/CD configuration changes
- `chore`: Other changes that don't modify src or test files

### Examples

```bash
# Feature
git commit -m "feat(dns): add support for DNSSEC validation"

# Bug fix
git commit -m "fix(replication): prevent duplicate records on sync"

# Breaking change
git commit -m "feat(api)!: remove deprecated v1 endpoints

BREAKING CHANGE: The v1 API endpoints have been removed. Please migrate to v2."

# With scope and detailed description
git commit -m "fix(geoip): correct continent detection for country codes

- Fixed mapping for Asian countries
- Added fallback logic for unknown countries
- Updated tests"
```

## Generating Changelog Locally

Install git-cliff:

```bash
# On macOS
brew install git-cliff

# On Linux
wget https://github.com/orhun/git-cliff/releases/download/v2.7.0/git-cliff-2.7.0-x86_64-unknown-linux-gnu.tar.gz
tar -xzf git-cliff-2.7.0-x86_64-unknown-linux-gnu.tar.gz
sudo mv git-cliff-2.7.0/git-cliff /usr/local/bin/
```

Generate changelog:

```bash
# Full changelog
git-cliff --output CHANGELOG.md

# Changelog for specific tag range
git-cliff v0.1.0..v0.2.0

# Changelog for unreleased changes
git-cliff --unreleased
```

## Automated Release Process

When you push a tag starting with `v`, GitHub Actions will:

1. Generate a changelog for this release (comparing with previous tag)
2. Create a full CHANGELOG.md with complete history
3. Attach both files to the GitHub Release
4. Use the release-specific changelog as the release notes

### Creating a Release

```bash
# Tag the release
git tag -a v0.2.0 -m "Release v0.2.0"

# Push the tag to trigger CI
git push origin v0.2.0
```

The CI will automatically:
- Build packages
- Generate changelog
- Create GitHub Release with release notes
- Attach DEB/RPM packages
- Update APT/YUM repositories
