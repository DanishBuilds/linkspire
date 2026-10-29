<!-- test/security/README.md -->
# Security tests

Tagged `security` so they are excluded from the default `go build`/`go test`:

```bash
go test -tags security ./test/security/...
```

| File | Covers | Needs a running stack? |
|------|--------|------------------------|
| `ssrf_test.go` | SSRF IP policy (metadata, loopback, RFC1918, IPv4-mapped, unspecified) and scheme deny-list | no |
| `decoder_fuzz_test.go` | Base62 decoder (never panics) and Feistel code round-trip bijection | no (native fuzzing) |
| `authz_integration_test.go` | tenant isolation (workspace A vs B), auth required, malicious-destination rejection | yes: `LINKSPIRE_API_URL` |
| `ratelimit_integration_test.go` | rate limiting + X-Forwarded-For spoof resistance | yes: `REDIRECT_URL` |

Deeper unit coverage for the resolver, DNS-rebinding, dword/octal encodings, slug/profanity and
URL policy lives beside the code in `pkg/*/*_test.go` and runs in the normal suite.

Fuzz a target for longer:

```bash
go test -tags security -run '^$' -fuzz FuzzGeneratorRoundTrip -fuzztime 60s ./test/security/
```
