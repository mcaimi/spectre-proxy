package vhost

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/mcaimi/spectre-proxy/internal/storage"
)

type WildcardHost struct {
	Pattern *regexp.Regexp
	VHost   *storage.VirtualHost
}

type Router struct {
	mu       sync.RWMutex
	exact    map[string]*storage.VirtualHost
	wildcard []*WildcardHost
}

func NewRouter() *Router {
	return &Router{
		exact:    make(map[string]*storage.VirtualHost),
		wildcard: make([]*WildcardHost, 0),
	}
}

func (r *Router) Add(vhost *storage.VirtualHost) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hostname := normalizeHostname(vhost.Hostname)

	if strings.Contains(hostname, "*") {
		pattern, err := wildcardToRegex(hostname)
		if err != nil {
			return fmt.Errorf("invalid wildcard pattern %s: %w", hostname, err)
		}

		r.wildcard = append(r.wildcard, &WildcardHost{
			Pattern: pattern,
			VHost:   vhost,
		})
	} else {
		r.exact[hostname] = vhost
	}

	return nil
}

func (r *Router) Remove(hostname string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	hostname = normalizeHostname(hostname)

	delete(r.exact, hostname)

	filtered := make([]*WildcardHost, 0)
	for _, wh := range r.wildcard {
		if wh.VHost.Hostname != hostname {
			filtered = append(filtered, wh)
		}
	}
	r.wildcard = filtered
}

func (r *Router) Match(hostname string) (*storage.VirtualHost, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hostname = normalizeHostname(hostname)

	if vhost, ok := r.exact[hostname]; ok && vhost.Enabled {
		return vhost, true
	}

	for _, wh := range r.wildcard {
		if wh.Pattern.MatchString(hostname) && wh.VHost.Enabled {
			return wh.VHost, true
		}
	}

	return nil, false
}

func (r *Router) LoadFromRepository(repo *storage.VHostRepository) error {
	vhosts, err := repo.List()
	if err != nil {
		return fmt.Errorf("failed to load virtual hosts: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.exact = make(map[string]*storage.VirtualHost)
	r.wildcard = make([]*WildcardHost, 0)

	for _, vhost := range vhosts {
		hostname := normalizeHostname(vhost.Hostname)

		if strings.Contains(hostname, "*") {
			pattern, err := wildcardToRegex(hostname)
			if err != nil {
				continue
			}
			r.wildcard = append(r.wildcard, &WildcardHost{
				Pattern: pattern,
				VHost:   vhost,
			})
		} else {
			r.exact[hostname] = vhost
		}
	}

	return nil
}

func normalizeHostname(hostname string) string {
	hostname = strings.ToLower(hostname)

	if idx := strings.Index(hostname, ":"); idx != -1 {
		hostname = hostname[:idx]
	}

	return hostname
}

func wildcardToRegex(pattern string) (*regexp.Regexp, error) {
	pattern = regexp.QuoteMeta(pattern)

	pattern = strings.ReplaceAll(pattern, `\*`, `[^.]+`)

	pattern = "^" + pattern + "$"

	return regexp.Compile(pattern)
}
