#pragma once

#include <vector>

#include "network_utils.h"

namespace fakedns {

class UpstreamResolver final {
 public:
  UpstreamResolver(Endpoint endpoint, int timeout_ms);

  bool Resolve(const std::vector<uint8_t>& query, std::vector<uint8_t>* response) const;

 private:
  bool SendAndReceive(int socket_fd, const std::vector<uint8_t>& query, std::vector<uint8_t>* response) const;
  int EnsureSocket() const;
  void ResetSocket() const;

  Endpoint endpoint_;
  int timeout_ms_ = 2000;

  static thread_local int tls_socket_fd_;
  static thread_local int tls_socket_family_;
  static thread_local socklen_t tls_socket_addr_len_;
  static thread_local sockaddr_storage tls_socket_addr_;
};

}  // namespace fakedns
