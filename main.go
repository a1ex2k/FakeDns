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
	port := flag.Int("port", defaultListenPort, "Port for FakeDNS to listen on")
	catchAllPort := flag.Int("catch-all-port", defaultCatchAllPort, "Port for faking all DNS requests")
	upstreamResolver := flag.String("upstream", defaultUpstream, "Upstream DNS resolver (host:port)")

	flag.Parse()

	log.Printf("Domains file:   %s", *domainsFile)
	log.Printf("Listen address: %s:%d (catch-all on :%d)", *listenIPStr, *port, *catchAllPort)
	log.Printf("Upstream DNS:   %s", *upstreamResolver)

	if err := SetupNftables(); err != nil {
		log.Printf("ERROR: Initial nftables setup failed: %v.", err)
	}

	fakeIPManager := NewFakeIPManager(AddDnat4Rule, AddDnat6Rule)
	dnsClient := &dns.Client{Net: "udp"}

	domainsList, err := DomainsListFromFile(*domainsFile)
	if err != nil {
		log.Fatalf("FATAL: Failed to load initial domains file: %v", err)
	}

	handler := &DnsHandler{
		upstreamAddr:  *upstreamResolver,
		fakeIPManager: fakeIPManager,
		domainsList:   domainsList,
		forceFakeAll:  false,
		dnsClient:     dnsClient,
	}
	udpServer := &dns.Server{
		Addr:    *listenIPStr + ":" + strconv.Itoa(*port),
		Net:     "udp",
		Handler: handler,
		UDPSize: maxUDPSize,
	}

	catchAllHandler := &DnsHandler{
		upstreamAddr:  *upstreamResolver,
		fakeIPManager: fakeIPManager,
		domainsList:   nil,
		forceFakeAll:  true,
		dnsClient:     dnsClient,
	}
	catchAllUdpServer := &dns.Server{
		Addr:    *listenIPStr + ":" + strconv.Itoa(*catchAllPort),
		Net:     "udp",
		Handler: catchAllHandler,
		UDPSize: maxUDPSize,
	}

	log.Printf("Starting FakeDNS server on %s, ports %d and %d", *listenIPStr, *port, *catchAllPort)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	serverFailed := make(chan error, 2)

	go func() {
		if err := udpServer.ListenAndServe(); err != nil {
			log.Printf("Failed to start server: %v", err)
			serverFailed <- err
		}
	}()
	go func() {
		if err := catchAllUdpServer.ListenAndServe(); err != nil {
			log.Printf("Failed to start fake-all server: %v", err)
			serverFailed <- err
		}
	}()

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
