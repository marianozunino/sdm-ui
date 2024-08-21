# SDM Wrapper

## Why Use the SDM Wrapper?

- **Performance**: The `sdm` command can be slow, especially when listing statuses/resources (`sdm status`). This issue might be exacerbated if you're outside the US.
- **User Interface**: `sdm` lacks a UI for Linux, which, combined with its performance issues, makes the experience less than ideal.
- **Personal Challenge**: Because it's a fun project and a great learning opportunity.

## Installation

To install the SDM Wrapper, run:

```bash
go install github.com/marianozunino/sdm-ui@latest
```

## Usage

```
Usage:
  sdm-ui [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  dmenu       Opens dmenu with available data sources
  fzf         Opens fzf with available data sources
  help        Help about any command
  list        List SDM resources
  sync        Syncronizes the internal cache
  version     Print the version number of sdm ui
  wipe        Wipe the SDM UI cache db

Flags:
      --config string   config file (default "/home/forbi/.config/sdm-ui.yaml")
  -d, --db string       database path (default "/home/forbi/.local/share")
  -e, --email string    email address (overrides config file)
  -h, --help            help for sdm-ui
  -v, --verbose         verbose output (overrides config file)
```

### Configuration

The SDM Wrapper can be configured using a YAML file located at `$XDG_CONFIG_HOME/sdm-ui.yaml`. Here's an example configuration:

```yaml
email: some_email@example.com
verbose: true
blacklistPatterns:
  - "*rds*"
  - "prod*"
  - "es-logs$"
```

The available configuration options are:

- `email`: Your email address used for authentication with the StrongDM platform.
- `verbose`: Enable verbose output.
- `blacklistPatterns`: A list of regular expression patterns used to filter out unwanted data sources.

You can also override the configuration options using command-line flags, as shown in the Usage section.

### How Does the Wrapper Address These Issues?

#### Slow Status

The typical workflow with `sdm` involves:

1. Running `sdm status | grep <something>` to filter the list of resources (e.g., `sdm status | grep rds` to find RDS resources).
2. Using `sdm connect <resource>` to connect to the selected resource.

The SDM Wrapper improves this by caching the resource list using [bbolt](https://github.com/etcd-io/bbolt). This makes resource retrieval faster and more efficient. The cache is populated automatically when you connect to a resource.

#### Lack of UI

While I'm not a UI expert, I appreciate efficiency. The wrapper integrates with [rofi](https://github.com/DaveDavenport/rofi), [wofi](https://sr.ht/~scoopta/wofi/), or [fzf](https://github.com/junegunn/fzf) to provide a user-friendly interface for selecting resources.

Credential management is handled using [keyring](https://github.com/tmc/keyring), and if credentials are missing, the wrapper prompts for them via [zenity](https://github.com/ncruces/zenity).

Additionally, unlike the macOS version of `sdm`, which opens a browser tab for web resources, the wrapper uses [open](https://github.com/skratchdot/open-golang) to achieve the same on Linux.

### Notes

- **Cross-Platform Testing**: This wrapper has only been tested in the environment where it was developed. If you encounter any issues, contributions or feedback are welcome!
- **SDM Version**: The wrapper was tested with the `sdm` version 44.31.0.
