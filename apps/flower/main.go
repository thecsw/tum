// flower: draws an 8-petal rose-curve flower on the reMarkable, blooming one
// petal at a time clockwise. Exit on any touch or any keyboard key press.
//
// IMPORTANT: never do full-screen updates in a tight loop — that freezes the
// EPDC. Draw one petal, update only that petal's small region, then pause.
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

// 8 petals: r = R*|cos(4θ)|. Petals are centered at θ = i*π/4 for i=0..7.
const k = 4.0

func main() {
	fb, err := rmfb.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "flower:", err)
		os.Exit(1)
	}
	defer fb.Close()
	fmt.Fprintln(os.Stderr, "flower:", fb)

	// Clear to white with a full refresh.
	fb.Fill(white)
	fb.FullUpdate()
	time.Sleep(200 * time.Millisecond)

	// Watch touch + keyboard; either signals exit.
	stop := make(chan struct{})
	var once sync.Once
	exitOn := func(path, name string) {
		r, err := rminput.New(path)
		if err != nil {
			return
		}
		defer r.Close()
		for {
			ev, err := r.Read()
			if err != nil {
				return
			}
			if ev.IsTouch() || ev.IsKeyPress() {
				once.Do(func() { close(stop) })
				return
			}
		}
	}
	go exitOn("/dev/input/event2", "touch")
	go exitOn("/dev/input/event3", "keyboard")

	// Flower geometry.
	cx := fb.Width / 2
	cy := fb.Height/2 - 100
	R := float64(fb.Width/2) - 80
	if r := float64(fb.Height/2 - 120); r < R {
		R = r
	}

	set := func(x, y int) { fb.SetPixel(x, y, black) }

	// Center disk.
	drawDisk(cx, cy, 16, set)
	fb.Update(cx-20, cy-20, 40, 40, rmfb.WaveformDU, 0)
	time.Sleep(300 * time.Millisecond)

	// Bloom one petal at a time, clockwise.
	// Each petal spans π/4 in θ. Petal i covers θ ∈ [i*π/4 - π/8, i*π/4 + π/8].
	numPetals := 8
	for i := 0; i < numPetals; i++ {
		select {
		case <-stop:
			return
		default:
		}
		// Compute the petal's bounding box as we draw it.
		minX, minY := cx, cy
		maxX, maxY := cx, cy
		updateBB := func(x, y int) {
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}

		thetaStart := float64(i)*math.Pi/4 - math.Pi/8
		thetaEnd := float64(i)*math.Pi/4 + math.Pi/8
		for theta := thetaStart; theta <= thetaEnd; theta += 0.003 {
			r := R * math.Abs(math.Cos(k*theta))
			for rr := 0.0; rr <= r; rr += 1.0 {
				x := int(float64(cx) + rr*math.Cos(theta))
				y := int(float64(cy) + rr*math.Sin(theta))
				set(x, y)
				updateBB(x, y)
			}
		}
		// Update ONLY this petal's region (small, fast, won't freeze EPDC).
		pad := 10
		fb.Update(minY-pad, minX-pad, (maxX-minX)+pad*2, (maxY-minY)+pad*2, rmfb.WaveformDU, 0)
		time.Sleep(350 * time.Millisecond)
	}

	// Stem + leaves (small regional DU update — no full-screen GC16 to avoid
	// the EPDC freeze that GC16-full causes).
	stemMinY := cy
	stemMaxY := fb.Height - 30
	for y := cy; y < fb.Height-30; y++ {
		set(cx, y)
		set(cx-1, y)
		set(cx+1, y)
	}
	leafY := cy + (fb.Height-cy)/3
	drawLeaf(cx, leafY, 80, 1, set)
	drawLeaf(cx, leafY+40, 70, -1, set)
	// One small regional update for the stem+leaves.
	fb.Update(stemMinY, cx-90, 180, stemMaxY-stemMinY, rmfb.WaveformDU, 0)
	fmt.Fprintln(os.Stderr, "flower: bloomed 🌸 (touch or press a key to exit)")

	<-stop
	// Exit: small regional DU update to clear, no full-screen GC16.
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

func drawLeaf(cx, cy, size, dir int, set func(int, int)) {
	for t := -math.Pi / 2; t <= math.Pi/2; t += 0.02 {
		r := float64(size) * math.Cos(t)
		x := int(r * float64(dir))
		y := int(float64(size) / 2 * math.Sin(t))
		set(cx+x, cy+y)
		set(cx+x-1*dir, cy+y)
	}
}
