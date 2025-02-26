using System;
using System.Collections;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using Ae.Dns.Protocol;

namespace FakeDns;

internal static class DnsLabelExtensions
{
    public static bool IsSubdomainOf(this DnsLabels domain, DnsLabels parent)
    {
        var dif = domain.Count - parent.Count;
        if (dif <= 0) return false;

        for (int i = parent.Count - 1; i >= 0; i--)
        {
            if (!parent[i].Equals(domain[i + dif], StringComparison.OrdinalIgnoreCase))
                return false;
        }
        return true;
    }

    public static int GetHashCodeByLabel(this DnsLabels domain)
    {
        int value = 14657;

        foreach (var t in domain)
        {
            value = HashCode.Combine(t, value);
        }

        return value;
    }


}

internal sealed class HostEqualityComparer : IEqualityComparer<DnsLabels>
{
    public bool Equals(DnsLabels x, DnsLabels y)
    {
        return x == y;
    }

    public int GetHashCode(DnsLabels obj)
    {
        return obj.GetHashCodeByLabel();
    }
}