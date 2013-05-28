package pixl

import (
	"io"
	// "fmt"
	// "sort"
	"image"
	"image/draw"
	"image/png"
	"image/color"
	"math"
	"math/rand"

	"x-go-binding/ui"

	_ "image/jpeg"
)

type Pixeler interface {

	Decode(r io.Reader) error

	Encode(w io.Writer) error

	// Down-sample image based on desired resolution and aggregate function
	Pixelate(nb int, f func (bl image.Point, p *Pixeler) color.Color) error

	// swap two tiles
	Swap(p1, p2 image.Point) error

	// Rearrange tiles, using f to bias the shuffle
	Shuffle(f func(p1, p2 image.Point) bool) error

	// Gets the bounding box for a specific block
	GetBlock(bl image.Point) image.Rectangle

	FillBlock(bl image.Point, c color.Color)

}

type Pixl struct {
	Image *image.RGBA
	NumCols int
	NumRows int
	BlockSize int
	Window ui.Window
}

func (p *Pixl) Decode(r io.Reader) error {
	img, _, err := image.Decode(r)
	if err == nil {
		// convert to RGBA image
		newImg := image.NewRGBA(img.Bounds())
		draw.Draw(newImg, newImg.Bounds(), img, image.ZP, draw.Src)
		p.Image = newImg
	}
	return err
}

func (p *Pixl) Encode(w io.Writer) error {
	return png.Encode(w, p.Image)
}

func (p *Pixl) Init(nb int) {
	bounds := p.Image.Bounds()
	width  := bounds.Dx()
	height := bounds.Dy()

	p.NumCols = nb
	p.BlockSize = width / nb
	p.NumRows = height / p.BlockSize
}

func (p *Pixl) Pixelate(nb int, f func (bl image.Point, p *Pixl) color.Color) error {
	p.Init(nb)
	var x, y int
	// columns
	for x=0; x < p.NumCols; x++ {
		// rows
		for y=0; y < p.NumRows; y++ {
			pt := image.Pt(x,y)
			p.FillBlock(pt, f(pt, p))
		}
	}
	p.Crop()
	return nil
}

// TODO: fixme
func (p *Pixl) Crop() {
	subImg := p.Image.SubImage(image.Rect(0,0, p.NumCols * p.BlockSize, p.NumRows * p.BlockSize))
	newImg := image.NewRGBA(subImg.Bounds())
	draw.Draw(newImg, newImg.Bounds(), subImg, image.ZP, draw.Src)
	p.Image = newImg
}

func (p *Pixl) ColorAt(bl image.Point) color.Color {
	return p.random(bl)
}

func (p *Pixl) Swap(p1, p2 image.Point) error {
	c1 := p.ColorAt(p1)
	c2 := p.ColorAt(p2)
	p.FillBlock(p2, c1)
	p.FillBlock(p1, c2)
	return nil
}

// fisher-yates-ish shuffle
func (p *Pixl) Shuffle(f func(p *Pixl, p1, p2 image.Point) bool) error {
	for i:= p.NumCols * p.NumRows - 1; i > 0; i-- {
		p1 := p.GetPoint(i)
		p2 := p.GetPoint(rand.Int() % (i + 1))
		if f(p, p1, p2) {
			p.Swap(p1, p2)
		}
	}
	return nil
}

func (p *Pixl) inBounds(pt image.Point) bool {
	return pt.X >= 0 && pt.X < p.NumCols && pt.Y >= 0 && pt.Y < p.NumRows
}

