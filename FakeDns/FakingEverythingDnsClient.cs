using Ae.Dns.Protocol;
using Ae.Dns.Protocol.Enums;
using Ae.Dns.Protocol.Records;

namespace FakeDns;

internal sealed class FakingEverythingDnsClient : IDnsClient
{
    private readonly IDnsClient _upstreamClient;
    private readonly FakeIPAddressProvider _provider;
    
    public FakingEverythingDnsClient(FakeIPAddressProvider provider, IDnsClient upstreamClient)
    {
        _upstreamClient = upstreamClient;
        _provider = provider;
    }

    public void Dispose() { }

    public async Task<DnsMessage> Query(DnsMessage query, CancellationToken token = default)
    {
        var result = await _upstreamClient.Query(query, token);
        foreach (var answer in result.Answers)
        {
            if (answer.Type != DnsQueryType.A
                && answer.Type != DnsQueryType.AAAA) continue;

            var ipResource = ((DnsIpAddressResource)answer.Resource!);
            var fakeIP = _provider.GetOrCreateFakeAddress(ipResource.IPAddress);
            if (fakeIP == null) continue;
            ipResource.IPAddress = fakeIP;
        }
        
        return result;
    }
}