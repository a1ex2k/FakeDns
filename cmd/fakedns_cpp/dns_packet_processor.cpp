#include "dns_packet_processor.h"

#include "ip_address.h"

namespace fakedns {
namespace {

constexpr uint32_t kFakedRecordTtl = 300;

}  // namespace

DnsPacketProcessor::DnsPacketProcessor(FakeIpManager* fake_ip_manager) : fake_ip_manager_(fake_ip_manager) {}

bool DnsPacketProcessor::PatchAnswers(std::vector<uint8_t>* response, uint32_t fwmark) const {
  if (response->size() < 12) {
    return false;
  }

  uint8_t* data = response->data();
  const std::size_t size = response->size();
  const uint16_t question_count = ReadU16(data + 4);
  const uint16_t answer_count = ReadU16(data + 6);

  std::size_t offset = 12;
  for (uint16_t i = 0; i < question_count; ++i) {
    if (!SkipDnsName(data, size, &offset) || offset + 4 > size) {
      return false;
    }
    offset += 4;
  }

  for (uint16_t i = 0; i < answer_count; ++i) {
    if (!SkipDnsName(data, size, &offset) || offset + 10 > size) {
      return false;
    }

    const uint16_t rr_type = ReadU16(data + offset);
    const uint16_t rr_class = ReadU16(data + offset + 2);
    const std::size_t ttl_offset = offset + 4;
    const uint16_t rd_length = ReadU16(data + offset + 8);
    const std::size_t rdata_offset = offset + 10;
    if (rdata_offset + rd_length > size) {
      return false;
    }

    if (rr_class == 1 && rr_type == 1 && rd_length == 4) {
      const uint32_t real_ip = ReadU32(data + rdata_offset);
      uint32_t fake_ip = 0;
      if (fake_ip_manager_->GetFakeIPv4(real_ip, fwmark, &fake_ip)) {
        WriteU32(data + rdata_offset, fake_ip);
        WriteU32(data + ttl_offset, kFakedRecordTtl);
      }
    } else if (rr_class == 1 && rr_type == 28 && rd_length == 16) {
      const IPv6Addr real_ip = IpAddress::FromRawV6(data + rdata_offset);
      IPv6Addr fake_ip {};
      if (fake_ip_manager_->GetFakeIPv6(real_ip, fwmark, &fake_ip)) {
        IpAddress::ToRawV6(fake_ip, data + rdata_offset);
        WriteU32(data + ttl_offset, kFakedRecordTtl);
      }
    }

    offset = rdata_offset + rd_length;
  }
  return true;
}

std::vector<uint8_t> DnsPacketProcessor::BuildServFailResponse(const std::vector<uint8_t>& query) {
  if (query.size() < 12) {
    return {};
  }
  std::vector<uint8_t> out = query;
  out[2] = static_cast<uint8_t>(out[2] | 0x80U);
  out[3] = static_cast<uint8_t>((out[3] & 0xF0U) | 0x02U);
  WriteU16(out.data() + 6, 0);
  WriteU16(out.data() + 8, 0);
  WriteU16(out.data() + 10, 0);
  return out;
}

bool DnsPacketProcessor::SkipDnsName(const uint8_t* data, std::size_t size, std::size_t* offset) {
  std::size_t position = *offset;
  std::size_t consumed = 0;
  bool jumped = false;
  int jump_count = 0;

  while (true) {
    if (position >= size) {
      return false;
    }
    const uint8_t length = data[position];

    if ((length & 0xC0U) == 0xC0U) {
      if (position + 1 >= size) {
        return false;
      }
      const uint16_t pointer = static_cast<uint16_t>(((length & 0x3FU) << 8) | data[position + 1]);
      if (pointer >= size) {
        return false;
      }
      if (!jumped) {
        consumed += 2;
      }
      position = pointer;
      jumped = true;
      if (++jump_count > 32) {
        return false;
      }
      continue;
    }

    if ((length & 0xC0U) != 0) {
      return false;
    }

    if (length == 0) {
      if (!jumped) {
        consumed += 1;
      }
      break;
    }

    ++position;
    if (position + length > size) {
      return false;
    }
    if (!jumped) {
      consumed += static_cast<std::size_t>(length) + 1;
    }
    position += length;
  }

  *offset += consumed;
  return true;
}

uint16_t DnsPacketProcessor::ReadU16(const uint8_t* p) {
  return static_cast<uint16_t>((static_cast<uint16_t>(p[0]) << 8) | p[1]);
}

uint32_t DnsPacketProcessor::ReadU32(const uint8_t* p) {
  return (static_cast<uint32_t>(p[0]) << 24) | (static_cast<uint32_t>(p[1]) << 16) |
         (static_cast<uint32_t>(p[2]) << 8) | static_cast<uint32_t>(p[3]);
}

void DnsPacketProcessor::WriteU16(uint8_t* p, uint16_t value) {
  p[0] = static_cast<uint8_t>((value >> 8) & 0xFF);
  p[1] = static_cast<uint8_t>(value & 0xFF);
}

void DnsPacketProcessor::WriteU32(uint8_t* p, uint32_t value) {
  p[0] = static_cast<uint8_t>((value >> 24) & 0xFF);
  p[1] = static_cast<uint8_t>((value >> 16) & 0xFF);
  p[2] = static_cast<uint8_t>((value >> 8) & 0xFF);
  p[3] = static_cast<uint8_t>(value & 0xFF);
}

}  // namespace fakedns
