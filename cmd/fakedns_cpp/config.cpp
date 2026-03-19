#include "config.h"

#include <getopt.h>

#include <cerrno>
#include <cstdlib>
#include <iostream>
#include <string>
#include <unordered_map>
#include <utility>

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

bool ParseListenerSpec(const std::string& spec, ListenerConfig* out) {
  const auto separator = spec.find('@');
  if (separator == std::string::npos || separator == 0 || separator + 1 >= spec.size()) {
    return false;
  }

  uint64_t port = 0;
  uint64_t fwmark = 0;
  if (!ParseUnsigned(spec.substr(0, separator), 1, 65535, &port)) {
    return false;
  }
  if (!ParseUnsigned(spec.substr(separator + 1), 0, 0xFFFFFFFFULL, &fwmark)) {
    return false;
  }

  out->port = static_cast<uint16_t>(port);
  out->fwmark = static_cast<uint32_t>(fwmark);
  return true;
}

void PrintUsage(const char* program_name) {
  std::cerr << "Usage: " << program_name << " [OPTIONS]\n\n"
            << "Always fakes A/AAAA answers and does not use a domains file.\n\n"
            << "Required:\n"
            << "  --listener=PORT@FWMARK      Repeatable (example: 50053@0x80000000)\n\n"
            << "Optional:\n"
            << "  --listen=IP                  Listen IP (default: 127.0.0.1)\n"
            << "  --upstream=HOST:PORT         Upstream resolver (default: 8.8.8.8:53)\n"
            << "  --fake4=CIDR                 Fake IPv4 pool (default: 198.18.0.0/15)\n"
            << "  --fake6=CIDR                 Fake IPv6 pool (default: abcd:bad:c0de::/64)\n"
            << "  --workers=N                  Worker threads (default: 1)\n"
            << "  --upstream-timeout-ms=MS     Upstream UDP timeout (default: 2000)\n\n"
            << "Legacy compatibility flags (still accepted):\n"
            << "  --port=PORT --catch-all-port=PORT --fwmark=MASK\n";
}

ParseResult ParseFailure(int exit_code) {
  ParseResult result;
  result.ok = false;
  result.exit_code = exit_code;
  return result;
}

ParseResult ParseSuccess(ServerConfig config) {
  ParseResult result;
  result.ok = true;
  result.exit_code = 0;
  result.config = std::move(config);
  return result;
}

}  // namespace

ParseResult ConfigParser::Parse(int argc, char** argv) {
  ServerConfig cfg;

  static option long_options[] = {
      {"listen", required_argument, nullptr, 'l'},
      {"listener", required_argument, nullptr, 'L'},
      {"upstream", required_argument, nullptr, 'u'},
      {"fake4", required_argument, nullptr, '4'},
      {"fake6", required_argument, nullptr, '6'},
      {"workers", required_argument, nullptr, 'w'},
      {"upstream-timeout-ms", required_argument, nullptr, 't'},
      {"port", required_argument, nullptr, 1000},
      {"catch-all-port", required_argument, nullptr, 1001},
      {"fwmark", required_argument, nullptr, 1002},
      {"domains", required_argument, nullptr, 1003},     // ignored
      {"autoreload", required_argument, nullptr, 1004},  // ignored
      {"help", no_argument, nullptr, 'h'},
      {nullptr, 0, nullptr, 0}};

  while (true) {
    int option_index = 0;
    const int c = getopt_long(argc, argv, "l:L:u:4:6:w:t:h", long_options, &option_index);
    if (c == -1) {
      break;
    }

    switch (c) {
      case 'l':
        cfg.listen_ip = optarg;
        break;
      case 'L': {
        ListenerConfig listener {};
        if (!ParseListenerSpec(optarg, &listener)) {
          Logger::Error(std::string("Invalid --listener format: ") + optarg + ". Expected PORT@FWMARK.");
          return ParseFailure(1);
        }
        cfg.listeners.push_back(listener);
        break;
      }
      case 'u':
        cfg.upstream = optarg;
        break;
      case '4':
        cfg.fake4_cidr = optarg;
        break;
      case '6':
        cfg.fake6_cidr = optarg;
        break;
      case 'w': {
        uint64_t parsed_workers = 0;
        if (!ParseUnsigned(optarg, 1, 1024, &parsed_workers)) {
          Logger::Error("Invalid --workers value.");
          return ParseFailure(1);
        }
        cfg.workers = static_cast<std::size_t>(parsed_workers);
        break;
      }
      case 't': {
        uint64_t timeout = 0;
        if (!ParseUnsigned(optarg, 100, 60000, &timeout)) {
          Logger::Error("Invalid --upstream-timeout-ms value.");
          return ParseFailure(1);
        }
        cfg.upstream_timeout_ms = static_cast<int>(timeout);
        break;
      }
      case 1000: {
        uint64_t port = 0;
        if (!ParseUnsigned(optarg, 1, 65535, &port)) {
          Logger::Error("Invalid --port value.");
          return ParseFailure(1);
        }
        cfg.legacy_port = static_cast<uint16_t>(port);
        break;
      }
      case 1001: {
        uint64_t port = 0;
        if (!ParseUnsigned(optarg, 1, 65535, &port)) {
          Logger::Error("Invalid --catch-all-port value.");
          return ParseFailure(1);
        }
        cfg.legacy_catch_all_port = static_cast<uint16_t>(port);
        break;
      }
      case 1002: {
        uint64_t fwmark = 0;
        if (!ParseUnsigned(optarg, 0, 0xFFFFFFFFULL, &fwmark)) {
          Logger::Error("Invalid --fwmark value.");
          return ParseFailure(1);
        }
        cfg.legacy_fwmark = static_cast<uint32_t>(fwmark);
        break;
      }
      case 1003:
        Logger::Warn("--domains is ignored: this build always fakes all domains.");
        break;
      case 1004:
        Logger::Warn("--autoreload is ignored: no domains file is used.");
        break;
      case 'h':
        PrintUsage(argv[0]);
        return ParseFailure(0);
      default:
        PrintUsage(argv[0]);
        return ParseFailure(1);
    }
  }

  if (cfg.listeners.empty()) {
    if (cfg.legacy_port > 0) {
      ListenerConfig listener;
      listener.port = cfg.legacy_port;
      listener.fwmark = cfg.legacy_fwmark;
      cfg.listeners.push_back(listener);
    }
    if (cfg.legacy_catch_all_port > 0) {
      ListenerConfig listener;
      listener.port = cfg.legacy_catch_all_port;
      listener.fwmark = cfg.legacy_fwmark;
      cfg.listeners.push_back(listener);
    }
  }

  if (cfg.listeners.empty()) {
    Logger::Error("At least one listener is required. Use --listener=PORT@FWMARK.");
    return ParseFailure(1);
  }

  std::unordered_map<uint16_t, uint32_t> used_ports;
  std::vector<ListenerConfig> deduped;
  deduped.reserve(cfg.listeners.size());
  for (const auto& listener : cfg.listeners) {
    const auto it = used_ports.find(listener.port);
    if (it == used_ports.end()) {
      used_ports.emplace(listener.port, listener.fwmark);
      deduped.push_back(listener);
      continue;
    }
    if (it->second != listener.fwmark) {
      Logger::Error("Conflicting fwmark for the same port: " + std::to_string(listener.port));
      return ParseFailure(1);
    }
  }
  cfg.listeners.swap(deduped);

  if (cfg.workers == 0) {
    cfg.workers = 1;
  }

  return ParseSuccess(std::move(cfg));
}

}  // namespace fakedns
