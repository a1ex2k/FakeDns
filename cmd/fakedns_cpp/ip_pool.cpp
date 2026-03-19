#include "ip_pool.h"

#include <arpa/inet.h>

#include <algorithm>
#include <array>
#include <cerrno>
#include <cstdlib>
#include <limits>

#include "logger.h"

namespace fakedns {
namespace {

bool ParseUnsigned(const std::string& input, uint64_t min, uint64_t max, uint64_t* out) {
  if (input.empty()) {
    return false;
  }
  errno = 0;
  char* end = nullptr;
  const unsigned long long value = std::strtoull(input.c_str(), &end, 0);
  if (errno != 0 || end == input.c_str() || *end != '\0') {
    return false;
  }
  if (value < min || value > max) {
    return false;
  }
  *out = static_cast<uint64_t>(value);
  return true;
}

bool SplitCidr(const std::string& cidr, std::string* ip, uint8_t* prefix) {
  const auto slash = cidr.find('/');
  if (slash == std::string::npos || slash == 0 || slash + 1 >= cidr.size()) {
    return false;
  }
  uint64_t parsed_prefix = 0;
  if (!ParseUnsigned(cidr.substr(slash + 1), 0, 128, &parsed_prefix)) {
    return false;
  }
  *ip = cidr.substr(0, slash);
  *prefix = static_cast<uint8_t>(parsed_prefix);
  return true;
}

}  // namespace

bool IpPoolParser::ParseIPv4Pool(const std::string& cidr, IPv4Pool* pool) {
  std::string ip_part;
  uint8_t prefix = 0;
  if (!SplitCidr(cidr, &ip_part, &prefix) || prefix > 32) {
    Logger::Error("Invalid fake4 CIDR: " + cidr);
    return false;
  }

  struct in_addr parsed_addr {};
  if (inet_pton(AF_INET, ip_part.c_str(), &parsed_addr) != 1) {
    Logger::Error("fake4 CIDR IP part is invalid: " + ip_part);
    return false;
  }

  const uint32_t ip = ntohl(parsed_addr.s_addr);
  const uint32_t mask = (prefix == 0) ? 0 : (0xFFFFFFFFu << static_cast<uint32_t>(32 - prefix));
  const uint32_t network = ip & mask;

  const uint32_t host_bits = static_cast<uint32_t>(32 - prefix);
  if (host_bits <= 1) {
    Logger::Error("fake4 subnet is too small. Need at least /30.");
    return false;
  }

  const uint64_t total = 1ULL << host_bits;
  const uint64_t usable = total - 2;
  const uint32_t max_count = static_cast<uint32_t>(std::min<uint64_t>(usable, kFakeIpMaxCount));
  if (max_count == 0) {
    Logger::Error("fake4 subnet cannot allocate any fake addresses.");
    return false;
  }

  pool->network = network;
  pool->max_count = max_count;
  return true;
}

bool IpPoolParser::ParseIPv6Pool(const std::string& cidr, IPv6Pool* pool) {
  std::string ip_part;
  uint8_t prefix = 0;
  if (!SplitCidr(cidr, &ip_part, &prefix) || prefix > 128) {
    Logger::Error("Invalid fake6 CIDR: " + cidr);
    return false;
  }
  if (prefix == 128) {
    Logger::Error("fake6 subnet is too small. Need host bits for allocations.");
    return false;
  }

  std::array<uint8_t, 16> raw {};
  if (inet_pton(AF_INET6, ip_part.c_str(), raw.data()) != 1) {
    Logger::Error("fake6 CIDR IP part is invalid: " + ip_part);
    return false;
  }
  const IPv6Addr parsed = IpAddress::FromRawV6(raw.data());
  const IPv6Addr network = IpAddress::ApplyPrefixMaskV6(parsed, prefix);
  const uint32_t host_bits = static_cast<uint32_t>(128 - prefix);

  uint64_t subnet_capacity = 0;
  if (host_bits >= 64) {
    subnet_capacity = std::numeric_limits<uint64_t>::max();
  } else {
    subnet_capacity = (1ULL << host_bits) - 1;
  }
  const uint64_t max_count = std::min<uint64_t>(subnet_capacity, kFakeIpMaxCount);
  if (max_count == 0) {
    Logger::Error("fake6 subnet cannot allocate any fake addresses.");
    return false;
  }

  pool->network = network;
  pool->max_count = max_count;
  return true;
}

}  // namespace fakedns
