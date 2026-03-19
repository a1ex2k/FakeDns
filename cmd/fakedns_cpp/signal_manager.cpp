#include "signal_manager.h"

#include <signal.h>

#include <atomic>

namespace fakedns {
namespace {

std::atomic<bool> g_stop_requested{false};

void HandleStopSignal(int /*signal*/) { g_stop_requested.store(true, std::memory_order_relaxed); }

}  // namespace

void SignalManager::Install() {
  struct sigaction stop_action {};
  stop_action.sa_handler = HandleStopSignal;
  sigemptyset(&stop_action.sa_mask);
  stop_action.sa_flags = 0;

  sigaction(SIGINT, &stop_action, nullptr);
  sigaction(SIGTERM, &stop_action, nullptr);

  // Keep compatibility with systemd ExecReload without terminating the process.
  struct sigaction hup_action {};
  hup_action.sa_handler = SIG_IGN;
  sigemptyset(&hup_action.sa_mask);
  hup_action.sa_flags = 0;
  sigaction(SIGHUP, &hup_action, nullptr);
}

bool SignalManager::ShouldStop() { return g_stop_requested.load(std::memory_order_relaxed); }

void SignalManager::RequestStop() { g_stop_requested.store(true, std::memory_order_relaxed); }

}  // namespace fakedns
