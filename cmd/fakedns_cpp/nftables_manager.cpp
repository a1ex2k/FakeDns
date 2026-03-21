#include "nftables_manager.h"

#include <sys/types.h>
#include <sys/socket.h>
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
constexpr const char* kNftNatChain = "prerouting_nat";
constexpr const char* kNftMarkChain = "prerouting_mangle";
constexpr const char* kNftNatPrio = "-101";
constexpr const char* kNftMarkPrio = "-160";
constexpr const char* kNftDnat4Map = "dnat4_map";
constexpr const char* kNftDnat6Map = "dnat6_map";
constexpr const char* kNftMark4Map = "mark4_map";
constexpr const char* kNftMark6Map = "mark6_map";
constexpr const char* kNftDnat4MapRef = "@dnat4_map";
constexpr const char* kNftDnat6MapRef = "@dnat6_map";
constexpr const char* kNftMark4MapRef = "@mark4_map";
constexpr const char* kNftMark6MapRef = "@mark6_map";

std::string JoinNftCommand(const std::vector<std::string>& command) {
  std::ostringstream ss;
  for (std::size_t i = 0; i < command.size(); ++i) {
    if (i > 0) {
      ss << ' ';
    }
    ss << command[i];
  }
  return ss.str();
}

}  // namespace

bool NftablesManager::Setup() {
  std::lock_guard<std::mutex> lock(mutex_);

  RunCommand({"delete", "table", kNftFamily, kNftTable}, false);
  return RunBatchCommands(
      {
          {"add", "table", kNftFamily, kNftTable},
          {"add", "map", kNftFamily, kNftTable, kNftDnat4Map, "{", "type", "ipv4_addr", ":", "ipv4_addr", ";", "}"},
          {"add", "map", kNftFamily, kNftTable, kNftDnat6Map, "{", "type", "ipv6_addr", ":", "ipv6_addr", ";", "}"},
          {"add", "map", kNftFamily, kNftTable, kNftMark4Map, "{", "type", "ipv4_addr", ":", "mark", ";", "}"},
          {"add", "map", kNftFamily, kNftTable, kNftMark6Map, "{", "type", "ipv6_addr", ":", "mark", ";", "}"},
          {"add", "chain", kNftFamily, kNftTable, kNftMarkChain, "{", "type", "filter", "hook", "prerouting",
           "priority", kNftMarkPrio, ";", "policy", "accept", ";", "}"},
          {"add", "rule", kNftFamily, kNftTable, kNftMarkChain, "meta", "mark", "set", "ct", "mark"},
          {"add", "rule", kNftFamily, kNftTable, kNftMarkChain, "ct", "state", "new", "meta", "mark", "set", "ip",
           "daddr", "map", kNftMark4MapRef, "ct", "mark", "set", "ip", "daddr", "map", kNftMark4MapRef},
          {"add", "rule", kNftFamily, kNftTable, kNftMarkChain, "ct", "state", "new", "meta", "mark", "set", "ip6",
           "daddr", "map", kNftMark6MapRef, "ct", "mark", "set", "ip6", "daddr", "map", kNftMark6MapRef},
          {"add", "chain", kNftFamily, kNftTable, kNftNatChain, "{", "type", "nat", "hook", "prerouting", "priority",
           kNftNatPrio, ";", "policy", "accept", ";", "}"},
          {"add", "rule", kNftFamily, kNftTable, kNftNatChain, "dnat", "to", "ip", "daddr", "map", kNftDnat4MapRef},
          {"add", "rule", kNftFamily, kNftTable, kNftNatChain, "dnat", "to", "ip6", "daddr", "map", kNftDnat6MapRef},
      },
      true);
}

