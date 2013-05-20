package pixl

import (
    "time"
    "image"
    "image/png"
    "image/color"
    "image/draw"
    "io"
    "math/rand"
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

    GetBlock(x, y int) image.Rectangle

    FillBlock(x, y int, c color.Color)

}


type Pixl struct {
    Image *image.RGBA
    NumBlocks int
    BlockSize int
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
    width := bounds.Dx()
    height := bounds.Dy()

    p.NumBlocks = nb
    p.BlockSize = width / nb

    var x, y int

    // columns
    for x=0; x < p.NumBlocks; x++ {
        // rows
        for y=0; y < height / p.BlockSize; y++ {
            p.FillBlock(x, y, f(x, y, p))
        }
    }

    // crop the image to size
    subImg := p.Image.SubImage(image.Rect(0,0, x * p.BlockSize, y * p.BlockSize))
    newImg := image.NewRGBA(subImg.Bounds())
    draw.Draw(newImg, newImg.Bounds(), subImg, image.ZP, draw.Src)
    p.Image = newImg

    return nil
}

func (p *Pixl) Swap(x1, y1, x2, y2 int) error {
    c1 := p.random(x1, y1)
    c2 := p.random(x2, y2)
    p.FillBlock(x2, y2, c1)
    p.FillBlock(x1, y1, c2)
    return nil
}

// fisher-yates-ish shuffle
func (p *Pixl) Shuffle(f func(p *Pixl, x1, y1, x2, y2 int) bool) error {

    rand.Seed(time.Now().UnixNano())

    bounds := p.Image.Bounds()
    height := bounds.Dy()
    numRow := (height / p.BlockSize)

    for i:= p.NumBlocks * numRow - 1; i > 0; i-- {
        x1, y1 := p.GetXY(i)
        x2, y2 := p.GetXY(rand.Int() % (i + 1))
        if f(p, x1, y1, x2, y2) {
            p.Swap(x1, y1, x2, y2)
        }
    }

    return nil
}

func (p *Pixl) GetXY(bn int) (x int, y int) {
    x = (bn % p.NumBlocks)
    y = (bn / p.NumBlocks)
    return x, y
}

// Gets the bounding box for a specific block
func (p *Pixl) GetBlock(x, y int) image.Rectangle {
    bs := p.BlockSize
    return image.Rect(x*bs, y*bs, (x+1)*bs, (y+1)*bs)
}


func (p *Pixl) FillBlock(x int, y int, c color.Color) {
    draw.Draw(p.Image, p.GetBlock(x, y), &image.Uniform{c}, image.ZP, draw.Src)
}

func (p *Pixl) random (x, y int) color.Color {
    subImg := p.Image.SubImage(p.GetBlock(x, y))
    bounds := subImg.Bounds()
    offsetX := rand.Int() % p.BlockSize
    offsetY := rand.Int() % p.BlockSize
    return p.Image.At(bounds.Min.X + offsetX, bounds.Min.Y + offsetY)
}
