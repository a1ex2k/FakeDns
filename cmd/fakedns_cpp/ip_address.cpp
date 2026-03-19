#include "ip_address.h"

#include <arpa/inet.h>

#include <array>
#include <cstring>

namespace fakedns {
namespace {

inline uint64_t HostToBigEndian64(uint64_t value) {
#if defined(__BYTE_ORDER__) && (__BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__)
  return __builtin_bswap64(value);
#else
  return value;
#endif
}

inline uint64_t BigEndianToHost64(uint64_t value) {
  return HostToBigEndian64(value);
}

}  // namespace

IPv6Addr IpAddress::FromRawV6(const uint8_t* bytes) {
  IPv6Addr addr {};
  std::memcpy(&addr.hi, bytes, sizeof(addr.hi));
  std::memcpy(&addr.lo, bytes + sizeof(addr.hi), sizeof(addr.lo));
  addr.hi = BigEndianToHost64(addr.hi);
  addr.lo = BigEndianToHost64(addr.lo);
  return addr;
}

void IpAddress::ToRawV6(const IPv6Addr& addr, uint8_t* bytes) {
  const uint64_t hi = HostToBigEndian64(addr.hi);
  const uint64_t lo = HostToBigEndian64(addr.lo);
  std::memcpy(bytes, &hi, sizeof(hi));
  std::memcpy(bytes + sizeof(hi), &lo, sizeof(lo));
}

std::string IpAddress::ToStringV4(uint32_t ip) {
  struct in_addr addr {};
  addr.s_addr = htonl(ip);
  char buffer[INET_ADDRSTRLEN] = {};
  if (inet_ntop(AF_INET, &addr, buffer, sizeof(buffer)) == nullptr) {
    return "0.0.0.0";
  }
  return std::string(buffer);
}

std::string IpAddress::ToStringV6(const IPv6Addr& addr) {
  std::array<uint8_t, 16> raw {};
  ToRawV6(addr, raw.data());
  char buffer[INET6_ADDRSTRLEN] = {};
  if (inet_ntop(AF_INET6, raw.data(), buffer, sizeof(buffer)) == nullptr) {
    return "::";
  }
  return std::string(buffer);
}

IPv6Addr IpAddress::ApplyPrefixMaskV6(const IPv6Addr& ip, uint8_t prefix) {
  if (prefix == 0) {
    return {};
  }
  if (prefix >= 128) {
    return ip;
  }
  if (prefix >= 64) {
    const uint8_t lo_bits = static_cast<uint8_t>(prefix - 64);
    const uint64_t lo_mask = (lo_bits == 0) ? 0 : (~0ULL << (64 - lo_bits));
    return {ip.hi, ip.lo & lo_mask};
  }
  const uint64_t hi_mask = ~0ULL << (64 - prefix);
  return {ip.hi & hi_mask, 0};
}

IPv6Addr IpAddress::AddV6(const IPv6Addr& base, uint64_t increment) {
  IPv6Addr out = base;
  const uint64_t before = out.lo;
  out.lo += increment;
  if (out.lo < before) {
    ++out.hi;
  }
  return out;
}

}  // namespace fakedns
