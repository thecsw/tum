// flower: draws a pretty 8-petal rose-curve flower on the reMarkable screen,
// with a tappable close button. The second Go rM app on rmfb + rminput.
package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/thecsw/tum/internal/rmfb"
	"github.com/thecsw/tum/internal/rminput"
)

const (
	white = 0xFFFF
	black = 0x0000
)

// Close button in the top-right corner.
const (
	btnSize = 120
	btnX    = 0 // top-left x
	btnY    = 0 // top-left y
)

func main() {
	fb, err := rmfb.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "flower:", err)
		os.Exit(1)
	}
	defer fb.Close()
	fmt.Fprintln(os.Stderr, "flower:", fb)

	// Clear to white (blank page).
	fb.Fill(white)
	drawFlower(fb)
	drawCloseButton(fb)

	if err := fb.FullUpdate(); err != nil {
		fmt.Fprintln(os.Stderr, "flower:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "flower: bloomed 🌸 (tap the ✕ button to close)")

	// Open touch input.
	// The qtfb-shim maps touch to /dev/input/touchscreen0 (symlink to the
	// touch panel). We read in raw evdev coordinates and transform.
	touch, err := rminput.New("/dev/input/touchscreen0", fb.Width, fb.Height)
	if err != nil {
		fmt.Fprintln(os.Stderr, "flower: touch input unavailable:", err)
		// Fall back to waiting for a signal.
		waitSignal()
		return
	}
	defer touch.Close()

	// Event loop: exit when the close button is tapped.
	for {
		ev, err := touch.Read()
		if err != nil {
			fmt.Fprintln(os.Stderr, "flower: read touch:", err)
			waitSignal()
			return
		}
		// On touch-up inside the button area, exit.
		if !ev.Down {
			if ev.X >= btnX && ev.X < btnX+btnSize &&
				ev.Y >= btnY && ev.Y < btnY+btnSize {
				fmt.Fprintln(os.Stderr, "flower: close button tapped, exiting")
				return
			}
		}
	}
}

func drawFlower(fb *rmfb.FB) {
	set := func(x, y int) { fb.SetPixel(x, y, black) }

	cx, cy := fb.Width/2, fb.Height/2-100
	R := float64(fb.Width/2) - 80
	if r := float64(fb.Height/2 - 120); r < R {
		R = r
	}

	// Outer bloom: rose curve r = R*|cos(4θ)| → 8 petals.
	drawRose(cx, cy, R, 4.0, 0.0, set)
	// Inner bloom: offset phase for depth.
	drawRose(cx, cy, R/2, 4.0, math.Pi/4, set)
	// Center disk.
	drawDisk(cx, cy, int(R/8), set)
	// Stem.
	for y := cy; y < fb.Height-30; y++ {
		set(cx, y)
		set(cx-1, y)
		set(cx+1, y)
	}
	// Two leaves on the stem.
	leafY := cy + (fb.Height-cy)/3
	drawLeaf(cx, leafY, 80, 1, set)
	drawLeaf(cx, leafY+40, 70, -1, set)
}

// drawCloseButton draws a black-outlined box with an ✕ in the top-left corner.
func drawCloseButton(fb *rmfb.FB) {
	set := func(x, y int) { fb.SetPixel(x, y, black) }
	// Box outline.
	for i := 0; i < btnSize; i++ {
		set(btnX+i, btnY)           // top
		set(btnX+i, btnY+btnSize-1) // bottom
		set(btnX, btnY+i)           // left
		set(btnX+btnSize-1, btnY+i) // right
	}
	// The ✕ (diagonal lines inset 20px).
	inset := 25
	for i := inset; i < btnSize-inset; i++ {
		set(btnX+i, btnY+i)
		set(btnX+i, btnY+btnSize-1-i)
	}
}

func drawRose(cx, cy int, R, k, phase float64, set func(int, int)) {
	for theta := 0.0; theta < 2*math.Pi; theta += 0.003 {
		r := R * math.Abs(math.Cos(k*theta+phase))
		for rr := 0.0; rr <= r; rr += 1.0 {
			x := int(float64(cx) + rr*math.Cos(theta))
			y := int(float64(cy) + rr*math.Sin(theta))
			set(x, y)
		}
	}
}

func drawDisk(cx, cy, r int, set func(int, int)) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				set(cx+x, cy+y)
			}
		}
	}
}

func drawLeaf(cx, cy, size int, dir int, set func(int, int)) {
	for t := -math.Pi / 2; t <= math.Pi/2; t += 0.02 {
		r := float64(size) * math.Cos(t)
		x := int(r * float64(dir))
		y := int(float64(size) / 2 * math.Sin(t))
		set(cx+x, cy+y)
		set(cx+x-1*dir, cy+y)
	}
}

func waitSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
}
