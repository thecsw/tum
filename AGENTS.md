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

## The yaft rotation bug (UNRESOLVED — needs investigation)

**Symptom**: With the Type Folio attached, yaft renders in **portrait** even
though all diagnostics say `rotation=3` (CounterClockwise) and
`hideKeyboard=1`. The screen should be landscape but isn't.

**What's confirmed working**:
- `folio-rotation = "counterclockwise"` is the correct direction — when it
  DID render landscape (once), the text was upright and readable.
  `clockwise` → upside-down.
- The folio IS detected: `hasKeyboard=1`, `pogoKeyboard != nullptr`.
- `checkLandscape` sets `rotation=3` (CCW) and `hideKeyboard=1`.
- `build()` is called with `rotation=3` — the `Rotated` widget receives CCW.
- The qtfb-shim applies **no rotation of its own** (confirmed via `strings`).
- Ctrl+Z exits the app correctly (scancode 0x5a/0x7a, not 0x1a).
- A **forced** `auto-rotate=false` + `rotation=counterclockwise` config
  rendered landscape correctly in ONE early test, but subsequent rebuilds
  with the same settings render portrait. This is deeply confusing.

**What's been tried (and didn't fully fix it)**:
1. `libs/rMlib/Input.cpp`: ignore all non-add udev events (the OS re-enumerates
   input devices on ink-refresh/wake, generating spurious remove/change events
   that clear `pogoKeyboard`). This stops the *mid-session revert* but the
   *initial* portrait rendering persists.
2. `apps/yaft/YaftWidget.cpp`: `checkLandscape` always hides the on-screen
   keyboard when the folio is attached (was only hiding it in the autoRotate
   branch).
3. `apps/yaft/config.h`: defaults changed to `autoRotate=false` +
   `rotation=CounterClockwise`.
4. Added `std::cerr` diagnostics in `checkLandscape` and `build()` — all show
   `rotation=3` consistently.

**The core mystery (SOLVED)**: The qtfb-shim/xochitl display compositing
applies its own rotation. The working combination for upright landscape is:
- `Screen::isLandscape=true` in `doLayout` (sizing) — sizes the terminal for
  the framebuffer dimensions (no w/h swap).
- `isLandscape=false` in `drawLine` (pixel drawing) — draws in portrait
  coordinates; the display's compositing handles the rotation to landscape.
- No `Rotated` widget wrapper.

The matrix of attempts:
| doLayout sizing | drawLine coords | Result |
|---|---|---|
| swap (1872 wide) | portrait (false) | upright landscape, ~25% margin |
| no swap (1404 wide) | portrait (false) | upright landscape, ~25% margin (CURRENT) |
| swap (1872 wide) | swapped (true) | 180° inverted portrait |
| Rotated widget | — | thin column, 90% blank |

**Known limitation**: ~25% blank margin on one side. The terminal fills the
framebuffer width (1404) but the display shows it at 1872 wide (landscape),
so there's unfilled space. Can't be eliminated with portrait drawing (the
terminal can't exceed the framebuffer width without swapped pixel coords,
which invert). A future fix might write pixels directly in landscape
coordinates with a custom transform that doesn't invert.

**Config error rendering (SOLVED)**: the `0??clockwise` garble was yaft's
`logTerm()` writing config error messages into the pty as visible terminal
text. Fixed: config errors now go to stderr only, so the terminal starts
clean with just the shell prompt. The config itself was always valid.

**Key files**:
- `apps/rM2-stuff/apps/yaft/YaftWidget.h` — `build()` creates `Rotated(rotation, ...)`
- `apps/rM2-stuff/apps/yaft/YaftWidget.cpp` — `checkLandscape()` sets rotation
- `apps/rM2-stuff/apps/yaft/screen.cpp` — `Screen::doLayout`/`doDraw` (always
  `isLandscape=false` now; rotation handled by the `Rotated` wrapper)
- `apps/rM2-stuff/libs/rMlib/include/UI/Rotate.h` — the rotation widget
- `apps/rM2-stuff/libs/rMlib/include/MathUtil.h` — `rotate()`/`invert()` math

