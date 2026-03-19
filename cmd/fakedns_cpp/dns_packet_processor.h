#pragma once

#include <cstdint>
#include <vector>

#include "fake_ip_manager.h"

namespace fakedns {

class DnsPacketProcessor final {
 public:
  explicit DnsPacketProcessor(FakeIpManager* fake_ip_manager);

  bool PatchAnswers(std::vector<uint8_t>* response, uint32_t fwmark) const;
  static std::vector<uint8_t> BuildServFailResponse(const std::vector<uint8_t>& query);

 private:
  static bool SkipDnsName(const uint8_t* data, std::size_t size, std::size_t* offset);
  static uint16_t ReadU16(const uint8_t* p);
  static uint32_t ReadU32(const uint8_t* p);
  static void WriteU16(uint8_t* p, uint16_t value);
  static void WriteU32(uint8_t* p, uint32_t value);

  FakeIpManager* fake_ip_manager_ = nullptr;
};

}  // namespace fakedns
