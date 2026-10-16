// Package lookup finds hop names and network numbers over DNS, caching each answer.
package lookup

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

// retryAfter is how long an address with no answers waits before it is looked up again.
const retryAfter = time.Minute

// queryTimeout limits each DNS query.
const queryTimeout = 5 * time.Second

// Info is what is known about one address.
type Info struct {
	Name   string
	ASN    int
	ASName string
}

type entry struct {
	info    Info
	tried   time.Time
	running bool
}

// Resolver looks addresses up in the background and caches the results.
type Resolver struct {
	mu    sync.Mutex
	cache map[netip.Addr]*entry
	asns  map[int]string
	dns   *net.Resolver
}

// New returns a Resolver using the system's DNS.
func New() *Resolver {
	return &Resolver{cache: map[netip.Addr]*entry{}, asns: map[int]string{}, dns: net.DefaultResolver}
}

// Get returns what is known about a so far, starting a lookup when a is new
// or its last lookup, more than retryAfter ago, left something unanswered.
func (r *Resolver) Get(a netip.Addr) Info {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.cache[a]
	if !ok {
		e = &entry{}
		r.cache[a] = e
	}
	if !e.running && !complete(a, e.info) && time.Since(e.tried) > retryAfter {
		e.running, e.tried = true, time.Now()
		go r.fill(a)
	}
	return e.info
}

func (r *Resolver) fill(a netip.Addr) {
	name := r.name(a)
	asn := 0
	if !isPrivate(a) {
		asn = r.origin(a)
	}
	asName := r.asName(asn)
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.cache[a]
	e.running = false
	if e.info.Name == "" {
		e.info.Name = name
	}
	if e.info.ASN == 0 {
		e.info.ASN, e.info.ASName = asn, asName
	}
}

func complete(a netip.Addr, info Info) bool {
	return info.Name != "" && (info.ASN != 0 || isPrivate(a))
}

func isPrivate(a netip.Addr) bool { return a.IsPrivate() || a.IsLinkLocalUnicast() }

func (r *Resolver) name(a netip.Addr) string {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	names, err := r.dns.LookupAddr(ctx, a.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

// origin asks Team Cymru which network announces a.
func (r *Resolver) origin(a netip.Addr) int {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	txt, err := r.dns.LookupTXT(ctx, originQuery(a))
	if err != nil || len(txt) == 0 {
		return 0
	}
	return parseOrigin(txt[0])
}

func (r *Resolver) asName(asn int) string {
	if asn == 0 {
		return ""
	}
	r.mu.Lock()
	name, ok := r.asns[asn]
	r.mu.Unlock()
	if ok {
		return name
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	txt, err := r.dns.LookupTXT(ctx, fmt.Sprintf("AS%d.asn.cymru.com", asn))
	if err == nil && len(txt) > 0 {
		name = parseASName(txt[0])
	}
	r.mu.Lock()
	r.asns[asn] = name
	r.mu.Unlock()
	return name
}

// originQuery returns the Team Cymru origin lookup name for an IPv4 address.
func originQuery(a netip.Addr) string {
	b := a.As4()
	return fmt.Sprintf("%d.%d.%d.%d.origin.asn.cymru.com", b[3], b[2], b[1], b[0])
}

// parseOrigin reads the first network number from a record like "64496 | 192.0.2.0/24 | US | arin | 2012-05-01".
func parseOrigin(txt string) int {
	first := strings.Fields(strings.Split(txt, "|")[0])
	if len(first) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(first[0])
	return n
}

// parseASName reads the name from a record like "64496 | US | arin | 2001-06-29 | EXAMPLE-ISP, US".
func parseASName(txt string) string {
	parts := strings.Split(txt, "|")
	name := strings.TrimSpace(parts[len(parts)-1])
	if i := strings.LastIndex(name, ", "); i > 0 && len(name)-i == 4 {
		name = name[:i]
	}
	return name
}
