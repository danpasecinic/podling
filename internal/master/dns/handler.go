package dns

import (
	"log"
	"net"
	"strings"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/miekg/dns"
)

const (
	DefaultClusterDomain = "cluster.local"
	DefaultTTL           = 30
)

type Handler struct {
	store         state.StateStore
	clusterDomain string
}

func NewHandler(store state.StateStore, clusterDomain string) *Handler {
	if clusterDomain == "" {
		clusterDomain = DefaultClusterDomain
	}
	return &Handler{
		store:         store,
		clusterDomain: clusterDomain,
	}
}

func (h *Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		switch q.Qtype {
		case dns.TypeA:
			h.handleARecord(m, q)
		default:
			log.Printf("DNS: unsupported query type %s for %s", dns.TypeToString[q.Qtype], q.Name)
		}
	}

	if len(m.Answer) == 0 && m.Rcode == dns.RcodeSuccess {
		m.SetRcode(r, dns.RcodeNameError)
	}

	if err := w.WriteMsg(m); err != nil {
		log.Printf("DNS: failed to write response: %v", err)
	}
}

func (h *Handler) handleARecord(m *dns.Msg, q dns.Question) {
	name := strings.TrimSuffix(q.Name, ".")

	serviceName, namespace, ok := h.parseServiceName(name)
	if !ok {
		log.Printf("DNS: cannot parse service name from %s", name)
		return
	}

	service, err := h.store.GetServiceByName(namespace, serviceName)
	if err != nil {
		log.Printf("DNS: service not found: %s.%s (%v)", serviceName, namespace, err)
		return
	}

	if service.ClusterIP == "" {
		log.Printf("DNS: service %s.%s has no ClusterIP", serviceName, namespace)
		return
	}

	ip := net.ParseIP(service.ClusterIP)
	if ip == nil {
		log.Printf("DNS: invalid ClusterIP %s for service %s.%s", service.ClusterIP, serviceName, namespace)
		return
	}

	rr := &dns.A{
		Hdr: dns.RR_Header{
			Name:   q.Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    DefaultTTL,
		},
		A: ip.To4(),
	}
	m.Answer = append(m.Answer, rr)
	log.Printf("DNS: resolved %s -> %s", name, service.ClusterIP)
}

func (h *Handler) parseServiceName(name string) (serviceName, namespace string, ok bool) {
	suffix := ".svc." + h.clusterDomain
	if !strings.HasSuffix(name, suffix) {
		return "", "", false
	}

	name = strings.TrimSuffix(name, suffix)
	parts := strings.Split(name, ".")

	if len(parts) == 1 {
		return parts[0], "default", true
	}
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}

	return "", "", false
}
