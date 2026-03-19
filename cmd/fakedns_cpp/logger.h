#pragma once

#include <iostream>
#include <string_view>

namespace fakedns {

class Logger final {
 public:
  static void Info(std::string_view message) { std::cerr << "[INFO] " << message << '\n'; }
  static void Warn(std::string_view message) { std::cerr << "[WARN] " << message << '\n'; }
  static void Error(std::string_view message) { std::cerr << "[ERROR] " << message << '\n'; }
};

}  // namespace fakedns
