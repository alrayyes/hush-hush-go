# hush-hush-go

[![ci](https://github.com/alrayyes/hush-hush-go/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/hush-hush-go/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/alrayyes/hush-hush-go/graph/badge.svg)](https://codecov.io/gh/alrayyes/hush-hush-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/alrayyes/hush-hush-go.svg)](https://pkg.go.dev/github.com/alrayyes/hush-hush-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/alrayyes/hush-hush-go)](https://goreportcard.com/report/github.com/alrayyes/hush-hush-go)
[![release](https://img.shields.io/github/v/release/alrayyes/hush-hush-go)](https://github.com/alrayyes/hush-hush-go/releases)
[![license](https://img.shields.io/github/license/alrayyes/hush-hush-go)](LICENSE)

The official Go SDK for [hush-hush](https://github.com/alrayyes/hush-hush),
generated from its OpenAPI spec and kept in sync with it automatically.

## Install

```sh
go get github.com/alrayyes/hush-hush-go
```

Requires Go 1.26 or newer.

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	hushhush "github.com/alrayyes/hush-hush-go"
)

func main() {
	client, err := hushhush.NewClient("https://hush-hush.example.com",
		hushhush.WithAPIKey("your-api-key"), // or set HUSH_HUSH_API_KEY
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Create is a write operation — it needs the credential above.
	if _, err := client.CreateObject(ctx, hushhush.CreateObjectRequest{
		Id:    "my-first-secret",
		Value: []byte("already-sealed-ciphertext"),
	}, "my-program"); err != nil {
		log.Fatal(err)
	}

	// Get needs no credential — hush-hush's confidentiality boundary is
	// "who holds a matching private key," not who's calling this endpoint.
	value, err := client.GetObject(ctx, "my-first-secret", "my-program")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("got %d bytes of sealed ciphertext\n", len(value))

	// The audit log records every read and write; querying it needs no
	// credential either, and returns the full matching result set (there's
	// no pagination on this endpoint).
	entries, err := client.QueryAuditLog(ctx, hushhush.AuditLogFilter{})
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		fmt.Println(entry.Action, entry.ObjectId, entry.Timestamp)
	}
}
```

The API key is only required for write operations (create/update/delete);
reads (get, used-by, audit-log query) work without one. `caller`, the last
argument to most methods, is optional — pass `""` to leave it unset. See
[GoDoc](https://pkg.go.dev/github.com/alrayyes/hush-hush-go) for the full API
surface.

## Versioning

This SDK's version tracks hush-hush's OpenAPI spec, not this repo's own
commit history — see [CONTRIBUTING.md](CONTRIBUTING.md) for how a spec
change becomes a release.

## License

[MIT](LICENSE)
