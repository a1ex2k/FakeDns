#pragma once

#include <cstdint>
#include <string>

#include <sys/socket.h>

namespace fakedns {

struct Endpoint {
  sockaddr_storage addr {};
  socklen_t len = 0;
  int family = AF_UNSPEC;
  std::string display;
};

class NetworkUtils final {
 public:
  static bool SplitHostPort(const std::string& upstream, std::string* host, std::string* port);
  static bool ResolveUdpEndpoint(const std::string& host, const std::string& port, Endpoint* endpoint);
  static int CreateListenerSocket(const std::string& listen_ip, uint16_t port);
};

}  // namespace fakedns
