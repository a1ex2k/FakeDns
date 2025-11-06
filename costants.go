package fakedns

const (
	nftTableName = "inet fake_ip" // Table name used in nftables
	nftChainName = "prerouting"   // Chain name used in nftables
	nftHookPrio  = "-100"         // Priority for the nftables hoo

	upstreamDNSServer = "1.1.1.1:53" // Default upstream DNS server (Cloudflare)
	maxUDPSize        = 65535        // Maximum UDP packet size for DNS
	fakedRecordTTL    = 300          // TTL (in seconds) for the DNS records we modify
)