bool NftablesManager::AddIPv4Rule(uint32_t real_ip, uint32_t fake_ip, uint32_t fwmark) {
  const std::string fake = IpAddress::ToStringV4(fake_ip);
  const std::string real = IpAddress::ToStringV4(real_ip);
  const std::string mark = ToMarkHex(fwmark);

  std::lock_guard<std::mutex> lock(mutex_);

  std::vector<std::vector<std::string>> commands;
  commands.push_back({"add", "element", kNftFamily, kNftTable, kNftDnat4Map, "{", fake, ":", real, "}"});
  if (fwmark != 0) {
    commands.push_back({"add", "element", kNftFamily, kNftTable, kNftMark4Map, "{", fake, ":", mark, "}"});
  }

  return RunBatchCommands(commands, true);
}

bool NftablesManager::AddIPv6Rule(const IPv6Addr& real_ip, const IPv6Addr& fake_ip, uint32_t fwmark) {
  const std::string fake = IpAddress::ToStringV6(fake_ip);
  const std::string real = IpAddress::ToStringV6(real_ip);
  const std::string mark = ToMarkHex(fwmark);

  std::lock_guard<std::mutex> lock(mutex_);

  std::vector<std::vector<std::string>> commands;
  commands.push_back({"add", "element", kNftFamily, kNftTable, kNftDnat6Map, "{", fake, ":", real, "}"});
  if (fwmark != 0) {
    commands.push_back({"add", "element", kNftFamily, kNftTable, kNftMark6Map, "{", fake, ":", mark, "}"});
  }

  return RunBatchCommands(commands, true);
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

bool NftablesManager::RunBatchCommands(const std::vector<std::vector<std::string>>& commands, bool log_on_error) {
  if (commands.empty()) {
    return true;
  }

  int socket_fds[2] = {-1, -1};
  if (socketpair(AF_UNIX, SOCK_STREAM, 0, socket_fds) != 0) {
    if (log_on_error) {
      Logger::Error("socketpair() failed while preparing nft batch command.");
    }
    return false;
  }

  pid_t pid = fork();
  if (pid < 0) {
    close(socket_fds[0]);
    close(socket_fds[1]);
    if (log_on_error) {
      Logger::Error("fork() failed while running nft batch command.");
    }
    return false;
  }

  if (pid == 0) {
    close(socket_fds[0]);
    if (dup2(socket_fds[1], STDIN_FILENO) < 0) {
      _exit(127);
    }
    close(socket_fds[1]);
    char* argv[] = {const_cast<char*>("nft"), const_cast<char*>("-f"), const_cast<char*>("-"), nullptr};
    execvp("nft", argv);
    _exit(127);
  }

  close(socket_fds[1]);

  std::ostringstream script_stream;
  for (const auto& command : commands) {
    script_stream << JoinNftCommand(command) << '\n';
  }
  const std::string script = script_stream.str();

  bool write_ok = true;
  std::size_t offset = 0;
  while (offset < script.size()) {
    const ssize_t written = send(socket_fds[0], script.data() + offset, script.size() - offset, MSG_NOSIGNAL);
    if (written < 0) {
      if (errno == EINTR) {
        continue;
      }
      write_ok = false;
      break;
    }
    offset += static_cast<std::size_t>(written);
  }
  shutdown(socket_fds[0], SHUT_WR);
  close(socket_fds[0]);

  int status = 0;
  while (waitpid(pid, &status, 0) < 0) {
    if (errno == EINTR) {
      continue;
    }
    if (log_on_error) {
      Logger::Error("waitpid() failed while waiting for nft batch command.");
    }
    return false;
  }

  if (write_ok && WIFEXITED(status) && WEXITSTATUS(status) == 0) {
    return true;
  }

  if (log_on_error) {
    std::ostringstream ss;
    ss << "nft batch command failed";
    if (WIFEXITED(status)) {
      ss << " (exit status " << WEXITSTATUS(status) << ")";
    }
    ss << ". Commands:";
    for (const auto& command : commands) {
      ss << "\n  " << JoinNftCommand(command);
    }
    if (!write_ok) {
      ss << "\n  [writer error while sending batch to nft]";
    }
    Logger::Error(ss.str());
  }
  return false;
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
