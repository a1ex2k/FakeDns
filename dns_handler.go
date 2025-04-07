package main

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns" // DNS library
)

type dnsHandler struct {
	upstreamAddr  string
	fakeIPManager *FakeIPManager
	forceFakeAll  bool
	logPrefix     string
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	resp, _, err := c.Exchange(r, h.upstreamAddr)
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

	out := resp.Copy()
	out.Answer = []dns.RR{}

	for _, rr := range resp.Answer {
		newRR := dns.Copy(rr)

		var domain string
		var realIP net.IP
		var isA, isAAAA bool
		var aRec *dns.A
		var aaaaRec *dns.AAAA
		var shouldFake bool
		var fakeIP net.IP

		if aRec, isA = newRR.(*dns.A); isA {
			domain = strings.ToLower(strings.TrimSuffix(aRec.Hdr.Name, "."))
			realIP = aRec.A
		} else if aaaaRec, isAAAA = newRR.(*dns.AAAA); isAAAA {
			domain = strings.ToLower(strings.TrimSuffix(aaaaRec.Hdr.Name, "."))
			realIP = aaaaRec.AAAA
		}

		if isA || isAAAA {
			fakeIP, shouldFake = h.fakeIPManager.GetFakeIP(domain, realIP, h.forceFakeAll)
			if shouldFake {
				log.Printf("%s Faking IP for %s: %s -> %s", h.logPrefix, domain, realIP, fakeIP)
				if isA {
					aRec.A = fakeIP
				} else {
					aaaaRec.AAAA = fakeIP
				}
				newRR.Header().Ttl = fakedRecordTTL
			}
		}

		out.Answer = append(out.Answer, newRR)
	}

	if err := w.WriteMsg(out); err != nil {
		log.Printf("%s Error writing response: %v", h.logPrefix, err)
	}
}
