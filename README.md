# tum

A sandy little app manager for the reMarkable. `tum` builds and installs apps
onto a reMarkable **over SSH**, as **suffixed sibling** appload entries
(`yaft-sandy`, …) that never conflict with vellum-managed apps.

```
appload/
  yaft/            ← vellum owns this (upstream yaft)
    yaft
    external.manifest.json   { "name": "YAFT" }
  yaft-sandy/      ← tum owns this (your build)
    yaft-sandy
    external.manifest.json   { "name": "YAFT (sandy)" }
```

Because the directory name ends in `-<suffix>`, tum owns exactly those
entries, vellum never sees them, `vellum upgrade` never clobbers them, and
each one shows up as its **own launcher tile** (A/B the upstream and your build
side by side).

## Install

```sh
git clone https://github.com/thecsw/tum
cd tum
go install .
```

Or just `go build .` for a `./tum` binary.

## Configure

Copy `tum.example.toml` to `./tum.toml` (or `~/.tum.toml`) and edit:

```toml
device          = "root@10.11.99.1"
default_suffix  = "sandy"
appload_root    = "/home/root/xovi/exthome/appload"
vellum          = "/home/root/.vellum/bin/vellum"
```

All of these can be overridden per-invocation with `--device` / `--suffix`.

## Use

```sh
tum doctor              # check device, vellum, appload dir
tum apps                # list available app recipes
tum build yaft          # cross-compile yaft (Docker), don't install
tum install yaft        # build + install as appload/yaft-sandy
tum install yaft --no-build   # deploy an already-built binary
tum list                # list tum-managed apps on the device
tum remove yaft        # remove yaft-sandy
tum remote add yaft    # passthrough: install upstream yaft via vellum
tum remote upgrade      # passthrough: vellum upgrade (won't touch -sandy apps)
```

Flags: `-d/--device`, `-s/--suffix`, `--dry-run`, `-v/--verbose`, `--config`,
`--recipes <dir>` (override built-in recipes with `<app>.toml` files).

## Recipes

An app is described by a recipe TOML (`internal/recipe/apps/<app>.toml`),
embedded into the binary. The `yaft` recipe is the exact build+deploy validated
against rM2-stuff:

```toml
name = "yaft"
display_name = "YAFT"
source_dir = "/path/to/rM2-stuff"
build_cmd = 'docker run --rm -v "{{.SourceDir}}:/src:cached" -w /src ...'
binary = "build/cross-armhf/apps/yaft/yaft"
binary_name = "yaft"                 # → yaft-<suffix> on device

[manifest.environment]
QTFB_SHIM_RESPECT_FULL_REFRESH_REQUESTS = "true"   # baked-in refresh fix
...
```

Add a new app by dropping a `<app>.toml` next to `yaft.toml` and rebuilding,
or override locally with `--recipes ./my-recipes`.

## How it talks to the device

tum shells out to the system `ssh`/`scp` (no Go SSH library), so it reuses your
`~/.ssh/config`, known_hosts and keys — exactly what already works for
`ssh root@10.11.99.1`. `tum remote` is a full-TTY passthrough so vellum
password prompts and progress bars work.

## Status

This is an early scaffold: `yaft` works end-to-end (build → install → list →
remove). Add recipes for more apps as needed.
