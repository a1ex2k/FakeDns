#include "upstream_resolver.h"

#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <unistd.h>

#include <cstring>
#include <utility>

namespace fakedns {
namespace {

constexpr std::size_t kUpstreamUdpMaxResponseBytes = 4096;

}  // namespace

thread_local int UpstreamResolver::tls_socket_fd_ = -1;
thread_local int UpstreamResolver::tls_socket_family_ = AF_UNSPEC;
thread_local socklen_t UpstreamResolver::tls_socket_addr_len_ = 0;
thread_local sockaddr_storage UpstreamResolver::tls_socket_addr_ {};

UpstreamResolver::UpstreamResolver(Endpoint endpoint, int timeout_ms)
    : endpoint_(std::move(endpoint)), timeout_ms_(timeout_ms) {}

bool UpstreamResolver::Resolve(const std::vector<uint8_t>& query, std::vector<uint8_t>* response) const {
  int socket_fd = EnsureSocket();
  if (socket_fd < 0) {
    return false;
  }
  if (SendAndReceive(socket_fd, query, response)) {
    return true;
  }

  ResetSocket();
  socket_fd = EnsureSocket();
  if (socket_fd < 0) {
    return false;
  }
  return SendAndReceive(socket_fd, query, response);
}

bool UpstreamResolver::SendAndReceive(int socket_fd, const std::vector<uint8_t>& query,
                                      std::vector<uint8_t>* response) const {
  const ssize_t sent = send(socket_fd, query.data(), query.size(), 0);
  if (sent != static_cast<ssize_t>(query.size())) {
    return false;
  }

  response->resize(kUpstreamUdpMaxResponseBytes);
  const ssize_t received = recv(socket_fd, response->data(), response->size(), 0);
  if (received <= 0) {
    return false;
  }
  response->resize(static_cast<std::size_t>(received));
  return true;
}

int UpstreamResolver::EnsureSocket() const {
  if (tls_socket_fd_ >= 0 && tls_socket_family_ == endpoint_.family && tls_socket_addr_len_ == endpoint_.len &&
      std::memcmp(&tls_socket_addr_, &endpoint_.addr, endpoint_.len) == 0) {
    return tls_socket_fd_;
  }
  ResetSocket();

  tls_socket_fd_ = socket(endpoint_.family, SOCK_DGRAM, IPPROTO_UDP);
  if (tls_socket_fd_ < 0) {
    return -1;
  }

  struct timeval timeout {};
  timeout.tv_sec = timeout_ms_ / 1000;
  timeout.tv_usec = (timeout_ms_ % 1000) * 1000;
  setsockopt(tls_socket_fd_, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
  setsockopt(tls_socket_fd_, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));

  if (connect(tls_socket_fd_, reinterpret_cast<const sockaddr*>(&endpoint_.addr), endpoint_.len) != 0) {
    ResetSocket();
    return -1;
  }

  tls_socket_family_ = endpoint_.family;
  tls_socket_addr_len_ = endpoint_.len;
  std::memcpy(&tls_socket_addr_, &endpoint_.addr, endpoint_.len);
  return tls_socket_fd_;
}

void UpstreamResolver::ResetSocket() const {
  if (tls_socket_fd_ >= 0) {
    close(tls_socket_fd_);
  }
  tls_socket_fd_ = -1;
  tls_socket_family_ = AF_UNSPEC;
  tls_socket_addr_len_ = 0;
  std::memset(&tls_socket_addr_, 0, sizeof(tls_socket_addr_));
}

}  // namespace fakedns
