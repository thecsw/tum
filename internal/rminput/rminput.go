// Package rminput reads raw evdev events from the reMarkable touch panel and
// keyboard. Minimal: reports touch-down/up and key events.
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

// Event is a decoded input event.
type Event struct {
	Type  int // EV_KEY=1, EV_ABS=3, EV_SYN=0, ...
	Code  int
	Value int
}

// Reader reads /dev/input/eventN.
type Reader struct {
	f *os.File
}

// New opens an evdev device.
func New(path string) (*Reader, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	fd := C.ev_open(cpath)
	if fd < 0 {
		return nil, errors.New("open " + path + " failed")
	}
	return &Reader{f: os.NewFile(uintptr(fd), path)}, nil
}

// Close closes the input device.
func (r *Reader) Close() error { return r.f.Close() }

// input_event on 32-bit arm: struct timeval { long, long } + __u16 type, code;
// __s32 value = 24 bytes.
const evSize = 24

// Event codes.
const (
	EvSyn = 0x00
	EvKey = 0x01
	EvAbs = 0x03

	// MT touch.
	absMTX          = 0x35
	absMTY          = 0x36
	absMtTrackingID = 0x39
	synReport       = 0x00

	// Key states.
	KeyPress   = 1
	KeyRelease = 0
)

// Read reads one raw input_event.
func (r *Reader) Read() (Event, error) {
	var ev [evSize]byte
	if _, err := r.f.Read(ev[:]); err != nil {
		return Event{}, err
	}
	typ := int(ev[16]) | int(ev[17])<<8
	code := int(ev[18]) | int(ev[19])<<8
	val := int(int32(uint32(ev[20]) | uint32(ev[21])<<8 | uint32(ev[22])<<16 | uint32(ev[23])<<24))
	return Event{Type: typ, Code: code, Value: val}, nil
}

// AnyTouch returns true if the event is a touch contact (MT tracking id >= 0).
func (e Event) IsTouch() bool {
	return e.Type == EvAbs && (e.Code == absMTX || e.Code == absMTY || e.Code == absMtTrackingID)
}

// IsKey returns true if the event is a key/button event.
func (e Event) IsKey() bool { return e.Type == EvKey }

// IsKeyPress returns true if the event is a key press (down).
func (e Event) IsKeyPress() bool { return e.Type == EvKey && e.Value == KeyPress }
