using Ae.Dns.Protocol;
using Ae.Dns.Server;
using Ae.Dns.Client;
using System.Net;
using System.Runtime.Caching;
using FakeDns;
using System.Diagnostics;
using System.Text;

AppDomain.CurrentDomain.ProcessExit += (_, _) => RunBashCommand("nft delete table inet fake_ip");
RunBashCommand("nft add table inet fake_ip; nft add chain inet fake_ip prerouting { type nat hook prerouting priority -100\\; }");

var domains = File.ReadAllLines(args[0]);

using var upstreamClient = new DnsUdpClient(IPAddress.Parse("1.1.1.1"));
using var cache = new MemoryCache("dns");
using var upstreamCachingClient = new DnsCachingClient(upstreamClient, cache);
var ipFaker = new FakeIPAddressProvider(AddToNft);

var listFakingClient = new ListFakingDnsClient(ipFaker, upstreamCachingClient, domains);
await listFakingClient.PreResolveAsync();
var allFakingClient = new FakingEverythingDnsClient(ipFaker, upstreamCachingClient);

using var listRawClient = new DnsRawClient(listFakingClient);
using var everythingRawClient = new DnsRawClient(allFakingClient);
using var server1 = new DnsUdpServer(listRawClient, new DnsUdpServerOptions
{
    Endpoint = new IPEndPoint(IPAddress.Parse(args[1]), short.Parse(args[2]))
});

using var server2 = new DnsUdpServer(everythingRawClient, new DnsUdpServerOptions
{
    Endpoint = new IPEndPoint(IPAddress.Parse(args[1]), short.Parse(args[3]))
});

await Task.WhenAll(
    server1.Listen(default),
    server2.Listen(default)
    );


static void RunBashCommand(string command)
{
    Process.Start("/bin/bash", $"-c \"{command}\"");
}
static void AddToNft(IPAddress real, IPAddress fake)
{
    var command = $"nft add rule inet fake_ip prerouting ip{(real.AddressFamily == System.Net.Sockets.AddressFamily.InterNetwork ? ' ' : '6')} daddr {fake} dnat to {real}";
    RunBashCommand(command);
}