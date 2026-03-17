package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
)

func main() {
	domainsFile := flag.String("domains", defaultDomainsFile, "Path to the domains file")
	listenIPStr := flag.String("listen", defaultListenIp, "IP address to listen on")
	port := flag.Uint("port", 0, "Port for faking by list")
	fwmarkMask := UintFlag("fwmark", 0, "Fwmark mask to set on packets/connnection")
	fake4CIDR := flag.String("fake4", defaultFake4CIDR, "IPv4 fake IP CIDR")
	fake6CIDR := flag.String("fake6", defaultFake6CIDR, "IPv6 fake IP CIDR (recommended /64)")
	catchAllPort := flag.Uint("catch-all-port", 0, "Port for faking all DNS requests")
	upstreamResolver := flag.String("upstream", defaultUpstream, "Upstream DNS resolver (host:port)")
	autoReload := flag.Bool("autoreload", true, "Auto-reload (whether to track changes of domains file)")

	flag.Parse()
	if *port == 0 && *catchAllPort == 0 {
		log.Fatalf("FATAL: Atleast one port required.")
		return
	}

	enableByList := *port > 0
	enableCatchAll := *catchAllPort > 0

	log.Printf("Fake IPv4 CIDR:      %s", *fake4CIDR)
	log.Printf("Fake IPv6 CIDR:      %s", *fake6CIDR)

	if enableByList {
		log.Printf("Listen Fake-by-List: %s:%d", *listenIPStr, *port)
		log.Printf("Domains file:        %s", *domainsFile)
		log.Printf("Auto-reload:         %v", *autoReload)
	}
	if enableCatchAll {
		log.Printf("Listen Fake-All:     %s:%d", *listenIPStr, *catchAllPort)
	}
	log.Printf("Upstream DNS:        %s", *upstreamResolver)
	if *fwmarkMask > 0 {
		log.Printf("Firewall mark mask:  %x", *fwmarkMask)
	}

	if err := SetupNftables(*fake4CIDR, *fake6CIDR, *fwmarkMask); err != nil {
		log.Fatalf("FATAL: Initial nftables setup failed: %v.", err)
		return
	}

	fakeIPManager, err := NewFakeIPManager(*fake4CIDR, *fake6CIDR, AddDnat4Rule, AddDnat6Rule)
	if err != nil {
		log.Fatalf("FATAL: Failed to init fake IP manager: %v", err)
		return
	}

	log.Printf("Starting FakeDNS server on %s...", *listenIPStr)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	serverFailed := make(chan error, 2)

	dnsClient := &dns.Client{Net: "udp"}
	var reloadDomains func() error

	if enableByList {
		domainsList, err := DomainsListFromFile(*domainsFile)
		if err != nil {
			log.Fatalf("FATAL: Failed to load initial domains file: %v", err)
			return
		}

		reloadDomains = domainsList.Load
		handler := &DnsHandler{
			upstreamAddr:  *upstreamResolver,
			fakeIPManager: fakeIPManager,
			domainsList:   domainsList,
			forceFakeAll:  false,
			dnsClient:     dnsClient,
		}
		fakeByListUdpServer := &dns.Server{
			Addr:    *listenIPStr + ":" + strconv.FormatUint(uint64(*port), 10),
			Net:     "udp",
			Handler: handler,
			UDPSize: maxUDPSize,
		}

		go func() {
			if err := fakeByListUdpServer.ListenAndServe(); err != nil {
				log.Printf("Failed to start server: %v", err)
				serverFailed <- err
			}
		}()
	}

	if enableCatchAll {
		catchAllHandler := &DnsHandler{
			upstreamAddr:  *upstreamResolver,
			fakeIPManager: fakeIPManager,
			domainsList:   nil,
			forceFakeAll:  true,
			dnsClient:     dnsClient,
		}
		fakeAlllUdpServer := &dns.Server{
			Addr:    *listenIPStr + ":" + strconv.FormatUint(uint64(*catchAllPort), 10),
			Net:     "udp",
			Handler: catchAllHandler,
			UDPSize: maxUDPSize,
		}

		go func() {
			if err := fakeAlllUdpServer.ListenAndServe(); err != nil {
				log.Printf("Failed to start fake-all server: %v", err)
				serverFailed <- err
			}
		}()
	}

	var watcher *fsnotify.Watcher
	var watcherEvents chan fsnotify.Event
	var watcherErrors chan error
	var debounceTimer *time.Timer

	if enableByList && *autoReload {
		var err error
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("FATAL: Failed to create file watcher: %v", err)
		}
		defer watcher.Close()
		if err := watcher.Add(*domainsFile); err != nil {
			log.Printf("WARNING: Failed to add %s to watcher: %v", *domainsFile, err)
		} else {
			log.Printf("Tracking changes in %s", *domainsFile)
		}
		watcherEvents = watcher.Events
		watcherErrors = watcher.Errors
	}

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGHUP:
				if enableByList {
					log.Println("SIGHUP received, reloading domains list...")
					if err := reloadDomains(); err != nil {
						log.Printf("ERROR: Failed to reload domains file: %v", err)
					} else {
						log.Println("Domains list reloaded successfully.")
					}
				} else {
					log.Println("SIGHUP received, but fake-by-list is disabled, ignoring.")
				}
			case syscall.SIGINT, syscall.SIGTERM:
				log.Println("Shutdown signal received, cleaning up...")
				CleanupNftables()
				log.Println("FakeDNS Go application finished.")
				return
			}
		case event, ok := <-watcherEvents:
			if !ok {
				watcherEvents = nil
				continue
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Create) {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(time.Second, func() {
					log.Printf("File modified (%s), reloading domains list...", event.Op)
					if err := reloadDomains(); err != nil {
						log.Printf("ERROR: Failed to auto-reload domains file: %v", err)
					} else {
						log.Println("Domains list auto-reloaded successfully.")
					}

					if watcher != nil {
						_ = watcher.Remove(*domainsFile)
						if err := watcher.Add(*domainsFile); err != nil {
							log.Printf("ERROR: Could not re-watch file after edit: %v", err)
						}
					}
				})
			}
		case err, ok := <-watcherErrors:
			if !ok {
				watcherErrors = nil
				continue
			}
			log.Printf("Watcher error: %v", err)
		case err := <-serverFailed:
			log.Printf("Server failed: %v. Initiating shutdown...", err)
			CleanupNftables()
			log.Println("FakeDNS Go application finished.")
			return
		}
	}
}

func UintFlag(name string, value uint, usage string) *uint {
	p := new(uint)
	*p = value
	flag.Func(name, usage, func(s string) error {
		// strconv.IntSize ensures it respects the architecture's uint size (32 or 64 bit)
		val, err := strconv.ParseUint(s, 0, strconv.IntSize)
		if err != nil {
			return err
		}
		*p = uint(val)
		return nil
	})
	return p
}
