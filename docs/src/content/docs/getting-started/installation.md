---
title: Installation
description: How to install Litmus on your system.
---

There are several ways to install Litmus on your system.

## Pre-built Binaries

Download a pre-built binary from the [latest release](https://github.com/lukecarr/litmus/releases/latest).

## Install with Go

If you have Go installed, you can install Litmus using:

```bash
go install go.carr.sh/litmus@latest
```

## Compile from Source

Clone the repository and build from source:

```bash
git clone https://github.com/lukecarr/litmus.git
cd litmus
go build -o litmus .
```

## Verify Installation

After installation, verify that Litmus is working:

```bash
litmus --help
```

You should see the help output with available commands and flags.

## Next Steps

- [Quick Start](/getting-started/quick-start/) - Run your first test
