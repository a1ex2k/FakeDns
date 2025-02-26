using System.Collections.Concurrent;
using System.Net;
using System.Net.Sockets;
using System.Runtime.InteropServices;
using Ae.Dns.Protocol;
using Ae.Dns.Protocol.Enums;
using Ae.Dns.Protocol.Records;

namespace FakeDns;

/// <summary>
/// Fake ranges are 198.18.0.0/15 and A000::/3
/// </summary>
internal sealed class FakeIPAddressProvider
{
    private readonly Action<IPAddress, IPAddress> _onFakeIp;
    private const uint FakeIPv4MaxCount = 131_070;
    private const uint SubnetIPv4 = 0xC6120000;  // 198.18.0.0;

    private readonly ConcurrentDictionary<IPAddress, IPAddress> _fakingMapping;
    private uint _currentIPv4Offset = 0;

    public FakeIPAddressProvider(Action<IPAddress, IPAddress> onFakeIP)
    {
        _onFakeIp = onFakeIP;
        _fakingMapping = new ConcurrentDictionary<IPAddress, IPAddress>();
    }


    public IPAddress? GetOrCreateFakeAddress(IPAddress realAddress)
    {
        if (_fakingMapping.TryGetValue(realAddress, out var fakeAddress))
        {
            return fakeAddress;
        }

        var fakeIp = CreateFakeAddressIfPossible(realAddress);
        if (fakeIp != null)
        {
            _fakingMapping.TryAdd(realAddress, fakeIp);
        }

        return fakeIp;
    }

    private IPAddress? CreateFakeAddressIfPossible(IPAddress realAddress)
    {
        var newAddress = realAddress.AddressFamily switch
        {
            AddressFamily.InterNetwork => MoveIPv4ToFakeRange(realAddress),
            AddressFamily.InterNetworkV6 => MoveIPv6ToFakeRange(realAddress),
            _ => null
        };

        if (newAddress != null)
        {
            _onFakeIp(realAddress, newAddress);
        }

        return newAddress;
    }

    private IPAddress? MoveIPv4ToFakeRange(IPAddress ipAddress)
    {
        var addressOffset = Interlocked.Increment(ref _currentIPv4Offset);
        if (addressOffset >= FakeIPv4MaxCount)
        {
            return null;
        }

        var uintAddress = SubnetIPv4 + addressOffset;
        Span<byte> bytes = stackalloc byte[4];
        MemoryMarshal.Write(bytes, uintAddress);
        if (BitConverter.IsLittleEndian)
        {
            bytes.Reverse();
        }
        return new IPAddress(bytes);
    }

    private IPAddress? MoveIPv6ToFakeRange(IPAddress ipAddress)
    {
        Span<byte> addrBytes = stackalloc byte[16];
        if (!ipAddress.TryWriteBytes(addrBytes, out _))
            return null;

        if (addrBytes[0] < 0x20 || addrBytes[0] > 0x3f)
            return null;

        addrBytes[0] = (byte)(addrBytes[0] + 0x80);
        return new IPAddress(addrBytes, ipAddress.ScopeId);
    }
}