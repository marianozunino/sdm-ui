# SDM UI

![License](https://img.shields.io/badge/license-MIT-blue.svg)

A modern wrapper for the StrongDM CLI focused on improving developer experience on Linux platforms.

```
 ___ ___  __  __   _   _ ___
/ __|   \|  \/  | | | | |_ _|
\__ \ |) | |\/| | | |_| || |
|___/___/|_|  |_|  \___/|___|
```

## Overview

SDM UI enhances the StrongDM CLI (`sdm`) experience by providing:

- **Faster resource access**: Caches resources locally for quick access
- **User-friendly menus**: Integrates with UI tools like rofi, wofi, and fzf
- **Simplified authentication**: Manages credentials securely
- **Smart features**: Sorts resources by most recently used

## Installation

### From Source

```bash
go install github.com/marianozunino/sdm-ui@latest
```

### From Binary Releases

Download the latest release from the [Releases page](https://github.com/marianozunino/sdm-ui/releases).

## Dependencies

- [StrongDM CLI](https://www.strongdm.com/docs/admin-ui/cli-reference)
- One of the following UI tools:
  - [rofi](https://github.com/davatorium/rofi) (default)
  - [wofi](https://hg.sr.ht/~scoopta/wofi)
  - [fzf](https://github.com/junegunn/fzf)
- [zenity](https://github.com/ncruces/zenity) (for GUI password prompts)

## Configuration

Create a configuration file at `$XDG_CONFIG_HOME/sdm-ui.yaml`:

```yaml
email: "your.email@example.com"
verbose: true
blacklistPatterns:
  - ".*prod.*" # Exclude production resources
  - "*rds*" # Exclude RDS resources
```

Available settings:

| Setting           | Description                                 | Default        |
| ----------------- | ------------------------------------------- | -------------- |
| email             | Your StrongDM email address                 | (required)     |
| verbose           | Enable verbose logging                      | false          |
| dbPath            | Path to database directory                  | $XDG_DATA_HOME |
| blacklistPatterns | Regular expressions to filter out resources | []             |

## Usage

```
Usage:
  sdm-ui [command]

Available Commands:
  completion  Generate shell completion scripts
  dmenu       Open resource selector using rofi/wofi
  fzf         Open resource selector using fzf
  help        Help about any command
  list | ls   List available SDM resources
  sync        Synchronize the local resource cache
  update      Update sdm-ui to the latest version
  version     Show version information
  wipe        Clear the local resource cache

Flags:
      --config string   Config file (default "$XDG_CONFIG_HOME/sdm-ui.yaml")
  -d, --db string       Database path (default "$XDG_DATA_HOME")
  -e, --email string    Email address (required)
  -h, --help            Help about any command
  -v, --verbose         Enable verbose output
```

## Quick Start

1. Set up your configuration file with your email address
2. Run `sdm-ui sync` to cache resources
3. Use `sdm-ui dmenu` or `sdm-ui fzf` to select and connect to resources

## Tips

- `sdm-ui dmenu` works best with rofi/wofi in desktop environments
- `sdm-ui fzf` works in any terminal environment
- Use blacklist patterns to filter out resources you don't need
- The cache automatically preserves "last used" information

### Notes

- **Cross-Platform Testing**: This wrapper has only been tested in the environment where it was developed. If you encounter any issues, contributions or feedback are welcome!
- **SDM Version**: The wrapper was tested with the `sdm` version
  > sdm version 47.50.0 (874de0373de72a563021d2d884f176c9b0f387e6) (crypto)

## License

This project is licensed under the MIT License - see the LICENSE file for details.
