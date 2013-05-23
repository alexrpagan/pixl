package pixl

import (
	"io"
	// "fmt"
	"sort"
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
	Pixelate(nb int, f func (x, y int, p *Pixeler) color.Color) error

	// swap two tiles
	Swap(x1, y1, x2, y2 int) error

	// Rearrange tiles, using f to bias the shuffle
	Shuffle(f func(x1, y1, x2, y2 int) bool) error

	// Gets the bounding box for a specific block
	GetBlock(x, y int) image.Rectangle

	FillBlock(x, y int, c color.Color)

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
		newImg := image.NewRGBA(img.Bounds())
		draw.Draw(newImg, newImg.Bounds(), img, image.ZP, draw.Src)
		p.Image = newImg
	}
	return err
}

func (p *Pixl) Encode(w io.Writer) error {
	return png.Encode(w, p.Image)
}

func (p *Pixl) Pixelate(nb int, f func (x, y int, p *Pixl) color.Color) error {

	bounds := p.Image.Bounds()
	width  := bounds.Dx()
	height := bounds.Dy()

	p.NumCols = nb
	p.BlockSize = width / nb
	p.NumRows = height / p.BlockSize

	var x, y int

	// columns
	for x=0; x < p.NumCols; x++ {
		// rows
		for y=0; y < p.NumRows; y++ {
			p.FillBlock(x, y, f(x, y, p))
		}
	}

	p.Crop()
	return nil
}

func (p *Pixl) Crop() {
	subImg := p.Image.SubImage(image.Rect(0,0, p.NumCols * p.BlockSize, p.NumRows * p.BlockSize))
	newImg := image.NewRGBA(subImg.Bounds())
	draw.Draw(newImg, newImg.Bounds(), subImg, image.ZP, draw.Src)
	p.Image = newImg
}

func (p *Pixl) ColorAt(x, y int) color.Color {
	return p.random(x,y)
}

func (p *Pixl) Swap(x1, y1, x2, y2 int) error {
	c1 := p.ColorAt(x1, y1)
	c2 := p.ColorAt(x2, y2)
	p.FillBlock(x2, y2, c1)
	p.FillBlock(x1, y1, c2)
	return nil
}

// fisher-yates-ish shuffle
func (p *Pixl) Shuffle(f func(p *Pixl, x1, y1, x2, y2 int) bool) error {
	for i:= p.NumCols * p.NumRows - 1; i > 0; i-- {
		x1, y1 := p.GetXY(i)
		x2, y2 := p.GetXY(rand.Int() % (i + 1))
		if f(p, x1, y1, x2, y2) {
			p.Swap(x1, y1, x2, y2)
		}
	}
	return nil
}


func (p *Pixl) DoStep(frequency int, dist func (color.Color, color.Color) float64) {
	numCells := p.NumCols * p.NumRows

	// FIXME
	iters := int(math.Floor(float64(numCells) * float64(frequency)/float64(100)))

	for i:=0; i < iters; i++ {
		bn := rand.Int() % numCells
		x, y := p.GetXY(bn)

		var minScore, score float64
		var minX, minY, newX, newY int

		minScore = 10000000000000000 // TODO: fix this
		minX = -1
		minY = -1

		for xDelta := -1; xDelta < 2; xDelta++ {
			for yDelta:= -1; yDelta < 2; yDelta++ {

				score = 0
				newX = xDelta + x
				newY = yDelta + y

				// sanity check
				if newX >= 0 && newX < p.NumCols && newY >= 0 && newY < p.NumRows {

					// all nearest neighbors
					for xd2 := -1; xd2 < 2; xd2++ {
						for yd2 := -1; yd2 < 2; yd2++ {

							// neighboring point to mover
							newX2 := xd2 + newX
							newY2 := yd2 + newY

							// neighboring point to moved point
							newX3 := xd2 + x
							newY3 := yd2 + y

							if newX2 >= 0 && newX2 < p.NumCols && newY2 >= 0 && newY2 < p.NumRows {

								// dissimilarity of points close to mover in hypothetical position
								score += dist(p.ColorAt(x,y), p.ColorAt(newX2, newY2))

								// dissimilarity of points closed to moved in hypothetical position
								score += dist(p.ColorAt(newX, newY), p.ColorAt(newX3, newY3))

							}
						}
					}

					if score < minScore {
						minScore = score
						minX = newX
						minY = newY
					}
				}
			}
		}
		if minX >= 0 && minY >= 0 {
			p.Swap(x,y, minX, minY)
		}
	}
}


func (p *Pixl) GetXY(bn int) (x int, y int) {
	x = (bn % p.NumCols)
	y = (bn / p.NumCols)
	return x, y
}

func (p *Pixl) GetBlock(x, y int) image.Rectangle {
	bs := p.BlockSize
	return image.Rect(x*bs, y*bs, (x+1)*bs, (y+1)*bs)
}

func (p *Pixl) FillBlock(x int, y int, c color.Color) {
	draw.Draw(p.Image, p.GetBlock(x, y), &image.Uniform{c}, image.ZP, draw.Src)
}

func (p *Pixl) SortRows() {
	for i:=0; i<p.NumRows; i++ {
		sp := new(SubPixl)
		sp.Init(p, i*p.NumCols, (i+1)*p.NumCols)
		sp.Sort()
	}
}

func (p *Pixl) WriteToScreen() {
	draw.Draw(p.Window.Screen(), p.Window.Screen().Bounds(), p.Image, image.ZP, draw.Src)
	p.Window.FlushImage()
}

func (p *Pixl) random (x, y int) color.Color {
	subImg  := p.Image.SubImage(p.GetBlock(x, y))
	bounds  := subImg.Bounds()
	offsetX := rand.Int() % p.BlockSize
	offsetY := rand.Int() % p.BlockSize
	return p.Image.At(bounds.Min.X + offsetX, bounds.Min.Y + offsetY)
}

// represents a consecutive run of pixels
type SubPixl struct {
	p *Pixl
	Start int
	End int
}

func (sp *SubPixl) Init(p *Pixl, s, e int) {
	sp.p = p
	sp.Start = s
	sp.End = e
}

func (sp *SubPixl) Len() int {
	return (sp.End + 1) - sp.Start
}

func (sp *SubPixl) GetXY(i int) (x, y int) {
	x, y = sp.p.GetXY(sp.Start + i)
	return x,y
}

func (sp *SubPixl) Color(i int) color.Color {
	return sp.p.ColorAt(sp.GetXY(i))
}

func (sp *SubPixl) Swap(i, j int) {
	x1, y1 := sp.GetXY(i)
	x2, y2 := sp.GetXY(j)
	sp.p.Swap(x1, y1, x2, y2)
}

func (sp *SubPixl) Less(i, j int) bool {
	r1, g1, b1, _ := sp.Color(i).RGBA()
	r2, g2, b2, _ := sp.Color(j).RGBA()
	// currently sorts on blueness
	return ((r1 < r2 || b1 < b2 && false) || g1 < g2) // TODO: less arbitrary criterion
}

func (sp *SubPixl) Sort() {
	sort.Sort(sp)
}
