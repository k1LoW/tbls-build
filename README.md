<p align="center">
<br>
<img src="https://github.com/k1LoW/tbls-build/raw/main/img/logo.png" width="200" alt="tbls-build">
<br>
<br>
</p>

[![Build Status](https://github.com/k1LoW/tbls-build/workflows/build/badge.svg)](https://github.com/k1LoW/tbls-build/actions) [![GitHub release](https://img.shields.io/github/release/k1LoW/tbls-build.svg)](https://github.com/k1LoW/tbls-build/releases)

`tbls-build` is an external subcommand of [tbls](https://github.com/k1LoW/tbls) for customizing config file of [tbls](https://github.com/k1LoW/tbls) using other tbls.yml or schema.json.

## Usage

tbls-build is provided as an external subcommand of [tbls](https://github.com/k1LoW/tbls).

```
$ tbls build -c tbls.yml \
--underlay default.yml \
--overlay override.yml \
--underlay original.json \
--overlay add.json \
--out customized-tbls.yml
```

### Architecture

`tbls build` is a merge tool with a layered structure.

![layer](img/layer.png)

## Install

**deb:**

``` console
$ export TBLS_BUILD_VERSION=X.X.X
$ curl -o tbls-build.deb -L https://github.com/k1LoW/tbls-build/releases/download/v$TBLS_BUILD_VERSION/tbls-build_$TBLS_BUILD_VERSION-1_amd64.deb
$ dpkg -i tbls-build.deb
```

**RPM:**

``` console
$ export TBLS_BUILD_VERSION=X.X.X
$ yum install https://github.com/k1LoW/tbls-build/releases/download/v$TBLS_BUILD_VERSION/tbls-build_$TBLS_BUILD_VERSION-1_amd64.rpm
```

**homebrew tap:**

```console
$ brew install k1LoW/tap/tbls-build
```

**manually:**

Download binary from [releases page](https://github.com/k1LoW/tbls-build/releases)

**go get:**

```console
$ go get github.com/k1LoW/tbls-build
```

## Requirements

- [tbls](https://github.com/k1LoW/tbls) > 1.81.0
