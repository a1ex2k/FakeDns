#pragma once

#include <atomic>
#include <cstdint>
#include <shared_mutex>
#include <unordered_map>

#include "ip_pool.h"
#include "nftables_manager.h"

namespace fakedns {

class FakeIpManager final {
 public:
  FakeIpManager(IPv4Pool v4_pool, IPv6Pool v6_pool, NftablesManager* nftables_manager);

  bool GetFakeIPv4(uint32_t real_ip, uint32_t fwmark, uint32_t* fake_ip);
  bool GetFakeIPv6(const IPv6Addr& real_ip, uint32_t fwmark, IPv6Addr* fake_ip);

 private:
  struct IPv4Key {
    uint32_t real_ip = 0;
    uint32_t fwmark = 0;

    bool operator==(const IPv4Key& other) const { return real_ip == other.real_ip && fwmark == other.fwmark; }
  };

  struct IPv6Key {
    IPv6Addr real_ip;
    uint32_t fwmark = 0;

    bool operator==(const IPv6Key& other) const { return real_ip == other.real_ip && fwmark == other.fwmark; }
  };

  struct IPv4KeyHash {
    std::size_t operator()(const IPv4Key& key) const {
      return (static_cast<std::size_t>(key.real_ip) << 32) ^ static_cast<std::size_t>(key.fwmark);
    }
  };

  struct IPv6KeyHash {
    std::size_t operator()(const IPv6Key& key) const {
      std::size_t h1 = std::hash<uint64_t>{}(key.real_ip.hi);
      std::size_t h2 = std::hash<uint64_t>{}(key.real_ip.lo);
      std::size_t h3 = std::hash<uint32_t>{}(key.fwmark);
      return h1 ^ (h2 << 1) ^ (h3 << 7);
    }
  };

  IPv4Pool v4_pool_;
  IPv6Pool v6_pool_;
  NftablesManager* nftables_manager_ = nullptr;

  std::unordered_map<IPv4Key, uint32_t, IPv4KeyHash> v4_map_;
  std::unordered_map<IPv6Key, IPv6Addr, IPv6KeyHash> v6_map_;
  std::shared_mutex v4_mutex_;
  std::shared_mutex v6_mutex_;
  std::atomic<uint32_t> next_v4_{0};
  std::atomic<uint64_t> next_v6_{0};
};

}  // namespace fakedns
