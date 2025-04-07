package main

const (
	nftTableName = "inet fake_ip" // Table name used in nftables
	nftChainName = "prerouting"   // Chain name used in nftables
	nftHookPrio  = "-100"         // Priority for the nftables hook

	fakeIPv4Subnet   = uint32(0xC6120000) // 198.18.0.0 - Start of the fake IP range (RFC 5737 TEST-NET-2)
	fakeIPv4MaxCount = uint32(131070)     // Maximum number of fake IPs to generate (~198.19.255.254)

	upstreamDNSServer = "1.1.1.1:53" // Default upstream DNS server (Cloudflare)
	maxUDPSize        = 65535        // Maximum UDP packet size for DNS
	fakedRecordTTL    = 300          // TTL (in seconds) for the DNS records we modify
)
