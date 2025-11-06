package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/miekg/dns"
)

func main() {
	domainsFile := flag.String("domains", "", "Path to the domains file")
	listenIPStr := flag.String("listen", "127.0.0.1", "IP address to listen on [127.0.0.1]")
	port := flag.String("port", "50053", "Port for FakeDNS to listen on [50053]")
	catchAllPort := flag.String("catch-all-port", "50054", "Port for aking all DNS requests [50054]")
	upstreamResolver := flag.String("upstream", "8.8.8.8:53", "Upstream DNS resolver (host:port) [8.8.8.8:53]")

	flag.Parse()
	if *domainsFile == "" {
		log.Fatal("Error: --domains parameter is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	listenIP := net.ParseIP(*listenIPStr)
	if listenIP == nil {
		log.Fatalf("Invalid listen IP address provided: %s", *listenIPStr)
	}

	// --- Load domain list ---
	targetDomains, err := DomainsListFromFile(*domainsFile)
	if err != nil {
		log.Fatalf("Failed to load domains: %v", err)
	}

	log.Printf("Listen address:    %s:%d", listenIP, *port)
	log.Printf("Catch-all port:    %d", *catchAllPort)
	log.Printf("Upstream resolver: %s", *upstreamResolver)

	if err := SetupNftables(); err != nil {
		log.Printf("ERROR: Initial nftables setup failed: %v.", err)
	}

	fakeIPManager := NewFakeIPManager(AddDnat4Rule, AddDnat6Rule)
	dnsClient := &dns.Client{Net: "udp"}

	handler := &DnsHandler{
		upstreamAddr:  *upstreamResolver,
		fakeIPManager: fakeIPManager,
		domainsList:   targetDomains,
		forceFakeAll:  false,
		dnsClient:     dnsClient,
	}
	udpServer := &dns.Server{
		Addr:    listenIP.String() + ":" + *port,
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
		Addr:    listenIP.String() + ":" + *catchAllPort,
		Net:     "udp",
		Handler: catchAllHandler,
		UDPSize: maxUDPSize,
	}

	log.Printf("Starting FakeDNS server on %s, ports %d and %d", listenIP, port, catchAllPort)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
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

	select {
	case <-sigs:
	case err := <-serverFailed:
		log.Printf("Server failed to start: %v. Initiating shutdown...", err)
		select {
		case sigs <- syscall.SIGTERM:
		default:
		}
	}

	CleanupNftables()
	log.Println("FakeDNS Go application finished.")
}
