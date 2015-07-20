package main

import "golang.org/x/tour/pic"

type Image struct {
	x int
	y int
	w int
	h int
}

func (i Image) Bounds() Rectangle {
	return image.Rect(m.x, m.y, m.w, m.h)
}

func (i Image) ColorModel() color.Model {
	//  return color.
}

func main() {
	m := Image{}
	m.x = 0
	m.y = 0
	m.w = 100
	m.h = 100
	pic.ShowImage(m)
}
