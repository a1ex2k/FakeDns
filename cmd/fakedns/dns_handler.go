package main

import (
	"log"
	"net"
	"strings"

	"github.com/miekg/dns" // DNS library
)

type DnsHandler struct {
	upstreamAddr  string
	domainsList   *DomainsList
	fakeIPManager *FakeIPManager
	forceFakeAll  bool
	logPrefix     string
	maxLevel      int
	dnsClient     *dns.Client
}

func (h *DnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		log.Printf("%s Received DNS query without question section", h.logPrefix)
		dns.HandleFailed(w, r)
		return
	}

	resp, _, err := h.dnsClient.Exchange(r, h.upstreamAddr)
	if err != nil {
		log.Printf("%s Error forwarding query to %s: %v", h.logPrefix, h.upstreamAddr, err)
		dns.HandleFailed(w, r)
		return
	}
	if resp == nil {
		log.Printf("%s Received nil response from upstream %s", h.logPrefix, h.upstreamAddr)
		dns.HandleFailed(w, r)
		return
	}

	domainName := strings.ToLower(r.Question[0].Name)
	shouldFake := h.forceFakeAll || h.domainsList.MatchDomain(domainName, h.maxLevel)
	if shouldFake {
		for i := range resp.Answer {
			var realIP net.IP
			var version int

			switch rec := resp.Answer[i].(type) {
			case *dns.A:
				realIP = rec.A
				version = 4
			case *dns.AAAA:
				realIP = rec.AAAA
				version = 6
			default:
				continue
			}

			fakeIP := h.fakeIPManager.GetFakeIP(realIP, version)
			if fakeIP == nil {
				log.Printf("%s Failed to allocate fake IP for %s (%s), keeping upstream answer", h.logPrefix, domainName, realIP)
				continue
			}

			log.Printf("%s Faking IP for %s: %s -> %s", h.logPrefix, domainName, realIP, fakeIP)
			switch rec := resp.Answer[i].(type) {
			case *dns.A:
				rec.A = fakeIP
				rec.Hdr.Ttl = fakedRecordTTL
			case *dns.AAAA:
				rec.AAAA = fakeIP
				rec.Hdr.Ttl = fakedRecordTTL
			}
		}
	}

	if err := w.WriteMsg(resp); err != nil {
		log.Printf("%s Error writing response: %v", h.logPrefix, err)
	}
}
