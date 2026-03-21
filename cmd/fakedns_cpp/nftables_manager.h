#pragma once

#include <cstdint>
#include <mutex>
#include <string>
#include <vector>

#include "ip_address.h"

namespace fakedns {

class NftablesManager final {
 public:
  bool Setup();
  bool AddIPv4Rule(uint32_t real_ip, uint32_t fake_ip, uint32_t fwmark);
  bool AddIPv6Rule(const IPv6Addr& real_ip, const IPv6Addr& fake_ip, uint32_t fwmark);
  void Cleanup();

 private:
  static std::string ToMarkHex(uint32_t fwmark);
  static bool RunBatchCommands(const std::vector<std::vector<std::string>>& commands, bool log_on_error);
  static bool RunCommand(const std::vector<std::string>& args, bool log_on_error);

  std::mutex mutex_;
};

}  // namespace fakedns
