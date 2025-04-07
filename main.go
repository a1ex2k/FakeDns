package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/miekg/dns"
)

func main() {
	log.Printf("Starting FakeDNS Go application on %s...", runtime.GOOS)

	if len(os.Args) < 4 {
		log.Fatalf("Usage: %s <domains_file> <listen_ip> <port> <catch_all_port>", os.Args[0])
	}
	domainsFilePath := os.Args[1]
	listenIPStr := os.Args[2]
	portStr := os.Args[3]
	catchAllPortStr := os.Args[4]

	listenIP := net.ParseIP(listenIPStr)
	if listenIP == nil {
		log.Fatalf("Invalid listen IP address provided: %s", listenIPStr)
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		log.Fatalf("Invalid listen port provided: %s", portStr)
		return
	}

	catchAllPort, err := strconv.ParseUint(catchAllPortStr, 10, 16)
	if err != nil {
		log.Fatalf("Invalid listen port provided: %s", catchAllPortStr)
		return
	}

	targetDomains, err := loadDomains(domainsFilePath)
	if err != nil {
		log.Fatalf("Failed to load domains: %v", err)
	}

	if err := setupNftables(); err != nil {
		log.Printf("ERROR: Initial nftables setup failed: %v.", err)
	}

	fakeIPManager := NewFakeIPManager(targetDomains, addNftRule)

	handler := &dnsHandler{
		upstreamAddr:  upstreamDNSServer,
		fakeIPManager: fakeIPManager,
	}
	udpServer := &dns.Server{
		Addr:    fmt.Sprintf("%s:%d", listenIP.String(), port),
		Net:     "udp",
		Handler: handler,
		UDPSize: maxUDPSize,
	}

	catchAllHandler := &dnsHandler{
		upstreamAddr:  upstreamDNSServer,
		fakeIPManager: fakeIPManager,
		forceFakeAll:  true,
	}
	catchAllUdpServer := &dns.Server{
		Addr:    fmt.Sprintf("%s:%d", listenIP.String(), catchAllPort),
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

	// Wait for either a shutdown signal or a failure
	select {
	case <-sigs:
	case err := <-serverFailed:
		log.Printf("Server failed to start: %v. Initiating shutdown...", err)
		select {
		case sigs <- syscall.SIGTERM:
		default:
		}
	}

	cleanupNftables()
	log.Println("FakeDNS Go application finished.")
}
