#include "fake_ip_manager.h"

#include <mutex>

#include "ip_address.h"

namespace fakedns {

FakeIpManager::FakeIpManager(IPv4Pool v4_pool, IPv6Pool v6_pool, NftablesManager* nftables_manager)
    : v4_pool_(v4_pool), v6_pool_(v6_pool), nftables_manager_(nftables_manager) {}

bool FakeIpManager::GetFakeIPv4(uint32_t real_ip, uint32_t fwmark, uint32_t* fake_ip) {
  IPv4Key key;
  key.real_ip = real_ip;
  key.fwmark = fwmark;
  {
    std::shared_lock<std::shared_mutex> lock(v4_mutex_);
    const auto it = v4_map_.find(key);
    if (it != v4_map_.end()) {
      *fake_ip = it->second;
      return true;
    }
  }

  std::unique_lock<std::shared_mutex> lock(v4_mutex_);
  const auto it = v4_map_.find(key);
  if (it != v4_map_.end()) {
    *fake_ip = it->second;
    return true;
  }

  const uint32_t index = next_v4_.fetch_add(1, std::memory_order_relaxed) + 1;
  if (index > v4_pool_.max_count) {
    return false;
  }

  const uint32_t generated = v4_pool_.network + index;
  if (!nftables_manager_->AddIPv4Rule(real_ip, generated, fwmark)) {
    return false;
  }
  v4_map_.emplace(key, generated);
  *fake_ip = generated;
  return true;
}

bool FakeIpManager::GetFakeIPv6(const IPv6Addr& real_ip, uint32_t fwmark, IPv6Addr* fake_ip) {
  IPv6Key key;
  key.real_ip = real_ip;
  key.fwmark = fwmark;
  {
    std::shared_lock<std::shared_mutex> lock(v6_mutex_);
    const auto it = v6_map_.find(key);
    if (it != v6_map_.end()) {
      *fake_ip = it->second;
      return true;
    }
  }

  std::unique_lock<std::shared_mutex> lock(v6_mutex_);
  const auto it = v6_map_.find(key);
  if (it != v6_map_.end()) {
    *fake_ip = it->second;
    return true;
  }

  const uint64_t index = next_v6_.fetch_add(1, std::memory_order_relaxed) + 1;
  if (index > v6_pool_.max_count) {
    return false;
  }

  const IPv6Addr generated = IpAddress::AddV6(v6_pool_.network, index);
  if (!nftables_manager_->AddIPv6Rule(real_ip, generated, fwmark)) {
    return false;
  }
  v6_map_.emplace(key, generated);
  *fake_ip = generated;
  return true;
}

}  // namespace fakedns
