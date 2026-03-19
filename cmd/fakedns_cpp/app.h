#pragma once

#include "config.h"

namespace fakedns {

class FakeDnsApp final {
 public:
  explicit FakeDnsApp(ServerConfig config);
  int Run();

 private:
  ServerConfig config_;
};

}  // namespace fakedns
