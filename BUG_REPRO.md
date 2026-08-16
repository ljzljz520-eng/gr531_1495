# BUG_REPRO

The following failures were observed while validating the initial project state.
Each section records what failed, how to reproduce it, and the complete command output.
They are preserved intentionally; only failing build gates are omitted from the generated Dockerfile.

## Failure 1: Go test (.)

- Observed problem: `Go test (.)` failed in the initial project state.
- Working directory: `.`
- Command: `cd /app && GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 ./...`
- Exit status: `1`

```text
?   	example.com/order-schema-console/cmd/conversion-console	[no test files]
?   	example.com/order-schema-console/internal/conversion	[no test files]
?   	example.com/order-schema-console/internal/domain	[no test files]
?   	example.com/order-schema-console/internal/fixture	[no test files]
--- FAIL: TestPreviewIdentifiesUnsupportedOrderField (0.00s)
    handler_test.go:85: conversion error = {Code:conversion_failed Message:schema conversion failed}, want the affected order field
FAIL
FAIL	example.com/order-schema-console/internal/httpapi	0.002s
?   	example.com/order-schema-console/internal/service	[no test files]
FAIL
```

## Architecture reproduction

### linux/amd64
- Go toolchain version: exit `0`
- Go build (.): exit `0`
- Go test (.): exit `1`
- Go run smoke (cmd/conversion-console): exit `0`
### linux/arm64
- Go toolchain version: exit `0`
- Go build (.): exit `0`
- Go test (.): exit `1`
- Go run smoke (cmd/conversion-console): exit `0`
