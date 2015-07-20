package main

import (
	"fmt"
	"image"
)

// Package image defines the IMage interface:
//
//  package image
//
//  type Image interface {
//    ColorModel() color.Model
//    Bounds() Rectangle
//    At(x, y int) color.Color
//  }
//
// The color.Color and color.Model types are also interfaces but we'll ignore that by using the
// predefined implementations color.RGBA and color.RGBAModel. These interfaces and types are
// specified by the image/color package

func main() {
	m := image.NewRGBA(image.Rect(0, 0, 100, 100))
	fmt.Println(m.Bounds())
	fmt.Println(m.At(0, 0).RGBA())
}
