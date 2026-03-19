#include "nftables_manager.h"

#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

#include <cerrno>
#include <sstream>
#include <vector>

#include "ip_address.h"
#include "logger.h"

namespace fakedns {
namespace {

constexpr const char* kNftFamily = "inet";
constexpr const char* kNftTable = "fake_ip";
constexpr const char* kNftNatChain = "prerouting";
constexpr const char* kNftMarkChain = "prerouting_mangle";
constexpr const char* kNftNatPrio = "-101";
constexpr const char* kNftMarkPrio = "-160";

}  // namespace

bool NftablesManager::Setup() {
  std::lock_guard<std::mutex> lock(mutex_);

  RunCommand({"delete", "table", kNftFamily, kNftTable}, false);
  if (!RunCommand({"add", "table", kNftFamily, kNftTable}, true)) {
    return false;
  }
  if (!RunCommand({"add", "chain", kNftFamily, kNftTable, kNftMarkChain, "{", "type", "filter", "hook",
                   "prerouting", "priority", kNftMarkPrio, ";", "}"},
                  true)) {
    return false;
  }
  if (!RunCommand({"add", "rule", kNftFamily, kNftTable, kNftMarkChain, "meta", "mark", "set", "ct", "mark"},
                  true)) {
    return false;
  }
  if (!RunCommand({"add", "chain", kNftFamily, kNftTable, kNftNatChain, "{", "type", "nat", "hook",
                   "prerouting", "priority", kNftNatPrio, ";", "}"},
                  true)) {
    return false;
  }
  return true;
}

bool NftablesManager::AddIPv4Rule(uint32_t real_ip, uint32_t fake_ip, uint32_t fwmark) {
  const std::string fake = IpAddress::ToStringV4(fake_ip);
  const std::string real = IpAddress::ToStringV4(real_ip);
  const std::string mark = ToMarkHex(fwmark);

  std::lock_guard<std::mutex> lock(mutex_);

  if (fwmark != 0 && !RunCommand({"add", "rule", kNftFamily, kNftTable, kNftMarkChain, "ip", "daddr", fake, "ct",
                                  "state", "new", "meta", "mark", "set", "meta", "mark", "|", mark, "ct", "mark",
                                  "set", "ct", "mark", "|", mark},
                                 true)) {
    return false;
  }

  return RunCommand(
      {"add", "rule", kNftFamily, kNftTable, kNftNatChain, "ip", "daddr", fake, "dnat", "to", real}, true);
}

bool NftablesManager::AddIPv6Rule(const IPv6Addr& real_ip, const IPv6Addr& fake_ip, uint32_t fwmark) {
  const std::string fake = IpAddress::ToStringV6(fake_ip);
  const std::string real = IpAddress::ToStringV6(real_ip);
  const std::string mark = ToMarkHex(fwmark);

  std::lock_guard<std::mutex> lock(mutex_);

  if (fwmark != 0 && !RunCommand({"add", "rule", kNftFamily, kNftTable, kNftMarkChain, "ip6", "daddr", fake, "ct",
                                  "state", "new", "meta", "mark", "set", "meta", "mark", "|", mark, "ct", "mark",
                                  "set", "ct", "mark", "|", mark},
                                 true)) {
    return false;
  }

  return RunCommand(
      {"add", "rule", kNftFamily, kNftTable, kNftNatChain, "ip6", "daddr", fake, "dnat", "to", real}, true);
}

void NftablesManager::Cleanup() {
  std::lock_guard<std::mutex> lock(mutex_);
  RunCommand({"delete", "table", kNftFamily, kNftTable}, false);
}

std::string NftablesManager::ToMarkHex(uint32_t fwmark) {
  std::ostringstream ss;
  ss << "0x" << std::hex << std::nouppercase << fwmark;
  return ss.str();
}

bool NftablesManager::RunCommand(const std::vector<std::string>& args, bool log_on_error) {
  pid_t pid = fork();
  if (pid < 0) {
    if (log_on_error) {
      Logger::Error("fork() failed while running nft command.");
    }
    return false;
  }

  if (pid == 0) {
    std::vector<char*> argv;
    argv.reserve(args.size() + 2);
    argv.push_back(const_cast<char*>("nft"));
    for (const auto& arg : args) {
      argv.push_back(const_cast<char*>(arg.c_str()));
    }
    argv.push_back(nullptr);
    execvp("nft", argv.data());
    _exit(127);
  }

  int status = 0;
  while (waitpid(pid, &status, 0) < 0) {
    if (errno == EINTR) {
      continue;
    }
    if (log_on_error) {
      Logger::Error("waitpid() failed while waiting for nft command.");
    }
    return false;
  }

  if (WIFEXITED(status) && WEXITSTATUS(status) == 0) {
    return true;
  }

  if (log_on_error) {
    std::ostringstream ss;
    ss << "nft command failed (exit status " << WEXITSTATUS(status) << "): nft";
    for (const auto& arg : args) {
      ss << ' ' << arg;
    }
    Logger::Error(ss.str());
  }
  return false;
}

}  // namespace fakedns
