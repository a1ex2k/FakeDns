package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/miekg/dns"
)

func main() {
	domainsFile := flag.String("domains", defaultDomainsFile, "Path to the domains file")
	listenIPStr := flag.String("listen", defaultListenIp, "IP address to listen on")
	port := flag.Uint("port", defaultListenPort, "Port for FakeDNS to listen on")
	fwmark := flag.Uint("fwmark", defaultFwMark, "Fwmark to set on packets")
	fake4CIDR := flag.String("fake4", defaultFake4CIDR, "IPv4 fake IP CIDR")
	fake6CIDR := flag.String("fake6", defaultFake6CIDR, "IPv6 fake IP CIDR (recommended /64)")

	catchAllPort := flag.Uint("catch-all-port", 0, "Port for faking all DNS requests")
	upstreamResolver := flag.String("upstream", defaultUpstream, "Upstream DNS resolver (host:port)")

	flag.Parse()
	skipCatchAll := *catchAllPort == 0
	log.Printf("Domains file:   %s", *domainsFile)
	log.Printf("Firewall mark:  %x", *fwmark)
	log.Printf("Fake IPv4 CIDR: %s", *fake4CIDR)
	log.Printf("Fake IPv6 CIDR: %s", *fake6CIDR)

	if skipCatchAll {
		log.Printf("Listen address: %s:%d (no catch-all server)", *listenIPStr, *port)
	} else {
		log.Printf("Listen address: %s:%d (catch-all on %s:%d)", *listenIPStr, *port, *listenIPStr, *catchAllPort)
	}
	log.Printf("Upstream DNS:   %s", *upstreamResolver)

	if err := SetupNftables(*fake4CIDR, *fake6CIDR, *fwmark); err != nil {
		log.Fatalf("FATAL: Initial nftables setup failed: %v.", err)
		return
	}

	fakeIPManager, err := NewFakeIPManager(*fake4CIDR, *fake6CIDR, AddDnat4Rule, AddDnat6Rule)
	if err != nil {
		log.Fatalf("FATAL: Failed to init fake IP manager: %v", err)
		return
	}
	dnsClient := &dns.Client{Net: "udp"}

	domainsList, err := DomainsListFromFile(*domainsFile)
	if err != nil {
		log.Fatalf("FATAL: Failed to load initial domains file: %v", err)
		return
	}

	handler := &DnsHandler{
		upstreamAddr:  *upstreamResolver,
		fakeIPManager: fakeIPManager,
		domainsList:   domainsList,
		forceFakeAll:  false,
		dnsClient:     dnsClient,
	}
	udpServer := &dns.Server{
		Addr:    *listenIPStr + ":" + strconv.FormatUint(uint64(*port), 10),
		Net:     "udp",
		Handler: handler,
		UDPSize: maxUDPSize,
	}

	var catchAllUdpServer *dns.Server
	if !skipCatchAll {
		catchAllHandler := &DnsHandler{
			upstreamAddr:  *upstreamResolver,
			fakeIPManager: fakeIPManager,
			domainsList:   nil,
			forceFakeAll:  true,
			dnsClient:     dnsClient,
		}
		catchAllUdpServer = &dns.Server{
			Addr:    *listenIPStr + ":" + strconv.FormatUint(uint64(*catchAllPort), 10),
			Net:     "udp",
			Handler: catchAllHandler,
			UDPSize: maxUDPSize,
		}
	}

	log.Printf("Starting FakeDNS server on %s...", *listenIPStr)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	serverFailed := make(chan error, 2)

	go func() {
		if err := udpServer.ListenAndServe(); err != nil {
			log.Printf("Failed to start server: %v", err)
			serverFailed <- err
		}
	}()

	if !skipCatchAll {
		go func() {
			if err := catchAllUdpServer.ListenAndServe(); err != nil {
				log.Printf("Failed to start fake-all server: %v", err)
				serverFailed <- err
			}
		}()
	}

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGHUP:
				log.Println("SIGHUP received, reloading domains list...")
				if err := domainsList.Load(); err != nil {
					log.Printf("ERROR: Failed to reload domains file: %v", err)
				} else {
					log.Println("Domains list reloaded successfully.")
				}
			case syscall.SIGINT, syscall.SIGTERM:
				log.Println("Shutdown signal received, cleaning up...")
				CleanupNftables()
				log.Println("FakeDNS Go application finished.")
				return
			}
		case err := <-serverFailed:
			log.Printf("Server failed: %v. Initiating shutdown...", err)
			CleanupNftables()
			log.Println("FakeDNS Go application finished.")
			return
		}
	}
}
