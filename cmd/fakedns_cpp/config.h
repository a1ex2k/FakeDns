#pragma once

#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

namespace fakedns {

struct ListenerConfig {
  uint16_t port = 0;
  uint32_t fwmark = 0;
};

struct ServerConfig {
  std::string listen_ip = "127.0.0.1";
  std::vector<ListenerConfig> listeners;
  std::string upstream = "8.8.8.8:53";
  std::string fake4_cidr = "198.18.0.0/15";
  std::string fake6_cidr = "abcd:bad:c0de::/64";
  std::size_t workers = 0;  // auto -> 1 worker
  int upstream_timeout_ms = 2000;

  // Backward compatibility flags.
  uint16_t legacy_port = 0;
  uint16_t legacy_catch_all_port = 0;
  uint32_t legacy_fwmark = 0;
};

struct ParseResult {
  bool ok = false;
  int exit_code = 1;
  ServerConfig config;
};

class ConfigParser final {
 public:
  static ParseResult Parse(int argc, char** argv);
};

}  // namespace fakedns
