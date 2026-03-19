#pragma once

#include <array>
#include <cstdint>
#include <string>

namespace fakedns {

struct IPv6Addr {
  uint64_t hi = 0;
  uint64_t lo = 0;

  bool operator==(const IPv6Addr& other) const { return hi == other.hi && lo == other.lo; }
};

class IpAddress final {
 public:
  static IPv6Addr FromRawV6(const uint8_t* bytes);
  static void ToRawV6(const IPv6Addr& addr, uint8_t* bytes);
  static std::string ToStringV4(uint32_t ip);
  static std::string ToStringV6(const IPv6Addr& addr);
  static IPv6Addr ApplyPrefixMaskV6(const IPv6Addr& ip, uint8_t prefix);
  static IPv6Addr AddV6(const IPv6Addr& base, uint64_t increment);
};

}  // namespace fakedns
