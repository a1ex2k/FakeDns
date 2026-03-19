#pragma once

#include <cstdint>
#include <string>

#include "ip_address.h"

namespace fakedns {

constexpr uint32_t kFakeIpMaxCount = 131070;

struct IPv4Pool {
  uint32_t network = 0;
  uint32_t max_count = 0;
};

struct IPv6Pool {
  IPv6Addr network;
  uint64_t max_count = 0;
};

class IpPoolParser final {
 public:
  static bool ParseIPv4Pool(const std::string& cidr, IPv4Pool* pool);
  static bool ParseIPv6Pool(const std::string& cidr, IPv6Pool* pool);
};

}  // namespace fakedns
