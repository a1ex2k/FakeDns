# FakeDns
DNS server that implements Fake-IP 

This .NET 8 app resolves domains from list to 'fake' ranges `198.18.0.0/15` and `a000::/3` and adds DNAT rules using nftables.  
Upstream DNS resolver is hardcoded as `1.1.1.1` for now. 

*Launch with argumennts:*
```
./FakeDns /path/to/fake-domains.list <Listen-IP> <Port-of-server> <Port-of-server-that-fakes-all-IPs>
./FakeDns /etc/fake-domains.list 10.0.0.1 53 54
```
Second port used when u have local server like dnsmasq and want let it decide what domains to fake itself.
