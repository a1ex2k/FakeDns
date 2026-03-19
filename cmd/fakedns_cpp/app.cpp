#include "app.h"

#include <iomanip>
#include <sstream>
#include <utility>

#include "dns_packet_processor.h"
#include "fake_ip_manager.h"
#include "ip_pool.h"
#include "logger.h"
#include "network_utils.h"
#include "nftables_manager.h"
#include "signal_manager.h"
#include "udp_dns_server.h"
#include "upstream_resolver.h"

namespace fakedns {
namespace {

std::string MarkToHex(uint32_t fwmark) {
  std::ostringstream ss;
  ss << "0x" << std::hex << std::nouppercase << fwmark;
  return ss.str();
}

class NftCleanupGuard final {
 public:
  explicit NftCleanupGuard(NftablesManager* nftables_manager) : nftables_manager_(nftables_manager) {}
  ~NftCleanupGuard() {
    if (nftables_manager_ != nullptr) {
      nftables_manager_->Cleanup();
    }
  }

 private:
  NftablesManager* nftables_manager_ = nullptr;
};

}  // namespace

FakeDnsApp::FakeDnsApp(ServerConfig config) : config_(std::move(config)) {}

int FakeDnsApp::Run() {
  std::string upstream_host;
  std::string upstream_port;
  if (!NetworkUtils::SplitHostPort(config_.upstream, &upstream_host, &upstream_port)) {
    Logger::Error("Invalid --upstream format. Expected HOST:PORT or [IPv6]:PORT.");
    return 1;
  }

  Endpoint upstream_endpoint;
  if (!NetworkUtils::ResolveUdpEndpoint(upstream_host, upstream_port, &upstream_endpoint)) {
    return 1;
  }

  IPv4Pool v4_pool;
  IPv6Pool v6_pool;
  if (!IpPoolParser::ParseIPv4Pool(config_.fake4_cidr, &v4_pool) ||
      !IpPoolParser::ParseIPv6Pool(config_.fake6_cidr, &v6_pool)) {
    return 1;
  }

  Logger::Info("Fake IPv4 CIDR: " + config_.fake4_cidr);
  Logger::Info("Fake IPv6 CIDR: " + config_.fake6_cidr);
  Logger::Info("Listen IP:       " + config_.listen_ip);
  for (const auto& listener : config_.listeners) {
    Logger::Info("Listener:        " + std::to_string(listener.port) + " @ " + MarkToHex(listener.fwmark));
  }
  Logger::Info("Upstream DNS:    " + upstream_endpoint.display);
  Logger::Info("Workers:         " + std::to_string(config_.workers));

  NftablesManager nftables_manager;
  if (!nftables_manager.Setup()) {
    Logger::Error("Initial nftables setup failed.");
    return 1;
  }
  NftCleanupGuard cleanup_guard(&nftables_manager);

  FakeIpManager fake_ip_manager(v4_pool, v6_pool, &nftables_manager);
  UpstreamResolver upstream_resolver(std::move(upstream_endpoint), config_.upstream_timeout_ms);
  DnsPacketProcessor packet_processor(&fake_ip_manager);

  UdpDnsServer server(config_.listen_ip, config_.listeners, config_.workers, &upstream_resolver, &packet_processor);
  if (!server.Initialize()) {
    return 1;
  }

  SignalManager::Install();
  Logger::Info("FakeDNS C++ is running.");
  server.Run();
  Logger::Info("FakeDNS C++ stopped.");

  return 0;
}

}  // namespace fakedns
