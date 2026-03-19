#include "network_utils.h"

#include <arpa/inet.h>
#include <fcntl.h>
#include <netdb.h>
#include <netinet/in.h>
#include <unistd.h>

#include <cstring>
#include <string>

#include "logger.h"

namespace fakedns {
namespace {

constexpr int kListenerSocketRcvBufBytes = 512 * 1024;

bool SetSocketNonBlocking(int fd) {
  const int flags = fcntl(fd, F_GETFL, 0);
  if (flags < 0) {
    return false;
  }
  return fcntl(fd, F_SETFL, flags | O_NONBLOCK) == 0;
}

}  // namespace

bool NetworkUtils::SplitHostPort(const std::string& upstream, std::string* host, std::string* port) {
  if (upstream.empty()) {
    return false;
  }

  if (upstream.front() == '[') {
    const auto close = upstream.find(']');
    if (close == std::string::npos || close + 1 >= upstream.size() || upstream[close + 1] != ':' || close <= 1) {
      return false;
    }
    *host = upstream.substr(1, close - 1);
    *port = upstream.substr(close + 2);
    return !port->empty();
  }

  const auto separator = upstream.rfind(':');
  if (separator == std::string::npos || separator == 0 || separator + 1 >= upstream.size()) {
    return false;
  }
  *host = upstream.substr(0, separator);
  *port = upstream.substr(separator + 1);
  return true;
}

bool NetworkUtils::ResolveUdpEndpoint(const std::string& host, const std::string& port, Endpoint* endpoint) {
  struct addrinfo hints {};
  hints.ai_family = AF_UNSPEC;
  hints.ai_socktype = SOCK_DGRAM;
  hints.ai_protocol = IPPROTO_UDP;

  struct addrinfo* result = nullptr;
  const int rc = getaddrinfo(host.c_str(), port.c_str(), &hints, &result);
  if (rc != 0) {
    Logger::Error(std::string("getaddrinfo failed for upstream: ") + gai_strerror(rc));
    return false;
  }

  for (struct addrinfo* it = result; it != nullptr; it = it->ai_next) {
    if (it->ai_family != AF_INET && it->ai_family != AF_INET6) {
      continue;
    }
    if (it->ai_addrlen > static_cast<socklen_t>(sizeof(sockaddr_storage))) {
      continue;
    }
    std::memcpy(&endpoint->addr, it->ai_addr, it->ai_addrlen);
    endpoint->len = static_cast<socklen_t>(it->ai_addrlen);
    endpoint->family = it->ai_family;

    char host_buf[NI_MAXHOST] = {};
    char port_buf[NI_MAXSERV] = {};
    if (getnameinfo(it->ai_addr, static_cast<socklen_t>(it->ai_addrlen), host_buf, sizeof(host_buf), port_buf,
                    sizeof(port_buf), NI_NUMERICHOST | NI_NUMERICSERV) == 0) {
      endpoint->display = (it->ai_family == AF_INET6)
                              ? ("[" + std::string(host_buf) + "]:" + std::string(port_buf))
                              : (std::string(host_buf) + ":" + std::string(port_buf));
    } else {
      endpoint->display = host + ":" + port;
    }

    freeaddrinfo(result);
    return true;
  }

  freeaddrinfo(result);
  Logger::Error("No usable upstream UDP endpoint was resolved.");
  return false;
}

int NetworkUtils::CreateListenerSocket(const std::string& listen_ip, uint16_t port) {
  struct addrinfo hints {};
  hints.ai_family = AF_UNSPEC;
  hints.ai_socktype = SOCK_DGRAM;
  hints.ai_protocol = IPPROTO_UDP;
  hints.ai_flags = AI_PASSIVE;

  const std::string port_str = std::to_string(port);
  struct addrinfo* result = nullptr;
  const int rc = getaddrinfo(listen_ip.c_str(), port_str.c_str(), &hints, &result);
  if (rc != 0) {
    Logger::Error(std::string("getaddrinfo failed for listener ") + listen_ip + ":" + port_str + ": " +
                  gai_strerror(rc));
    return -1;
  }

  int best_fd = -1;
  for (struct addrinfo* it = result; it != nullptr; it = it->ai_next) {
    int fd = socket(it->ai_family, it->ai_socktype, it->ai_protocol);
    if (fd < 0) {
      continue;
    }

    const int one = 1;
    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one));

    const int recv_buf_size = kListenerSocketRcvBufBytes;
    setsockopt(fd, SOL_SOCKET, SO_RCVBUF, &recv_buf_size, sizeof(recv_buf_size));

    if (!SetSocketNonBlocking(fd)) {
      close(fd);
      continue;
    }

    if (bind(fd, it->ai_addr, it->ai_addrlen) == 0) {
      best_fd = fd;
      break;
    }
    close(fd);
  }

  freeaddrinfo(result);
  return best_fd;
}

}  // namespace fakedns
