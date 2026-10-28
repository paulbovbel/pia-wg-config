# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `--gateway-file` flag writing the selected server's `server_vip`, the in-tunnel
  address of PIA's port forwarding API
- `-p, --port-forwarding` flag refusing regions that do not offer port forwarding
- `SelectedServerVip()` and `RequirePortForwarding()` on the PIA client
- `regions` command to list all available PIA regions
- Enhanced CLI help text to emphasize region configurability
- `GetAvailableRegions()` method to PIA client
- Comprehensive documentation improvements
- Contributing guidelines (CONTRIBUTING.md)
- Examples for popular regions in README
- Integration examples (Docker, Bash scripts)
- Troubleshooting section in documentation

### Changed
- Improved README with clear emphasis on region selection
- Enhanced CLI flag description for region parameter
- Better error messages and help text

### Fixed
- Registration no longer always picks the region's first server. PIA orders the
  list deterministically, so a server that refuses keys or lacks the port
  forwarding service used to break every attempt until PIA reshuffled; the
  region's servers are now tried in a random order until one accepts.
- `addKey` now checks the `status` field, which PIA uses to report refusal
  instead of an HTTP status code, rather than accepting the response regardless
- An empty region returns an error instead of calling `log.Fatalf`
- `addKey` and non-200 PIA responses no longer leak the response body
- Clarified that regions are NOT hardcoded but configurable via CLI flags

## [Previous Versions]

### Added
- Initial release with Wireguard config generation
- Support for all PIA regions via `-r/--region` flag
- File output option via `-o/--outfile` flag
- Verbose logging option via `-v/--verbose` flag
- MIT License
- Basic README documentation

### Technical Details
- Built with Go 1.23+
- Uses PIA's official API endpoints
- Self-contained binary with no external dependencies
- Cross-platform compatibility