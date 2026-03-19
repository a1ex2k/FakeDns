#include "udp_dns_server.h"

#include <poll.h>
#include <unistd.h>

#include <algorithm>
#include <array>
#include <cerrno>
#include <utility>

#include "logger.h"
#include "network_utils.h"
#include "signal_manager.h"

namespace fakedns {
namespace {

constexpr std::size_t kMaxUdpPacketSize = 4096;
constexpr std::size_t kTaskQueueMinDepth = 1024;
constexpr std::size_t kTaskQueueDepthPerWorker = 256;

}  // namespace

UdpDnsServer::UdpDnsServer(std::string listen_ip, std::vector<ListenerConfig> listeners, std::size_t worker_count,
                           const UpstreamResolver* upstream_resolver, const DnsPacketProcessor* packet_processor)
    : listen_ip_(std::move(listen_ip)),
      listener_configs_(std::move(listeners)),
      worker_count_(worker_count),
      upstream_resolver_(upstream_resolver),
      packet_processor_(packet_processor),
      queue_(std::max<std::size_t>(kTaskQueueMinDepth, worker_count_ * kTaskQueueDepthPerWorker)) {}

UdpDnsServer::~UdpDnsServer() {
  StopWorkers();
  CleanupListeners();
}

bool UdpDnsServer::Initialize() {
  listeners_.reserve(listener_configs_.size());
  for (const auto& cfg : listener_configs_) {
    const int fd = NetworkUtils::CreateListenerSocket(listen_ip_, cfg.port);
    if (fd < 0) {
      Logger::Error("Failed to bind listener on " + listen_ip_ + ":" + std::to_string(cfg.port));
      CleanupListeners();
      return false;
    }
    ListenerSocket listener;
    listener.fd = fd;
    listener.config = cfg;
    listeners_.push_back(listener);
  }
  return true;
}

void UdpDnsServer::Run() {
  workers_.reserve(worker_count_);
  for (std::size_t i = 0; i < worker_count_; ++i) {
    workers_.emplace_back(&UdpDnsServer::WorkerLoop, this);
  }

  ReceiverLoop();
  StopWorkers();
}

void UdpDnsServer::WorkerLoop() {
  Task task;
  std::vector<uint8_t> response;

  while (queue_.Pop(&task)) {
    response.clear();
    if (upstream_resolver_->Resolve(task.query, &response)) {
      packet_processor_->PatchAnswers(&response, task.fwmark);
    } else {
      response = DnsPacketProcessor::BuildServFailResponse(task.query);
      if (response.empty()) {
        continue;
      }
    }

    sendto(task.listener_fd, response.data(), response.size(), 0, reinterpret_cast<const sockaddr*>(&task.client_addr),
           task.client_addr_len);
  }
}

void UdpDnsServer::ReceiverLoop() {
  std::vector<pollfd> poll_fds;
  poll_fds.reserve(listeners_.size());
  for (const auto& listener : listeners_) {
    pollfd pfd {};
    pfd.fd = listener.fd;
    pfd.events = POLLIN;
    pfd.revents = 0;
    poll_fds.push_back(pfd);
  }

  while (!SignalManager::ShouldStop()) {
    const int rc = poll(poll_fds.data(), poll_fds.size(), 250);
    if (rc < 0) {
      if (errno == EINTR) {
        continue;
      }
      Logger::Error("poll() failed in receiver loop.");
      SignalManager::RequestStop();
      break;
    }
    if (rc == 0) {
      continue;
    }

    for (std::size_t i = 0; i < poll_fds.size(); ++i) {
      if ((poll_fds[i].revents & POLLIN) == 0) {
        continue;
      }

      while (!SignalManager::ShouldStop()) {
        std::array<uint8_t, kMaxUdpPacketSize> buffer {};
        sockaddr_storage client_addr {};
        socklen_t client_len = sizeof(client_addr);
        const ssize_t n = recvfrom(poll_fds[i].fd, buffer.data(), buffer.size(), 0,
                                   reinterpret_cast<sockaddr*>(&client_addr), &client_len);
        if (n < 0) {
          if (errno == EAGAIN || errno == EWOULDBLOCK) {
            break;
          }
          if (errno == EINTR) {
            continue;
          }
          break;
        }
        if (n == 0) {
          break;
        }

        Task task;
        task.listener_fd = listeners_[i].fd;
        task.fwmark = listeners_[i].config.fwmark;
        task.client_addr = client_addr;
        task.client_addr_len = client_len;
        task.query.assign(buffer.begin(), buffer.begin() + static_cast<std::size_t>(n));

        if (!queue_.Push(std::move(task))) {
          return;
        }
      }
    }
  }
}

void UdpDnsServer::CleanupListeners() {
  for (auto& listener : listeners_) {
    if (listener.fd >= 0) {
      close(listener.fd);
      listener.fd = -1;
    }
  }
  listeners_.clear();
}

void UdpDnsServer::StopWorkers() {
  queue_.Stop();
  for (auto& worker : workers_) {
    if (worker.joinable()) {
      worker.join();
    }
  }
  workers_.clear();
}

}  // namespace fakedns
