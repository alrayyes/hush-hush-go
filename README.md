# hush-hush-go

[![ci](https://github.com/alrayyes/hush-hush-go/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/hush-hush-go/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/alrayyes/hush-hush-go/graph/badge.svg)](https://codecov.io/gh/alrayyes/hush-hush-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/alrayyes/hush-hush-go.svg)](https://pkg.go.dev/github.com/alrayyes/hush-hush-go)
[![release](https://img.shields.io/github/v/release/alrayyes/hush-hush-go)](https://github.com/alrayyes/hush-hush-go/releases)
[![license](https://img.shields.io/github/license/alrayyes/hush-hush-go)](LICENSE)

The official Go SDK for [hush-hush](https://github.com/alrayyes/hush-hush),
generated from its OpenAPI spec and kept in sync with it automatically.

## Install

```sh
go get github.com/alrayyes/hush-hush-go/v4
```

Requires Go 1.26 or newer.

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
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
	// credential either. QueryAuditLog returns one page at a time (default
	// 50 entries, capped at 500) — set AuditLogFilter.Limit for the page
	// size and After to the previous page's last entry's Id to fetch more.
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
argument to most methods, is optional — pass `""` to leave it unset. The
consumer directory (`ListConsumers`, `AddConsumer`, `UpdateConsumer`,
`DeleteConsumer`, and `GetConsumerPublicKey` for resolving one consumer's
registered age public key by exact name) always requires a credential, the
same as `ListObjects` — listing needs no ID the caller already holds.
Both list calls read the server a page of 500 at a time and return the whole
list, so nothing is cut off however many objects there are. `ListObjects`
narrows by consumer; `ListObjectsFiltered` takes a `ListObjectsFilter` to
narrow by tag too, where an object must carry every tag listed.
`GetOwnerIdentity` returns the owner's escrowed public key, which a client adds
as a recipient before sealing to honor `keep_readable_copy`; it works with an
API key and needs hush-hush v2.54.0 or later.
Consumer read tokens (`CreateConsumerToken`, `ListConsumerTokens`,
`RotateConsumerToken`, `RevokeConsumerToken`, `PurgeConsumerToken`) always
require a credential too, the same as the consumer directory — this
client's own bearer credential may mint, list, rotate, revoke, or purge
one, not only an administrator session. See
[GoDoc](https://pkg.go.dev/github.com/alrayyes/hush-hush-go) for the full API
surface.

## Reports

The latest green run on `main` publishes its test and coverage reports at
<https://apis.ryankes.eu/hush-hush-go/reports/>. It has the JUnit test results
and the coverage as HTML, Cobertura XML and Go's native profile.

## Versioning

This SDK's version tracks hush-hush's OpenAPI spec, not this repo's own
commit history — see [CONTRIBUTING.md](CONTRIBUTING.md) for how a spec
change becomes a release.

## License

[MIT](LICENSE)
