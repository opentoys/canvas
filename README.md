# Go canvas [![GoDoc](https://godoc.org/github.com/tfriedel6/canvas?status.svg)](https://godoc.org/github.com/tfriedel6/canvas)

Canvas is a pure Go library that provides drawing functionality as similar as possible to the HTML5 canvas API. It has nothing to do with HTML or Javascript, the functions are just made to be approximately the same.

Most of the functions are supported, but it is still a work in progress. The library aims to accept a lot of different parameters on each function in a similar way as the Javascript API does.

Whereas the Javascript API uses a context that all draw calls go to, here all draw calls are directly on the canvas type. The other difference is that here setters are used instead of properties for things like fonts and line width. 

## Software backend

The software backend can also be used if no OpenGL context is available. It will render into a standard Go RGBA image. 

There is experimental MSAA anti-aliasing, but it doesn't fully work properly yet. The best option for anti-aliasing currently is to render to a larger image and then scale it down.

# Example

Look at the example/drawing package for some drawing examples. 

Here is a simple example for how to get started:

```go
package main

import (
	"image/png"
	"math"
	"os"

	"github.com/opentoys/canvas"
)

func main() {
	backend := canvas.NewSoftware(720, 720)
	cv := canvas.New(backend)

	w, h := float64(cv.Width()), float64(cv.Height())
	cv.SetFillStyle("#000")
	cv.FillRect(0, 0, w, h)

	for r := 0.0; r < math.Pi*2; r += math.Pi * 0.1 {
		cv.SetFillStyle(int(r*10), int(r*20), int(r*40))
		cv.BeginPath()
		cv.MoveTo(w*0.5, h*0.5)
		cv.Arc(w*0.5, h*0.5, math.Min(w, h)*0.4, r, r+0.1*math.Pi, false)
		cv.ClosePath()
		cv.Fill()
	}

	cv.SetStrokeStyle("#FFF")
	cv.SetLineWidth(10)
	cv.BeginPath()
	cv.Arc(w*0.5, h*0.5, math.Min(w, h)*0.4, 0, math.Pi*2, false)
	cv.Stroke()

	f, err := os.OpenFile("result.png", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		panic(err)
	}
	err = png.Encode(f, backend.Image)
	if err != nil {
		panic(err)
	}
}
```

The result:

<img src="https://i.imgur.com/Nz8cT4M.png" width="320">

# Implemented features

These features *should* work just like their HTML5 counterparts, but there are likely to be a lot of edge cases where they don't work exactly the same way.

- beginPath
- closePath
- moveTo
- lineTo
- rect
- arc
- arcTo
- quadraticCurveTo
- bezierCurveTo
- stroke
- fill
- clip
- save
- restore
- scale
- translate
- rotate
- transform
- setTransform
- fillText
- measureText
- textAlign
- textBaseline
- fillStyle
- strokeText
- strokeStyle
- linear gradients
- radial gradients
- image patterns with repeat and transform
- lineWidth
- lineEnd (square, butt, round)
- lineJoin (bevel, miter, round)
- miterLimit
- lineDash
- getLineDash
- lineDashOffset
- global alpha
- drawImage
- getImageData
- putImageData
- clearRect
- shadowColor
- shadowOffset(X/Y)
- shadowBlur
- isPointInPath
- isPointInStroke
- self intersecting polygons

# Missing features

- globalCompositeOperation
- imageSmoothingEnabled
- textBaseline hanging and ideographic (currently work just like top and bottom)
