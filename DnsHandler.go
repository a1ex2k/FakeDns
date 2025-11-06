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
	out := resp.Copy()
	out.Answer = []dns.RR{}

	for _, rr := range resp.Answer {
		newRR := dns.Copy(rr)
		var realIP net.IP
		var isA, isAAAA bool
		var aRec *dns.A
		var aaaaRec *dns.AAAA
		version := 4
		if aRec, isA = newRR.(*dns.A); isA {
			realIP = aRec.A
		} else if aaaaRec, isAAAA = newRR.(*dns.AAAA); isAAAA {
			realIP = aaaaRec.AAAA
			version = 6
		}

		if shouldFake && (isA || isAAAA) {
			fakeIP := h.fakeIPManager.GetFakeIP(domainName, realIP, version)
			log.Printf("%s Faking IP for %s: %s -> %s", h.logPrefix, domainName, realIP, fakeIP)
			if isA {
				aRec.A = fakeIP
			} else {
				aaaaRec.AAAA = fakeIP
			}
			newRR.Header().Ttl = fakedRecordTTL
		}
		out.Answer = append(out.Answer, newRR)
	}

	if err := w.WriteMsg(out); err != nil {
		log.Printf("%s Error writing response: %v", h.logPrefix, err)
	}
}