func (p *Pixl) DoStep(frequency float64, dist func (color.Color, color.Color) float64) {

	numCells := p.NumCols * p.NumRows

	iters := int(math.Floor(float64(numCells) * frequency))

	for i:=0; i < iters; i++ {
		bn := rand.Int() % numCells

		pt := p.GetPoint(bn)

		var minScore, currScore float64

		minScore = math.MaxFloat64
		minPt := image.Pt(-1,-1)  //invalid location

		// look at all locations that are one away from the current point

		for xDelta := -1; xDelta < 2; xDelta++ {
			for yDelta:= -1; yDelta < 2; yDelta++ {
				currScore = 0
				delta := image.Pt(xDelta, yDelta)
				newPt := pt.Add(delta)
				if p.inBounds(newPt) {
					// all nearest neighbors
					for xd2 := -1; xd2 < 2; xd2++ {
						for yd2 := -1; yd2 < 2; yd2++ {
							delta := image.Pt(xd2, yd2)
							neighborToMover := newPt.Add(delta)
							neighborToMoved := pt.Add(delta)
							if p.inBounds(neighborToMover) {
								currScore += dist(p.ColorAt(pt), p.ColorAt(neighborToMover))
							}
							if p.inBounds(neighborToMoved) {
								currScore += dist(p.ColorAt(newPt), p.ColorAt(neighborToMoved))
							}
						}
					}
					if currScore < minScore {
						minScore = currScore
						minPt = newPt
					}
				}
			}
		}
		if minPt.X >= 0 && minPt.Y >= 0 {
			p.Swap(pt, minPt)
		}
	}
}


func (p *Pixl) GetPoint(bn int) image.Point {
	x := (bn % p.NumCols)
	y := (bn / p.NumCols)
	return image.Pt(x, y)
}

func (p *Pixl) GetBlock(bl image.Point) image.Rectangle {
	bs := p.BlockSize
	return image.Rect(bl.X * bs, bl.Y * bs, (bl.X + 1)*bs, (bl.Y + 1)*bs)
}

func (p *Pixl) FillBlock(bl image.Point, c color.Color) {
	draw.Draw(p.Image, p.GetBlock(bl), &image.Uniform{c}, image.ZP, draw.Src)
}

// func (p *Pixl) SortRows() {
// 	for i:=0; i<p.NumRows; i++ {
// 		sp := new(SubPixl)
// 		sp.Init(p, i*p.NumCols, (i+1)*p.NumCols)
// 		sp.Sort()
// 	}
// }

func (p *Pixl) WriteToScreen() {
	draw.Draw(p.Window.Screen(), p.Window.Screen().Bounds(), p.Image, image.ZP, draw.Src)
	p.Window.FlushImage()
}

// TODO: replace with function that bins colors and selects the mode.
func (p *Pixl) random (bl image.Point) color.Color {
	subImg  := p.Image.SubImage(p.GetBlock(bl))
	bounds  := subImg.Bounds()
	offsetX := rand.Int() % p.BlockSize
	offsetY := rand.Int() % p.BlockSize
	return p.Image.At(bounds.Min.X + offsetX, bounds.Min.Y + offsetY)
}


// // represents a consecutive run of pixels
// type SubPixl struct {
// 	p *Pixl
// 	Start int
// 	End int
// }

// func (sp *SubPixl) Init(p *Pixl, s, e int) {
// 	sp.p = p
// 	sp.Start = s
// 	sp.End = e
// }

// func (sp *SubPixl) Len() int {
// 	return (sp.End + 1) - sp.Start
// }

// func (sp *SubPixl) GetXY(i int) (x, y int) {
// 	x, y = sp.p.GetXY(sp.Start + i)
// 	return x,y
// }

// func (sp *SubPixl) Color(i int) color.Color {
// 	return sp.p.ColorAt(sp.GetXY(i))
// }

// func (sp *SubPixl) Swap(i, j int) {
// 	x1, y1 := sp.GetXY(i)
// 	x2, y2 := sp.GetXY(j)
// 	sp.p.Swap(x1, y1, x2, y2)
// }

// func (sp *SubPixl) Less(i, j int) bool {
// 	r1, g1, b1, _ := sp.Color(i).RGBA()
// 	r2, g2, b2, _ := sp.Color(j).RGBA()
// 	// currently sorts on blueness
// 	return ((r1 < r2 || b1 < b2 && false) || g1 < g2) // TODO: less arbitrary criterion
// }

// func (sp *SubPixl) Sort() {
// 	sort.Sort(sp)
// }
