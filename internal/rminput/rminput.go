// Package rminput reads raw evdev events from the reMarkable touch panel.
// Minimal: reports touch-down/move/up with x,y in framebuffer coordinates.
package rminput

/*
#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>
#include <linux/input.h>

static int ev_open(const char* p) { return open(p, O_RDONLY); }
*/
import "C"

import (
	"errors"
	"os"
	"unsafe"
)

// Event is a decoded touch event in framebuffer pixel coordinates.
type Event struct {
	Down bool
	X, Y int
}

// Reader reads /dev/input/eventN and decodes MT-B touch events.
type Reader struct {
	f        *os.File
	width    int // screen width, for coordinate transform
	height   int
	slotX    int
	slotY    int
	tracking bool
}

// New opens an evdev device and maps abs coords to the given screen size.
func New(path string, screenW, screenH int) (*Reader, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	fd := C.ev_open(cpath)
	if fd < 0 {
		return nil, errors.New("open " + path + " failed")
	}
	return &Reader{
		f:      os.NewFile(uintptr(fd), path),
		width:  screenW,
		height: screenH,
	}, nil
}

// Close closes the input device.
func (r *Reader) Close() error { return r.f.Close() }

// ErrNoEvent is returned for non-touch events (callers can loop).
var ErrNoEvent = errors.New("non-touch event")

// input_event on 32-bit arm: struct timeval { long sec; long usec; } + __u16
// type, code; __s32 value = 8+8 + 2+2+4 = 24 bytes.
const evSize = 24

// MT event codes.
const (
	evSyn           = 0x00
	evAbs           = 0x03
	absMTX          = 0x35
	absMTY          = 0x36
	absMtTrackingID = 0x39
	synReport       = 0x00
)

// Read decodes one SYN_REPORT's worth of events into a touch Event.
func (r *Reader) Read() (Event, error) {
	var ev [evSize]byte
	var trackingID, x, y int
	hasX, hasY := false, false

	for {
		if _, err := r.f.Read(ev[:]); err != nil {
			return Event{}, err
		}
		typ := int(ev[16]) | int(ev[17])<<8
		code := int(ev[18]) | int(ev[19])<<8
		val := int(int32(uint32(ev[20]) | uint32(ev[21])<<8 | uint32(ev[22])<<16 | uint32(ev[23])<<24))

		if typ == evSyn && code == synReport {
			if hasX {
				r.slotX = x
			}
			if hasY {
				r.slotY = y
			}
			if trackingID == -1 {
				r.tracking = false
				return Event{Down: false, X: r.slotX, Y: r.slotY}, nil
			}
			r.tracking = true
			return Event{Down: true, X: r.slotX, Y: r.slotY}, nil
		}

		if typ == evAbs {
			switch code {
			case absMTX:
				x = val
				hasX = true
			case absMTY:
				y = val
				hasY = true
			case absMtTrackingID:
				trackingID = val
			}
		}
	}
}
