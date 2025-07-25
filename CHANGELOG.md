# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.5.0] - 2025-07-22

- Add temporal_extents to collection output
- `--download-skip-checksum` flag to skip downloads when local files exist with matching checksum
- Undeprecate granules `--shortname` flag
- Add granules `--version` flag

## [v0.4.3] - 2025-01-31

### Fixed

- Deadlock when number of results required pagination (> 200 results)

## [v0.4.2] - 2025-01-01

### Added

- Check for inadvertent HTML content when downloading files

### Fixed

- Checksum failure when downloading using EDL token (#8)




