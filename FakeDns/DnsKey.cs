using Ae.Dns.Protocol;
using System;
using System.Collections.Generic;
using System.Diagnostics.CodeAnalysis;
using System.Linq;
using System.Net;
using System.Text;
using System.Threading.Tasks;

namespace FakeDns;

internal readonly struct DnsKey : IEquatable<DnsKey>
{ 
    public readonly DnsLabels Host;
    public readonly IPAddress RealAddress;

    public override bool Equals([NotNullWhen(true)] object? obj)
    {
        return obj is DnsKey other && Equals(other);
    }

    public DnsKey(DnsLabels host, IPAddress realAddress)
    {
        Host = host;
        RealAddress = realAddress;
    }

    public bool Equals(DnsKey other)
    {
        return Host.Equals(other.Host) && RealAddress.Equals(other.RealAddress);
    }

    public override int GetHashCode()
    {
        return HashCode.Combine(Host.GetHashCodeByLabel(), RealAddress);
    }

    public override string ToString()
    {
        return $"{Host} at {RealAddress}";
    }
}