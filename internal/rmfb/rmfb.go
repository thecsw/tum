// Package rmfb is a tiny Go binding to the reMarkable framebuffer via the
// qtfb-shim. It opens /dev/fb0, mmaps it, and sends MXCFB updates with the
// right waveform. Used by Go rM apps that draw directly to the screen.
//
// IMPORTANT: must be built with cgo + dynamically linked so the qtfb-shim's
// LD_PRELOAD hooks (which replace libc open/mmap/ioctl) actually intercept.
// A static Go binary makes raw syscalls that bypass the shim and hits the
// raw tiled /dev/fb0 instead.
package rmfb

/*
#cgo CFLAGS: -I${SRCDIR}/../../apps/rM2-stuff/vendor
#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <linux/fb.h>
#include <linux/mxcfb.h>

static int rm_open(const char* p) { return open(p, O_RDWR); }
static int rm_ioctl(int fd, unsigned long req, void* arg) { return ioctl(fd, req, arg); }
static void* rm_mmap(int fd, size_t len) {
    return mmap(NULL, len, PROT_READ|PROT_WRITE, MAP_SHARED, fd, 0);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// Waveform modes (vendor/linux/rm2.h).
const (
	WaveformDU   = 1 // fast, low ghosting — typing/cursor
	WaveformGC16 = 2 // high quality, clears ghosting — full refresh
	WaveformGL16 = 3
	WaveformA2   = 4 // animation
)

const (
	fbPath       = "/dev/fb0"
	fbGetVScreen = 0x4600 // FBIOGET_VSCREENINFO
	fbGetFScreen = 0x4602 // FBIOGET_FSCREENINFO
)

// FB is an open, mmap'd framebuffer.
type FB struct {
	Fd     int
	Width  int
	Height int
	Stride int
	BPP    int
	Buf    []byte // mmap'd memory, length = Stride*Height
}

// Open opens and maps /dev/fb0.
func Open() (*FB, error) {
	cpath := C.CString(fbPath)
	defer C.free(unsafe.Pointer(cpath))
	fd := C.rm_open(cpath)
	if fd < 0 {
		return nil, errors.New("open /dev/fb0 failed")
	}

	var v C.struct_fb_var_screeninfo
	var f C.struct_fb_fix_screeninfo
	if C.rm_ioctl(fd, fbGetVScreen, unsafe.Pointer(&v)) != 0 ||
		C.rm_ioctl(fd, fbGetFScreen, unsafe.Pointer(&f)) != 0 {
		C.close(fd)
		return nil, errors.New("ioctl screeninfo failed")
	}

	w := int(v.xres)
	h := int(v.yres)
	stride := int(f.line_length)
	bpp := int(v.bits_per_pixel)
	size := stride * h

	mem := C.rm_mmap(fd, C.size_t(size))
	if mem == nil {
		C.close(fd)
		return nil, errors.New("mmap failed")
	}
	buf := (*[1 << 30]byte)(unsafe.Pointer(mem))[:size:size]

	return &FB{Fd: int(fd), Width: w, Height: h, Stride: stride, BPP: bpp, Buf: buf}, nil
}

// Close unmaps and closes the framebuffer.
func (fb *FB) Close() error {
	C.munmap(unsafe.Pointer(&fb.Buf[0]), C.size_t(len(fb.Buf)))
	C.close(C.int(fb.Fd))
	return nil
}

// SetPixel sets an RGB565 pixel at (x,y). Black=0x0000, white=0xFFFF.
func (fb *FB) SetPixel(x, y int, val uint16) {
	if x < 0 || y < 0 || x >= fb.Width || y >= fb.Height {
		return
	}
	off := y*fb.Stride + x*(fb.BPP/8)
	if off+1 >= len(fb.Buf) {
		return
	}
	fb.Buf[off] = byte(val)
	fb.Buf[off+1] = byte(val >> 8)
}

// Fill fills the whole framebuffer with an RGB565 value.
func (fb *FB) Fill(val uint16) {
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			fb.SetPixel(x, y, val)
		}
	}
}

// Update sends an MXCFB update for a region. mode: 0=PARTIAL,1=FULL.
func (fb *FB) Update(top, left, width, height int, waveform, mode int) error {
	upd := C.struct_mxcfb_update_data{
		update_region: C.struct_mxcfb_rect{
			top:    C.uint(top),
			left:   C.uint(left),
			width:  C.uint(width),
			height: C.uint(height),
		},
		waveform_mode: C.uint(waveform),
		update_mode:   C.uint(mode),
	}
	// MXCFB_SEND_UPDATE = _IOW('F',0x2E,...)
	if C.rm_ioctl(C.int(fb.Fd), 0x4048462E, unsafe.Pointer(&upd)) != 0 {
		return errors.New("SEND_UPDATE failed")
	}
	return nil
}

// FullUpdate sends a full-screen GC16 full refresh (clears ghosting).
func (fb *FB) FullUpdate() error {
	return fb.Update(0, 0, fb.Width, fb.Height, WaveformGC16, 1)
}

// String for debugging.
func (fb *FB) String() string {
	return fmt.Sprintf("fb %dx%d stride=%d bpp=%d", fb.Width, fb.Height, fb.Stride, fb.BPP)
}
