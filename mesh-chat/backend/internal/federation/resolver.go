package federation

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dnsPrefix      = "_meshchat._tcp."
	cacheTTL       = 15 * time.Minute
	defaultTimeout = 5 * time.Second
)

type cacheEntry struct {
	server    ResolvedServer
	expiresAt time.Time
}

type Resolver struct {
	db    *pgxpool.Pool
	mu    sync.RWMutex
	cache map[string]cacheEntry
}

func NewResolver(db *pgxpool.Pool) *Resolver {
	return &Resolver{
		db:    db,
		cache: make(map[string]cacheEntry),
	}
}

func (r *Resolver) ResolveServer(ctx context.Context, domain string) (ResolvedServer, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return ResolvedServer{}, fmt.Errorf("domain is required")
	}

	if server, ok := r.getCached(domain); ok {
		return server, nil
	}

	publicKey, err := r.GetServerKey(ctx, domain)
	if err != nil {
		return ResolvedServer{}, err
	}

	endpoint, err := r.lookupEndpoint(ctx, domain)
	if err != nil {
		endpoint = buildDefaultEndpoint(domain)
	}

	resolved := ResolvedServer{
		Domain:    domain,
		PublicKey: publicKey,
		Endpoint:  endpoint,
	}
	r.setCached(domain, resolved)

	return resolved, nil
}

func (r *Resolver) GetServerKey(ctx context.Context, domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return "", fmt.Errorf("domain is required")
	}

	if server, ok := r.getCached(domain); ok && server.PublicKey != "" {
		return server.PublicKey, nil
	}

	var publicKey string
	err := r.db.QueryRow(ctx, `
		SELECT public_key
		FROM federated_servers
		WHERE domain = $1 AND status <> 'blocked'
	`, domain).Scan(&publicKey)
	if err == nil && publicKey != "" {
		cached := ResolvedServer{
			Domain:    domain,
			PublicKey: publicKey,
			Endpoint:  buildDefaultEndpoint(domain),
		}
		r.setCached(domain, cached)
		return publicKey, nil
	}

	publicKey, endpoint, lookupErr := r.lookupTXTRecord(ctx, domain)
	if lookupErr != nil {
		if err != nil {
			return "", fmt.Errorf("resolve server key for %s: %w", domain, lookupErr)
		}
		return "", fmt.Errorf("resolve server key for %s: %w", domain, err)
	}

	if upsertErr := r.upsertServer(ctx, domain, publicKey); upsertErr != nil {
		return "", upsertErr
	}

	r.setCached(domain, ResolvedServer{
		Domain:    domain,
		PublicKey: publicKey,
		Endpoint:  endpoint,
	})

	return publicKey, nil
}

func (r *Resolver) UpsertServerKey(ctx context.Context, domain, publicKey string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" || strings.TrimSpace(publicKey) == "" {
		return fmt.Errorf("domain and public key are required")
	}

	if err := r.upsertServer(ctx, domain, publicKey); err != nil {
		return err
	}

	cached, _ := r.getCached(domain)
	cached.Domain = domain
	cached.PublicKey = publicKey
	if cached.Endpoint == "" {
		cached.Endpoint = buildDefaultEndpoint(domain)
	}
	r.setCached(domain, cached)

	return nil
}

func (r *Resolver) lookupEndpoint(ctx context.Context, domain string) (string, error) {
	_, endpoint, err := r.lookupTXTRecord(ctx, domain)
	return endpoint, err
}

func (r *Resolver) lookupTXTRecord(ctx context.Context, domain string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	records, err := net.DefaultResolver.LookupTXT(ctx, dnsPrefix+domain)
	if err != nil {
		return "", "", fmt.Errorf("dns txt lookup: %w", err)
	}

	for _, record := range records {
		publicKey, endpoint := parseTXTRecord(record)
		if publicKey == "" {
			continue
		}
		if endpoint == "" {
			endpoint = buildDefaultEndpoint(domain)
		}
		return publicKey, endpoint, nil
	}

	return "", "", fmt.Errorf("no meshchat txt record found")
}

func (r *Resolver) upsertServer(ctx context.Context, domain, publicKey string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO federated_servers (domain, public_key, status, last_ping)
		VALUES ($1, $2, 'active', NOW())
		ON CONFLICT (domain)
		DO UPDATE SET public_key = EXCLUDED.public_key, status = 'active', last_ping = NOW()
	`, domain, publicKey)
	if err != nil {
		return fmt.Errorf("upsert federated server: %w", err)
	}

	return nil
}

func (r *Resolver) getCached(domain string) (ResolvedServer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.cache[domain]
	if !ok || time.Now().After(entry.expiresAt) {
		return ResolvedServer{}, false
	}

	return entry.server, true
}

func (r *Resolver) setCached(domain string, server ResolvedServer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[domain] = cacheEntry{
		server:    server,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

func parseTXTRecord(record string) (string, string) {
	parts := strings.Split(record, ";")
	values := make(map[string]string, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}

		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		values[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}

	if values["v"] != "mc1" || values["key"] == "" {
		return "", ""
	}

	endpoint := values["endpoint"]
	if endpoint != "" {
		if _, err := url.ParseRequestURI(endpoint); err != nil {
			endpoint = ""
		}
	}

	return values["key"], endpoint
}

func buildDefaultEndpoint(domain string) string {
	return "https://" + domain
}
