//go:build !linux

// Package rminput is a stub for non-Linux platforms (macOS host development).
// The real implementation is in rminput_linux.go and requires Linux headers
// (linux/input.h) only available when cross-compiling for arm.
package rminput

import "errors"

type Event struct {
	Type  int
	Code  int
	Value int
}
type Reader struct{}

func New(path string) (*Reader, error) { return nil, errors.New("rminput: only available on Linux") }
func (r *Reader) Close() error         { return nil }
func (r *Reader) Read() (Event, error) {
	return Event{}, errors.New("rminput: only available on Linux")
}
func (e Event) IsTouch() bool    { return false }
func (e Event) IsKey() bool      { return false }
func (e Event) IsKeyPress() bool { return false }
