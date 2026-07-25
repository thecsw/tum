# AGENTS.md — tum

Guidance for AI coding agents (Codex, etc.) working in this repo. Read this
before touching anything. It captures the hard-won context that isn't obvious
from the code alone.

## What this is

`tum` is a Go app manager + framework for the **reMarkable 2** tablet. It
builds and installs apps over SSH as **suffixed sibling** appload entries
(`yaft-sandy`, `flower-sandy`, …) that never conflict with **vellum**
(the device's existing package manager). Longer term, `tum` is becoming a
framework for writing **new rM apps in Go**.

The device runs a custom Linux ("Codex") with a launcher (`xochitl`) that
drives an e-ink framebuffer via a Qt shim (`qtfb-shim.so`), orchestrated by
`xovi` (an `LD_PRELOAD` extension framework). `tum` deploys into xovi's
`appload/` directory so apps appear as launcher tiles.

## Critical domain knowledge (the traps)

These are the things that cost hours to discover. **Do not forget them.**

### 1. Git LFS — the segfault trap
`apps/rM2-stuff/vendor/noto/NotoSansMono-Regular.ttf` is a **Git LFS object**.
The real font is 403040 bytes; the LFS pointer is 131 bytes of text. If you
build without fetching LFS, `xxd -i` embeds the pointer *text* as "font
data", and `stbtt_InitFont` **segfaults at runtime** (apps "blink and close"
when launched from the tablet's launcher). Always:
```sh
git submodule update --init --recursive
cd apps/rM2-stuff && git lfs install --local && git lfs pull
```
The Docker build images include `git-lfs` for this reason.

### 2. cgo is mandatory for Go rM apps
The qtfb-shim intercepts `open`/`mmap`/`ioctl` by replacing **libc symbols**
via `LD_PRELOAD`. A **statically linked** Go binary makes raw `syscall.*`
instructions that **bypass libc entirely** → the shim never fires → you hit
the raw tiled `/dev/fb0` (260×23936) and everything breaks.
- **Always build Go apps with `CGO_ENABLED=1`** and the arm cross-gcc.
- The result is **dynamically linked** (needs only libc.so.6 at runtime,
  which the device has) and the shim intercepts correctly — exactly like the
  C++ `yaft` binary.
- Build image: `tum-go-armhf` (Dockerfile.build-go-armhf).

### 3. Apps must stay alive
The launcher (appload) launches `application` via QProcess and treats **process
exit as app-close**. `yaft` stays alive (terminal loop); a one-shot like
`flower` must **block after drawing** (wait for a signal/touch), or it
"crashes" immediately from the launcher's perspective.

### 4. Two yaft binaries on device
`vellum`'s `yaft` package owns `appload/yaft/`. `tum install yaft` creates a
**separate** `appload/yaft-sandy/`. They coexist as two launcher tiles. Do
**not** overwrite vellum's copy unless explicitly asked; the `-sandy` suffix
is the whole conflict-avoidance design.

### 5. SSH quirks on the device
- The device runs **BusyBox**; `head`/`cat`/`sed`/`grep` are limited (no `-n`
  for head counts in pipes, no `cat -A`, `sed -i` can corrupt multi-byte
  lines). Prefer `grep -n`, `sed -n 'Np'`, or write files via `ssh ... 'cat > f'`.
- The `vellum` binary is at `/home/root/.vellum/bin/vellum` but is only on
  `PATH` in **interactive** shells (`.bashrc`). Non-interactive SSH needs the
  full path (`tum remote` handles this).
- The device **suspends** and drops off the network. SSH won't wake it — a
  physical power-tap is required. Don't assume SSH timeouts are bugs.

### 6. The framebuffer is RGB565
`/dev/fb0` (via the shim) presents `1404×1872`, 16bpp, stride=2808. Pixels are
**RGB565** (0x0000 black, 0xFFFF white). Use `WaveformGC16`+`mode=1`(FULL) to
clear ghosting; `WaveformDU` for fast partial/animated updates. See `internal/rmfb`.

### 7. Input device paths (confusing!)
The `/dev/input/touchscreen0` symlink points to **event1 = the Wacom pen
digitizer**, NOT the touch panel. The real devices are:
- `event0` — power button (`snvs-powerkey`)
- `event1` — Wacom pen digitizer (a.k.a. `touchscreen0` symlink — misleading)
- `event2` — **touch panel** (`pt_mt`) — this is what you want for touches
- `event3` — Type Folio keyboard (`rM_Keyboard`)
The `QTFB_SHIM_INPUT_PATH_NULL=/dev/input/touchscreen0` env in manifests tells
the shim to nullify its own input handling of that path so the app can read
raw evdev. To exit on "any input", open event2 (touch) + event3 (keyboard).

## Repo layout

```
main.go                     entrypoint → cmd
cmd/                        cobra commands: build, install, list, remove,
                            remote (vellum passthrough), doctor, apps, emulate
internal/
  config/      tum.toml (device host, suffix, paths)
  device/      SSH/SCP client (shells out to ssh/scp; reuses ~/.ssh)
  recipe/      apps/*.toml recipes (embedded); {{.TumRoot}}/{{.SourceDir}}
  manifest/    external.manifest.json (tum owns it for sandy apps)
  appload/     install/list/remove (suffixed-sibling model)
  build/       runs a recipe's build_cmd
  rmfb/        Go framebuffer binding (cgo) — the framework foundation
  rminput/     Go evdev touch input binding (cgo)
apps/
  rM2-stuff/   git submodule (GPLv3, our fork with rotation/refresh fixes)
  flower/      first Go rM app: rose-curve flower + tappable close button
  gotest/      framebuffer proof-of-concept
docker/        Dockerfile.build-armhf (C++/yaft), .build-go-armhf (Go apps),
               .emulate (SDL host build for testing)
```

## How to build and test

### tum itself
```sh
go build -o tum . && ./tum doctor    # device must be awake + SSH-able
```

### yaft (C++, from the submodule)
```sh
docker build -t rm2stuff-cross-armhf -f docker/Dockerfile.build-armhf .
tum build yaft        # cross-compile via Docker
tum install yaft      # deploy as yaft-sandy
```
Host emulation (SDL window, no device needed):
```sh
tum emulate yaft     # builds + runs in a native macOS window
```

### Go apps (flower, etc.)
```sh
docker build -t tum-go-armhf -f docker/Dockerfile.build-go-armhf .
tum build flower
tum install flower
```

### On-device verification
```sh
ssh root@10.11.99.1 'bash /home/root/xovi/start'   # restart launcher to pick up new tiles
# then tap the tile on the tablet
```

## The yaft rotation bug (history + state)

The Type Folio rotation was the original task. Status:
- **Config is fixed**: `folio-rotation = "counterclockwise"` in
  `~/.config/yaft/config.toml` produces **upright, readable** landscape with
  the folio attached (confirmed). `clockwise` → upside-down.
- **A `change`-event fix was applied** in `libs/rMlib/Input.cpp` (only
  `removeDevice` on explicit `"remove"`, not on `"change"`), because the folio
  emits `change` events that spuriously cleared `pogoKeyboard` mid-session.
  **This fix is NOT yet verified end-to-end** — if yaft still reverts to
  portrait on a full ink refresh with the folio attached, the `change`-event
  theory was incomplete and further debugging of the udev/`onDeviceUpdate`
  path is needed. The diagnostic `std::cerr` lines in `checkLandscape` log
  `hasKeyboard`/`rotation` live — grep for `yaft: checkLandscape` in stderr.
- The qtfb-shim applies **no rotation of its own** (confirmed via strings).

## Framework direction (the vision)

The goal is a Go framework for easily writing rM apps: buttons, text,
"sprinkles"/animations, input handling — a single Go file → `tum install`.
`internal/rmfb` + `internal/rminput` are the first two foundation packages.
The `flower` app is the reference example. Next steps: a higher-level widget
layer (Canvas, Button, Text, EventLoop) on top of rmfb/rminput, and a
`tum` recipe that builds Go apps natively (no Docker needed for pure-Go
logic, only cgo for the fb/input bindings).

## Conventions

- Go code: gofmt + `go vet ./...` clean before committing.
- New app = a `apps/<name>/` dir + `internal/recipe/apps/<name>.toml` + (if
  using rmfb/rminput) a go.mod that `replace`s `github.com/thecsw/tum`.
- Don't commit `tum.toml` (machine-specific; `.gitignore`d). `tum.example.toml`
  is the template.
- The `apps/rM2-stuff/` submodule is GPLv3; our Go framework code is the
  license of this repo (see LICENSE).
