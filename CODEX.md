# Codex prompt: fix the yaft Type Folio rotation bug

## Context

Read `AGENTS.md` first — it has the full domain knowledge (LFS traps, cgo
requirements, SSH quirks, input device paths, etc.).

## The problem

yaft (a framebuffer terminal for the reMarkable 2) should render in
**landscape** when the Type Folio keyboard is attached. Instead it renders in
**portrait**, even though diagnostics confirm `rotation=3` (CounterClockwise)
is set correctly throughout the widget tree.

The reMarkable 2 is at `root@10.11.99.1` (SSH key auth). The device runs
"Codex Linux" with a launcher (xochitl) that drives e-ink via a Qt shim
(`qtfb-shim.so`, LD_PRELOAD'd). Apps are launched from `xovi`'s `appload/`
directory.

## What's already been done

1. **Config**: `~/.config/yaft/config.toml` on the device has
   `rotation = "counterclockwise"` (the correct direction — confirmed in an
   early test that rendered upright landscape).
2. **Source defaults** (`apps/rM2-stuff/apps/yaft/config.h`): changed to
   `autoRotate=false`, `rotation=CounterClockwise`.
3. **`checkLandscape`** (`apps/rM2-stuff/apps/yaft/YaftWidget.cpp`): always
   hides the on-screen keyboard when the folio is detected, regardless of
   autoRotate.
4. **udev events** (`apps/rM2-stuff/libs/rMlib/Input.cpp`): ignore all non-add
   events to stop the mid-session revert (the OS re-enumerates input devices
   on ink-refresh, generating spurious remove events).
5. **Diagnostics**: `std::cerr` lines in `checkLandscape` and `build()` show
   `rotation=3` and `hideKeyboard=1` consistently.

## The mystery

`build()` creates `Rotated(rotation=3, Column(Screen, Keyboard))`. The
`Rotated` widget should rotate the whole tree 90° CCW. Diagnostics confirm
`rotation=3` is passed. But the screen renders portrait.

One early test with forced CCW DID render landscape correctly. Subsequent
rebuilds with the same settings render portrait. Something is inconsistent.

## Your task

1. **Read `AGENTS.md`** — especially sections 1-7 and "The yaft rotation bug".
2. **Build and deploy**:
   ```sh
   cd ~/gits/tum
   git submodule update --init --recursive
   cd apps/rM2-stuff && git lfs install --local && git lfs pull && cd ../..
   docker build -t rm2stuff-cross-armhf -f docker/Dockerfile.build-armhf .
   go build -o tum .
   ./tum build yaft
   ./tum install yaft
   ssh root@10.11.99.1 'bash /home/root/xovi/start'
   ```
3. **Investigate** the rotation pipeline. Start with:
   - `apps/rM2-stuff/libs/rMlib/include/UI/Rotate.h` — does `doDraw`/`doLayout`
     actually rotate the canvas and constraints correctly?
   - `apps/rM2-stuff/libs/rMlib/include/MathUtil.h` — the `rotate()`/`invert()`
     math for Size, Point, Rect, Constraints.
   - `apps/rM2-stuff/apps/yaft/screen.cpp` — `Screen::doLayout` always gets
     `isLandscape=false`; the `Rotated` wrapper is supposed to handle rotation.
     Is the terminal being sized in portrait dimensions?
   - Compare with the **vellum original binary** on device:
     `/home/root/xovi/exthome/appload/yaft/yaft.orig` — does it render
     landscape with the folio? If so, what's different?
4. **Test on device** — the folio must be attached. Capture diagnostics:
   ```sh
   ssh root@10.11.99.1 'cd /home/root/xovi/exthome/appload/yaft-sandy && \
     LD_PRELOAD=/home/root/shims/qtfb-shim.so QTFB_SHIM_MODEL=RM1 \
     QTFB_SHIM_INPUT_PATH_NULL=/dev/input/touchscreen0 \
     QTFB_SHIM_INITIAL_DISPLAY_MODE=ANIMATE HOME=/home/root \
     ./yaft-sandy /bin/sh 2>&1' | grep -E 'build #|checkLandscape|init fb'
   ```
5. **Commit** fixes to the `apps/rM2-stuff` submodule (push to
   `github.com/thecsw/rM2-stuff`, branch `master`) and bump the submodule pin
   in tum (push to `github.com/thecsw/tum`, branch `main`).

## Key constraints

- **Never do full-screen GC16 updates in a tight loop** — it freezes the EPDC.
- The device **suspends** — tap the power button to wake it before SSH.
- BusyBox `head`/`cat`/`sed` are limited — use `grep -n`, `sed -n 'Np'`.
- `vellum` owns `appload/yaft/`; tum owns `appload/yaft-sandy/`. Don't
  overwrite vellum's unless testing requires it.
- Ctrl+Z exits yaft (already implemented and working).
