// flower: draws an animated 8-petal rose-curve flower on the reMarkable screen,
// growing petal-by-petal. Exit on any touch or any keyboard key press.
//
// The reference Go rM app on rmfb + rminput.
package main

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/thecsw/tum/internal/rmfb"
	"github.com/thecsw/tum/internal/rminput"
)

const (
	white = 0xFFFF
	black = 0x0000
)

func main() {
	fb, err := rmfb.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "flower:", err)
		os.Exit(1)
	}
	defer fb.Close()
	fmt.Fprintln(os.Stderr, "flower:", fb)

	// Clear to white (blank page) with a full refresh.
	fb.Fill(white)
	if err := fb.FullUpdate(); err != nil {
		fmt.Fprintln(os.Stderr, "flower:", err)
		os.Exit(1)
	}

	// Watch touch + keyboard in the background; either signals exit.
	stop := make(chan struct{})
	var once sync.Once
	exitOn := func(path, name string) {
		r, err := rminput.New(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "flower: %s unavailable: %v\n", name, err)
			return
		}
		defer r.Close()
		for {
			ev, err := r.Read()
			if err != nil {
				return
			}
			// Any touch contact or any key press → exit.
			if ev.IsTouch() || ev.IsKeyPress() {
				fmt.Fprintf(os.Stderr, "flower: %s activity, exiting\n", name)
				once.Do(func() { close(stop) })
				return
			}
		}
	}
	// Touch is /dev/input/event2 (pt_mt). Keyboard/folio is event3 (rM_Keyboard).
	go exitOn("/dev/input/event2", "touch")
	go exitOn("/dev/input/event3", "keyboard")

	// Animate the bloom: grow the flower petal by petal with DU updates.
	animateBloom(fb, stop)
	fmt.Fprintln(os.Stderr, "flower: bloomed 🌸 (touch or press a key to exit)")

	// Wait for exit signal.
	<-stop
	// Final clean full refresh.
	fb.FullUpdate()
}

// animateBloom grows the flower from center outward, drawing petals one at a
// time. Each step does a DU (fast) update so you can watch it grow.
func animateBloom(fb *rmfb.FB, stop <-chan struct{}) {
	set := func(x, y int) { fb.SetPixel(x, y, black) }

	cx, cy := fb.Width/2, fb.Height/2-100
	R := float64(fb.Width/2) - 80
	if r := float64(fb.Height/2 - 120); r < R {
		R = r
	}
	k := 4.0 // 8 petals

	// Center disk first.
	drawDisk(fb, cx, cy, 18, set)
	fb.Update(cx-20, cy-20, 40, 40, rmfb.WaveformDU, 0)

	// Grow each petal: walk θ from 0 to 2π, drawing the radius out to the
	// current max. Step in small angular increments so it blooms smoothly.
	steps := 60
	for step := 1; step <= steps; step++ {
		select {
		case <-stop:
			return
		default:
		}
		frac := float64(step) / float64(steps)
		maxR := R * frac
		// Draw the bloom up to maxR for all angles.
		for theta := 0.0; theta < 2*math.Pi; theta += 0.02 {
			r := R * math.Abs(math.Cos(k*theta))
			if r > maxR {
				// only draw the outer ring at the current radius (growing edge)
				r = maxR
			}
			// fill from center to current edge
			for rr := 0.0; rr <= r; rr += 2.0 {
				x := int(float64(cx) + rr*math.Cos(theta))
				y := int(float64(cy) + rr*math.Sin(theta))
				set(x, y)
			}
		}
		// DU update the bloom region (fast, no full flash).
		fb.Update(0, 0, fb.Width, fb.Height, rmfb.WaveformDU, 0)
		time.Sleep(40 * time.Millisecond)
	}

	// Inner bloom for depth.
	for theta := 0.0; theta < 2*math.Pi; theta += 0.003 {
		r := (R / 2) * math.Abs(math.Cos(k*theta+math.Pi/k))
		for rr := 0.0; rr <= r; rr += 1.0 {
			x := int(float64(cx) + rr*math.Cos(theta))
			y := int(float64(cy) + rr*math.Sin(theta))
			set(x, y)
		}
	}

	// Stem + leaves.
	for y := cy; y < fb.Height-30; y++ {
		set(cx, y)
		set(cx-1, y)
		set(cx+1, y)
	}
	leafY := cy + (fb.Height-cy)/3
	drawLeaf(fb, cx, leafY, 80, 1, set)
	drawLeaf(fb, cx, leafY+40, 70, -1, set)

	// Final full refresh to clean up ghosting and show the whole flower.
	fb.FullUpdate()
}

func drawDisk(fb *rmfb.FB, cx, cy, r int, set func(int, int)) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				set(cx+x, cy+y)
			}
		}
	}
}

func drawLeaf(fb *rmfb.FB, cx, cy, size, dir int, set func(int, int)) {
	for t := -math.Pi / 2; t <= math.Pi/2; t += 0.02 {
		r := float64(size) * math.Cos(t)
		x := int(r * float64(dir))
		y := int(float64(size) / 2 * math.Sin(t))
		set(cx+x, cy+y)
		set(cx+x-1*dir, cy+y)
	}
}
