// gotest (cgo): draws a pretty flower on the reMarkable screen.
//
// Uses a filled rose curve r = R * |cos(kθ)| (k=4 → 8 petals) — a genuine
// flower shape, and a nice framebuffer/e-ink test. Via cgo so the qtfb-shim's
// LD_PRELOAD hooks intercept open/mmap/ioctl (a static Go binary would bypass
// them with raw syscalls).
package main

/*
#include <fcntl.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <linux/fb.h>
#include <linux/mxcfb.h>

#define WAVEFORM_GC16 2
#define UPDATE_MODE_FULL 1

static int mxcfb_send_update(int fd, struct mxcfb_update_data* u) {
    return ioctl(fd, MXCFB_SEND_UPDATE, u);
}
static int fb_get_vscreen(int fd, struct fb_var_screeninfo* v) {
    return ioctl(fd, FBIOGET_VSCREENINFO, v);
}
static int fb_get_fscreen(int fd, struct fb_fix_screeninfo* f) {
    return ioctl(fd, FBIOGET_FSCREENINFO, f);
}
static void* fb_mmap(int fd, size_t len) {
    return mmap(NULL, len, PROT_READ|PROT_WRITE, MAP_SHARED, fd, 0);
}
static int fb_open(void) { return open("/dev/fb0", O_RDWR); }
*/
import "C"

import (
	"fmt"
	"math"
	"os"
	"unsafe"
)

func main() {
	fd := C.fb_open()
	if fd < 0 {
		fmt.Fprintln(os.Stderr, "open /dev/fb0 failed")
		os.Exit(1)
	}
	defer C.close(fd)

	var v C.struct_fb_var_screeninfo
	var f C.struct_fb_fix_screeninfo
	if C.fb_get_vscreen(fd, &v) != 0 || C.fb_get_fscreen(fd, &f) != 0 {
		fmt.Fprintln(os.Stderr, "ioctl screeninfo failed")
		os.Exit(1)
	}
	w := int(v.xres)
	h := int(v.yres)
	stride := int(f.line_length)
	bpp := int(v.bits_per_pixel)
	fmt.Fprintf(os.Stderr, "flower: fb %dx%d stride=%d bpp=%d\n", w, h, stride, bpp)

	size := stride * h
	mem := C.fb_mmap(fd, C.size_t(size))
	if mem == nil {
		fmt.Fprintln(os.Stderr, "mmap failed")
		os.Exit(1)
	}
	buf := (*[1 << 30]byte)(unsafe.Pointer(mem))[:size:size]
	bppB := bpp / 8

	// White background (RGB565 0xFFFF = white).
	for i := 0; i+1 < size; i += bppB {
		buf[i] = 0xFF
		buf[i+1] = 0xFF
	}
	set := func(x, y int) {
		if x < 0 || y < 0 || x >= w || y >= h {
			return
		}
		off := y*stride + x*bppB
		if off+1 < size {
			buf[off] = 0x00 // black
			buf[off+1] = 0x00
		}
	}

	// --- A pretty flower: filled rose curve r = R*|cos(kθ)| ---
	cx, cy := w/2, h/2
	R := float64((w - 1) / 2)
	if r := float64((h - 1) / 2); r < R {
		R = r
	}
	R -= 60
	k := 4.0 // 8 petals
	// Fill the bloom outward from the center along each angle.
	for theta := 0.0; theta < 2*math.Pi; theta += 0.004 {
		r := R * math.Abs(math.Cos(k*theta))
		for rr := 0.0; rr <= r; rr += 1.0 {
			x := int(float64(cx) + rr*math.Cos(theta))
			y := int(float64(cy) + rr*math.Sin(theta))
			set(x, y)
		}
	}
	// A smaller inner bloom for depth.
	for theta := 0.0; theta < 2*math.Pi; theta += 0.004 {
		r := (R / 2) * math.Abs(math.Cos(k*theta+math.Pi/k))
		for rr := 0.0; rr <= r; rr += 1.0 {
			x := int(float64(cx) + rr*math.Cos(theta))
			y := int(float64(cy) + rr*math.Sin(theta))
			set(x, y)
		}
	}
	// A stem.
	for y := cy; y < h-20; y++ {
		set(cx, y)
		set(cx-1, y)
		set(cx+1, y)
	}

	// Full-screen GC16 full refresh → clean clear + draw (RM1 path semantics
	// via the shim: update_mode=FULL, flags=0).
	upd := C.struct_mxcfb_update_data{
		update_region: C.struct_mxcfb_rect{top: 0, left: 0, width: C.uint(w), height: C.uint(h)},
		waveform_mode: C.WAVEFORM_GC16,
		update_mode:   C.UPDATE_MODE_FULL,
	}
	if C.mxcfb_send_update(fd, &upd) != 0 {
		fmt.Fprintln(os.Stderr, "SEND_UPDATE failed")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "flower: drew an 8-petal rose + stem. Check screen!")
}
