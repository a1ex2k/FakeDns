package main

const (
	nftTableName        = "inet fake_ip"
	nftChainName        = "prerouting"
	nftMarkingChainName = "prerouting_mangle"
	nftHookPrio         = "-101"
	nftMarkingHookPrio  = "-151"

	maxUDPSize     = 65535 // Maximum UDP packet size for DNS
	fakedRecordTTL = 300   // TTL (in seconds) for the DNS records we modify

	defaultDomainsFile  = "/etc/fakedns/fake-domains.list"
	defaultListenIp     = "127.0.0.1"
	defaultListenPort   = 50053
	defaultCatchAllPort = 50054
	defaultUpstream     = "1.1.1.1:53"
	defaultFwMark       = 200
)
