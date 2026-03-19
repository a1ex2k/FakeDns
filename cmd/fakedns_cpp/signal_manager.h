#pragma once

namespace fakedns {

class SignalManager final {
 public:
  static void Install();
  static bool ShouldStop();
  static void RequestStop();
};

}  // namespace fakedns