**How to debug**:
```sh
# Run yaft with diagnostics, capture stderr:
ssh root@10.11.99.1 'cd /home/root/xovi/exthome/appload/yaft-sandy && \
  LD_PRELOAD=/home/root/shims/qtfb-shim.so QTFB_SHIM_MODEL=RM1 \
  QTFB_SHIM_INPUT_PATH_NULL=/dev/input/touchscreen0 \
  QTFB_SHIM_INITIAL_DISPLAY_MODE=ANIMATE HOME=/home/root \
  ./yaft-sandy /bin/sh 2>&1' | grep -E 'build #|checkLandscape|init fb'
# Then type in yaft to trigger rebuilds and watch if rotation changes.
```

## Framework direction (the vision)

The goal is a Go framework for easily writing rM apps: buttons, text,
"sprinkles"/animations, input handling — a single Go file → `tum install`.
`internal/rmfb` + `internal/rminput` are the first two foundation packages.
The `flower` app is the reference example. Next steps: a higher-level widget
layer (Canvas, Button, Text, EventLoop) on top of rmfb/rminput, and a
`tum` recipe that builds Go apps natively (no Docker needed for pure-Go
logic, only cgo for the fb/input bindings).

## Cross-compilation reference

The reMarkable 2 is an **ARMv7-A** (armv7l, Cortex-A7, hard-float, NEON) device
running **glibc 2.39** on kernel 5.4.70. There are **no on-device compilers**
(no gcc, no make, no ld, no headers, no crt objects) — everything must be
cross-compiled or shipped as a pre-built binary.

Run `tum target` for the live specs. Key flags:

| What         | Flags                                                              |
|--------------|--------------------------------------------------------------------|
| Go (pure)    | `GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build`            |
| Go (cgo)     | `GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 CC=arm-linux-gnueabihf-gcc` |
| C/C++ gcc    | `arm-linux-gnueabihf-gcc -march=armv7-a -mfpu=neon -mfloat-abi=hard` |
| Linker interp| `/lib/ld-linux-armhf.so.3`                                          |
| Runtime deps | `libc.so.6` (glibc 2.39), `libstdc++.so.6`, `libudev.so.1`          |

### Docker images

- `rm2stuff-cross-armhf` — C/C++ apps (yaft). Has `arm-linux-gnueabihf-gcc`,
  git-lfs, and the full cross-toolchain with crt objects + glibc headers.
- `tum-go-armhf` — Go cgo apps (flower). Has Go + the cross-compiler.
- `tum-emulate` — host SDL emulation for testing without a device.

### On-device C compilation (tcc)

`tcc` (TinyCC) is installed on the device at `/usr/local/bin/tcc` with
crt objects, libtcc1.a, and glibc headers. You can compile and run C
programs directly on the reMarkable:

```sh
tcc hello.c -o hello && ./hello
tcc -run hello.c          # compile and run in one step
```

**Critical build detail**: tcc must be built with `--triplet=arm-linux-gnueabihf`
so that `TCC_ARM_EABI` is defined. Without it, the ARM stack-alignment fixup
in `gfunc_epilog` is skipped, and calls to glibc functions that use LDRD/STRD
(puts, printf, fputs, fputc — anything touching stdio streams) crash with
SIGBUS (exit code 135). Direct function calls and syscall wrappers (write,
strlen, getpid, rand) work even without the fixup, which makes it hard to spot.

The device has no `libc.so` symlink (only `libc.so.6`) and no
`ld-linux.so.3` (only `ld-linux-armhf.so.3`). The tcc install creates these
symlinks. Use `-L/lib` if tcc can't find libc.

## Conventions

- Go code: gofmt + `go vet ./...` clean before committing.
- New app = a `apps/<name>/` dir + `internal/recipe/apps/<name>.toml` + (if
  using rmfb/rminput) a go.mod that `replace`s `github.com/thecsw/tum`.
- Don't commit `tum.toml` (machine-specific; `.gitignore`d). `tum.example.toml`
  is the template.
- The `apps/rM2-stuff/` submodule is GPLv3; our Go framework code is the
  license of this repo (see LICENSE).
