package main

import (
	"fmt"
	"pixl"
	"time"
	"math"
	"math/rand"
	"flag"
	"os"
	"bufio"
	"image/color"

	"x-go-binding/ui"
	"x-go-binding/ui/x11"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var input = flag.String("i", "", "input file")
var output = flag.String("o", "out.png", "output file")
var shuffle   = flag.Bool("s", false, "shuffle the pixels?")
var sortrows  = flag.Bool("sr", false, "sort the rows?")
var blocksize = flag.Int("b", 10, "blocksize")
var iters = flag.Int("iters", 0, "number of iterations of clustering algorithm to perform")

func random (x, y int, p *pixl.Pixl) color.Color {
	subImg := p.Image.SubImage(p.GetBlock(x, y))
	bounds := subImg.Bounds()
	offsetX := rand.Int() % p.BlockSize
	offsetY := rand.Int() % p.BlockSize
	return p.Image.At(bounds.Min.X + offsetX, bounds.Min.Y + offsetY)
}

func unbiased (p *pixl.Pixl, x1, y1, x2, y2 int) bool {
	return true
}

func euclid(c1 color.Color, c2 color.Color) float64 {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()

	// convert to YUI
	y1, cb1, cr1 := color.RGBToYCbCr(uint8(r1), uint8(g1), uint8(b1))
	y2, cb2, cr2 := color.RGBToYCbCr(uint8(r2), uint8(g2), uint8(b2))

	score := math.Sqrt(math.Pow(float64(y1-y2),2) + math.Pow(float64(cb1-cb2),2) + math.Pow(float64(cr1-cr2),2))
	return score
}

func main () {

	rand.Seed(time.Now().UnixNano())

	flag.Parse()

	if *input == "" {
		fmt.Println("No input image!")
	}

	pix := new(pixl.Pixl)

	inf, err := os.Open(*input)
	defer inf.Close()
	if err == nil {
		reader := bufio.NewReader(inf)
		if pix.Decode(reader) != nil {
			fmt.Println("file open failed")
			os.Exit(1)
		}
	}

	// resize the window to fit the picture
	bounds := pix.Image.Bounds()

	w, err := x11.NewWindow(bounds.Dx(), bounds.Dy())
	if err != nil {
			fmt.Println(err)
			return
	}
	pix.Window = w

	pix.Pixelate(*blocksize, random)

	if *shuffle {
		pix.Shuffle(unbiased)
	}

	// run the clustering algo iters times
	if *iters != 0 {
		for i:=0; i < *iters; i++ {
			pix.DoStep(5, euclid)
			fmt.Println(i)
		}
	}

	pix.WriteToScreen()

	for e := range w.EventChan() {
		switch e := e.(type) {
		case ui.KeyEvent:
			if e.Key == ' ' { // perform another iteration
				pix.DoStep(5, euclid)
				pix.WriteToScreen()
			} else if e.Key == 's' { // save image
				outf, err := os.Create(*output)
				defer outf.Close()
				if err == nil {
					writer := bufio.NewWriter(outf)
					pix.Encode(writer)
				} else {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		}
	}

}

