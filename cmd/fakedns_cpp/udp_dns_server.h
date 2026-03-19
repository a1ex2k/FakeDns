#pragma once

#include <cstdint>
#include <string>
#include <thread>
#include <vector>

#include <sys/socket.h>

#include "config.h"
#include "dns_packet_processor.h"
#include "task_queue.h"
#include "upstream_resolver.h"

namespace fakedns {

class UdpDnsServer final {
 public:
  UdpDnsServer(std::string listen_ip, std::vector<ListenerConfig> listeners, std::size_t worker_count,
               const UpstreamResolver* upstream_resolver, const DnsPacketProcessor* packet_processor);
  ~UdpDnsServer();

  bool Initialize();
  void Run();

 private:
  struct Task {
    int listener_fd = -1;
    uint32_t fwmark = 0;
    sockaddr_storage client_addr {};
    socklen_t client_addr_len = 0;
    std::vector<uint8_t> query;
  };

  struct ListenerSocket {
    int fd = -1;
    ListenerConfig config;
  };

  void WorkerLoop();
  void ReceiverLoop();
  void CleanupListeners();
  void StopWorkers();

  std::string listen_ip_;
  std::vector<ListenerConfig> listener_configs_;
  std::vector<ListenerSocket> listeners_;

  std::size_t worker_count_ = 1;
  const UpstreamResolver* upstream_resolver_ = nullptr;
  const DnsPacketProcessor* packet_processor_ = nullptr;

  TaskQueue<Task> queue_;
  std::vector<std::thread> workers_;
};

}  // namespace fakedns
