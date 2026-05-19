# CHANGELOG.md
# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning.

## [Unreleased]

### Added
- GitHub Actions CI workflow to validate formatting, run tests, and build the project.
- New unit tests for configuration parsing and RAG signal/prompt behaviour.
- Docker helper script to install/start local Qdrant and Redis containers for development.
- Documentation updates for CI, local infra bootstrap, and release/versioning process.

## [0.1.0] - 2026-05-05

### Added
- Initial ArchiMind release with Go HTTP server, RAG pipeline, Redis memory/cache, Qdrant integration, and static web UI.
