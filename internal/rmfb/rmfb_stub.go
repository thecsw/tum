//go:build !linux

// Package rmfb is a stub for non-Linux platforms (macOS host development).
// The real implementation is in rmfb_linux.go and requires Linux headers
// (linux/fb.h, linux/mxcfb.h) only available when cross-compiling for arm.
package rmfb

import "errors"

type FB struct{}

func Open() (*FB, error)                          { return nil, errors.New("rmfb: only available on Linux") }
func (fb *FB) Close() error                       { return nil }
func (fb *FB) SetPixel(x, y int, v uint16)        {}
func (fb *FB) Fill(v uint16)                      {}
func (fb *FB) Update(t, l, w, h, wf, m int) error { return nil }
func (fb *FB) FullUpdate() error                  { return nil }

const (
	WaveformDU   = 1
	WaveformGC16 = 2
	WaveformGL16 = 3
	WaveformA2   = 4
)
