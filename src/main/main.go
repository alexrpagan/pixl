package main

import (
    "fmt"
    "pixl"
    "time"
    "math/rand"
    "flag"
    "os"
    "bufio"
    "image/color"
)

var input     = flag.String("i", "", "input file")
var output    = flag.String("o", "out.png", "output file")
var shuffle   = flag.Bool("s", false, "shuffle the pixels?")
var blocksize = flag.Int("b", 10, "blocksize")

func random (x, y int, p *pixl.Pixl) color.Color {
    rand.Seed(time.Now().UnixNano())
    subImg := p.Image.SubImage(p.GetBlock(x, y))
    bounds := subImg.Bounds()
    offsetX := rand.Int() % p.BlockSize
    offsetY := rand.Int() % p.BlockSize
    return p.Image.At(bounds.Min.X + offsetX, bounds.Min.Y + offsetY)
}

func unbiased (p *pixl.Pixl, x1, y1, x2, y2 int) bool {
    return true
}

func bluey (p *pixl.Pixl, x1, y1, x2, y2 int) bool {
    _, _, b1, _ := random(x1, y1, p).RGBA()
    return b1 < 30000
}

func main () {

    flag.Parse()

    if *input != "" {

        pix := new(pixl.Pixl)

        inf, err := os.Open(*input)
        defer inf.Close()
        if err == nil {
            reader := bufio.NewReader(inf)
            if pix.Decode(reader) != nil {
                os.Exit(1)
            }
        }

        pix.Pixelate(*blocksize, random)

        if *shuffle {
            pix.Shuffle(unbiased)
        }

        outf, err := os.Create(*output)
        if err == nil {
            writer := bufio.NewWriter(outf)
            pix.Encode(writer)
        } else {
            fmt.Println(err)
            os.Exit(1)
        }
        outf.Close()

    }

}

