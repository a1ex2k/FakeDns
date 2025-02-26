using Ae.Dns.Protocol;
using Ae.Dns.Protocol.Enums;
using Ae.Dns.Protocol.Records;

namespace FakeDns;

internal sealed class ListFakingDnsClient : IDnsClient
{
    private readonly IDnsClient _upstreamClient;
    private readonly FakeIPAddressProvider _provider;
    private readonly ReaderWriterLockSlim _fakedDomainsLockObject = new();
    private readonly HashSet<DnsLabels> _fakedDomains;
    private readonly DnsLabels[] _initialDomains;

    public ListFakingDnsClient(FakeIPAddressProvider provider, IDnsClient upstreamClient, string[] initialDomains)
    {
        _upstreamClient = upstreamClient;
        _provider = provider;
        _fakedDomains = new HashSet<DnsLabels>(100, new HostEqualityComparer());
        _initialDomains = Array.ConvertAll(initialDomains, d => new DnsLabels(d));
        foreach (var domain in _initialDomains)
        {
            _fakedDomains.Add(domain);
        }
    }

    public void Dispose()
    {
        _fakedDomainsLockObject.Dispose();
    }

    public async Task<DnsMessage> Query(DnsMessage query, CancellationToken token = default)
    {
        var result = await _upstreamClient.Query(query, token);
        foreach (var answer in result.Answers)
        {
            if (answer.Type == DnsQueryType.CNAME)
            {
                var domainResource = (DnsDomainResource)answer.Resource!;
                _fakedDomains.Add(new DnsLabels(domainResource.Domain));
                continue;
            }

            if (answer.Type != DnsQueryType.A && answer.Type != DnsQueryType.AAAA)
            {
                continue;
            }

            if (!_fakedDomainsLockObject.TryEnterReadLock(5_000)) continue;
            var isInSetAlready = _fakedDomains.Contains(answer.Host);
            _fakedDomainsLockObject.ExitReadLock();

            if (isInSetAlready)
            {
                var ipResource = ((DnsIpAddressResource)answer.Resource!);
                var fakeIP = _provider.GetOrCreateFakeAddress(ipResource.IPAddress);
                if (fakeIP != null)
                {
                    ipResource.IPAddress = fakeIP;
                }
                continue;
            }

            foreach (var domain in _initialDomains)
            {
                if (!answer.Host.IsSubdomainOf(domain)) continue;

                var ipResource = ((DnsIpAddressResource)answer.Resource!);
                var fakeIP = _provider.GetOrCreateFakeAddress(ipResource.IPAddress);
                if (fakeIP == null)
                {
                    continue;
                }

                ipResource.IPAddress = fakeIP;

                if (!_fakedDomainsLockObject.TryEnterWriteLock(5_000)) continue;
                _fakedDomains.Add(answer.Host);
                _fakedDomainsLockObject.ExitWriteLock();
            }
        }

        return result;
    }

    public async Task PreResolveAsync()
    {
        foreach (var domain in _initialDomains)
        {
            var result = await _upstreamClient.Query(DnsQueryFactory.CreateQuery(domain));
            foreach (var answer in result.Answers)
            {
                if (answer.Type == DnsQueryType.CNAME)
                {
                    var domainResource = (DnsDomainResource)answer.Resource!;
                    _fakedDomains.Add(new DnsLabels(domainResource.Domain));
                    continue;
                }

                if (answer.Type != DnsQueryType.A && answer.Type != DnsQueryType.AAAA)
                {
                    continue;
                }

                var ipResource = (DnsIpAddressResource)answer.Resource!;
                _ = _provider.GetOrCreateFakeAddress(ipResource.IPAddress);
            }
        }
    }
}