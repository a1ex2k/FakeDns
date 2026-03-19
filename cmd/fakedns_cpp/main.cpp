#include "app.h"
#include "config.h"

#include <utility>

int main(int argc, char** argv) {
  fakedns::ParseResult parse_result = fakedns::ConfigParser::Parse(argc, argv);
  if (!parse_result.ok) {
    return parse_result.exit_code;
  }

  fakedns::FakeDnsApp app(std::move(parse_result.config));
  return app.Run();
}
