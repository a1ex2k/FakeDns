package main

const (
	nftFamily           = "inet"
	nftTableName        = "fake_ip"
	nftChainName        = "prerouting"
	nftMarkingChainName = "prerouting_mangle"
	nftHookPrio         = "-101"
	nftMarkingHookPrio  = "-160"

	maxUDPSize     = 65535 // Maximum UDP packet size for DNS
	fakedRecordTTL = 300   // TTL (in seconds) for the DNS records we modify

	defaultDomainsFile = "/etc/fakedns/fake-domains.list"
	defaultListenIp    = "127.0.0.1"
	defaultUpstream    = "8.8.8.8:53"

	defaultFake4CIDR = "198.18.0.0/15"
	defaultFake6CIDR = "abcd:bad:c0de::/64"
)
