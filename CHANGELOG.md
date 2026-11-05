# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0](https://github.com/manovaspace/orbit-notifications/compare/v0.3.2...v0.4.0) (2026-11-05)


### Features

* **dev:** use the orbit-block dev tooling ports ([65e26a4](https://github.com/manovaspace/orbit-notifications/commit/65e26a467b4d407850e9e2483e94665920fe5d52))
* **dev:** use the orbit-block dev tooling ports ([54c2996](https://github.com/manovaspace/orbit-notifications/commit/54c2996891be27ff9f9c2469b7f3f5f081776a73))

## [0.3.2](https://github.com/manovaspace/orbit-notifications/compare/v0.3.1...v0.3.2) (2026-11-03)


### Bug Fixes

* **smtp:** authenticate only when the server advertises AUTH ([#51](https://github.com/manovaspace/orbit-notifications/issues/51)) ([72ae571](https://github.com/manovaspace/orbit-notifications/commit/72ae5713da44f381feb1c9f593509c74636a1b8d))
* **smtp:** upgrade to STARTTLS before AUTH ([#49](https://github.com/manovaspace/orbit-notifications/issues/49)) ([ebb039d](https://github.com/manovaspace/orbit-notifications/commit/ebb039ddc5cef542d87e028411939d2c1a041dbf))

## [0.3.1](https://github.com/manovaspace/orbit-notifications/compare/v0.3.0...v0.3.1) (2026-11-03)


### Bug Fixes

* **build:** report the real service version ([#47](https://github.com/manovaspace/orbit-notifications/issues/47)) ([8e1df04](https://github.com/manovaspace/orbit-notifications/commit/8e1df040fcd2e780596b291d3b7b6db29cc6b83f))

## [0.3.0](https://github.com/manovaspace/orbit-notifications/compare/v0.2.0...v0.3.0) (2026-11-03)


### Features

* **flags:** read feature flags from the platform endpoint ([#45](https://github.com/manovaspace/orbit-notifications/issues/45)) ([e331da8](https://github.com/manovaspace/orbit-notifications/commit/e331da822a71c98d23bcedef9bb5cdcad5e584fb))

## [0.2.0](https://github.com/manovaspace/orbit-notifications/compare/v0.1.1...v0.2.0) (2026-11-03)


### Features

* **templates:** add org_invite and signup_verify email templates ([#34](https://github.com/manovaspace/orbit-notifications/issues/34)) ([f81f499](https://github.com/manovaspace/orbit-notifications/commit/f81f499e97aa6bc4bd6b82c9d23a5896b63b55c8))

## [0.1.1](https://github.com/manovaspace/orbit-notifications/compare/v0.1.0...v0.1.1) (2026-11-02)


### Bug Fixes

* **featureflags:** register the Unleash provider without blocking ([#27](https://github.com/manovaspace/orbit-notifications/issues/27)) ([e0c96e6](https://github.com/manovaspace/orbit-notifications/commit/e0c96e69e194fd59c080b23abbb053ec30d447e1))
* **featureflags:** stop waiting forever for Unleash at startup ([#25](https://github.com/manovaspace/orbit-notifications/issues/25)) ([e6caff4](https://github.com/manovaspace/orbit-notifications/commit/e6caff42b5170a216504607210dd13c10b7d5c07))

## 0.1.0 (2026-11-02)


### Features

* add paper-letter mail catalog and multipart SMTP ([85632ca](https://github.com/manovaspace/orbit-notifications/commit/85632ca0ee0913fc0eba77b8558edd38fe652b1f))


### Bug Fixes

* **ci:** allow the invite token fixture in render_test ([7d38b82](https://github.com/manovaspace/orbit-notifications/commit/7d38b8296b64501e25eea5f4031304661345a7f9))
* **lint:** resolve staticcheck issues in service, mime, and render tests ([fc99529](https://github.com/manovaspace/orbit-notifications/commit/fc99529146720bb1e49c6bbd752e39779a2a4fd6))
* pin orbit observability dependency ([#17](https://github.com/manovaspace/orbit-notifications/issues/17)) ([9ae50b3](https://github.com/manovaspace/orbit-notifications/commit/9ae50b313aead222be01c371b5e3dae243da1c6d))
* require DATABASE_URL instead of a stale dev default ([#19](https://github.com/manovaspace/orbit-notifications/issues/19)) ([3ccd242](https://github.com/manovaspace/orbit-notifications/commit/3ccd242c100c0c8d8f0159960dd8395ffdb75363))


### Code Refactoring

* adopt the Go service standard ([#21](https://github.com/manovaspace/orbit-notifications/issues/21)) ([a46f429](https://github.com/manovaspace/orbit-notifications/commit/a46f429bbee3fb53f2bb169d6593c5000826dc4e))

## [Unreleased]

### Added

- Initial public MIT release.

### Changed

- Pin orbit-observability to an immutable Go pseudo-version so downstream consumers do not depend on local replace directives.
- Build container images from the committed module graph instead of resolving orbit-observability from `main` during the image build.
