// services/redirect/internal/handler/host.go
package handler

import (
	"context"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

// DomainRepo resolves a hostname to an active domain id.
type DomainRepo interface {
	DomainByHost(ctx context.Context, host string) (int64, bool, error)
}

// HostResolver maps a request Host header to a domain id, with a small TTL cache in front of
// the repo. A configured defaultDomainID is used when the host is unmapped (single-domain /
// dev deployments), so the service is usable without a populated domains table.
type HostResolver struct {
	repo            DomainRepo
	cache           *lru.LRU[string, int64]
	defaultDomainID int64
}

// NewHostResolver builds the resolver. repo may be nil (then only the default is used).
func NewHostResolver(repo DomainRepo, defaultDomainID int64, ttl time.Duration) *HostResolver {
	return &HostResolver{
		repo:            repo,
		cache:           lru.NewLRU[string, int64](10_000, nil, ttl),
		defaultDomainID: defaultDomainID,
	}
}

// NormalizeHost lowercases and strips any port from a Host header value.
func NormalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}

// Resolve returns the domain id for host.
func (h *HostResolver) Resolve(ctx context.Context, host string) (int64, bool) {
	host = NormalizeHost(host)
	if id, ok := h.cache.Get(host); ok {
		return id, true
	}
	if h.repo != nil {
		if id, ok, err := h.repo.DomainByHost(ctx, host); err == nil && ok {
			h.cache.Add(host, id)
			return id, true
		}
	}
	if h.defaultDomainID != 0 {
		return h.defaultDomainID, true
	}
	return 0, false
}
