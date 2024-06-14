package canvas

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

type SoftwareBackend struct {
	Image *image.RGBA

	MSAA int

	blurSwap *image.RGBA

	clip    *image.Alpha
	stencil *image.Alpha
	w, h    int
}

func NewBackend(w, h int) *SoftwareBackend {
	b := &SoftwareBackend{}
	b.SetSize(w, h)
	return b
}

func (b *SoftwareBackend) SetSize(w, h int) {
	b.w, b.h = w, h
	b.Image = image.NewRGBA(image.Rect(0, 0, w, h))
	b.clip = image.NewAlpha(image.Rect(0, 0, w, h))
	b.stencil = image.NewAlpha(image.Rect(0, 0, w, h))
	b.ClearClip()
}

func (b *SoftwareBackend) Bytes() []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, b.Image)
	return buf.Bytes()
}

func (b *SoftwareBackend) Size() (int, int) {
	return b.w, b.h
}

func (b *SoftwareBackend) GetImageData(x, y, w, h int) *image.RGBA {
	return b.Image.SubImage(image.Rect(x, y, w, h)).(*image.RGBA)
}

func (b *SoftwareBackend) PutImageData(img *image.RGBA, x, y int) {
	draw.Draw(b.Image, image.Rect(x, y, img.Rect.Dx(), img.Rect.Dy()), img, image.ZP, draw.Src)
}

func (b *SoftwareBackend) CanUseAsImage(b2 Backend) bool {
	return false
}

func (b *SoftwareBackend) AsImage() BackendImage {
	return nil
}

type SoftwareLinearGradient struct {
	data BackendGradient
}
type SoftwareRadialGradient struct {
	data BackendGradient
}

func (b *SoftwareBackend) LoadLinearGradient(data BackendGradient) BackendLinearGradient {
	return &SoftwareLinearGradient{data: data}
}

func (b *SoftwareBackend) LoadRadialGradient(data BackendGradient) BackendRadialGradient {
	return &SoftwareRadialGradient{data: data}
}

func (g *SoftwareLinearGradient) Delete() {
}

func (g *SoftwareLinearGradient) Replace(data BackendGradient) {
	g.data = data
}

func (g *SoftwareRadialGradient) Delete() {
}

func (g *SoftwareRadialGradient) Replace(data BackendGradient) {
	g.data = data
}

func (b *SoftwareBackend) activateBlurTarget() {
	b.blurSwap = b.Image
	b.Image = image.NewRGBA(b.Image.Rect)
}

func (b *SoftwareBackend) drawBlurred(size float64) {
	blurred := box3(b.Image, size)
	b.Image = b.blurSwap
	draw.Draw(b.Image, b.Image.Rect, blurred, image.ZP, draw.Over)
}

func box3(img *image.RGBA, size float64) *image.RGBA {
	size *= 1 - 1/(size+1) // this just seems to improve the accuracy

	fsize := math.Floor(size)
	sizea := int(fsize)
	sizeb := sizea
	sizec := sizea
	if size-fsize > 0.333333333 {
		sizeb++
	}
	if size-fsize > 0.666666666 {
		sizec++
	}
	img = box3x(img, sizea)
	img = box3x(img, sizeb)
	img = box3x(img, sizec)
	img = box3y(img, sizea)
	img = box3y(img, sizeb)
	img = box3y(img, sizec)
	return img
}

func box3x(img *image.RGBA, size int) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)
	w, h := bounds.Dx(), bounds.Dy()

	for y := 0; y < h; y++ {
		if size >= w {
			var r, g, b, a float64
			for x := 0; x < w; x++ {
				col := img.RGBAAt(x, y)
				r += float64(col.R)
				g += float64(col.G)
				b += float64(col.B)
				a += float64(col.A)
			}

			factor := 1.0 / float64(w)
			col := color.RGBA{
				R: uint8(math.Round(r * factor)),
				G: uint8(math.Round(g * factor)),
				B: uint8(math.Round(b * factor)),
				A: uint8(math.Round(a * factor)),
			}
			for x := 0; x < w; x++ {
				result.SetRGBA(x, y, col)
			}
			continue
		}

		var r, g, b, a float64
		for x := 0; x <= size; x++ {
			col := img.RGBAAt(x, y)
			r += float64(col.R)
			g += float64(col.G)
			b += float64(col.B)
			a += float64(col.A)
		}

		samples := size + 1
		x := 0
		for {
			factor := 1.0 / float64(samples)
			col := color.RGBA{
				R: uint8(math.Round(r * factor)),
				G: uint8(math.Round(g * factor)),
				B: uint8(math.Round(b * factor)),
				A: uint8(math.Round(a * factor)),
			}
			result.SetRGBA(x, y, col)

			if x >= w-1 {
				break
			}

			if left := x - size; left >= 0 {
				col = img.RGBAAt(left, y)
				r -= float64(col.R)
				g -= float64(col.G)
				b -= float64(col.B)
				a -= float64(col.A)
				samples--
			}

			x++

			if right := x + size; right < w {
				col = img.RGBAAt(right, y)
				r += float64(col.R)
				g += float64(col.G)
				b += float64(col.B)
				a += float64(col.A)
				samples++
			}
		}
	}

	return result
}

func box3y(img *image.RGBA, size int) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)
	w, h := bounds.Dx(), bounds.Dy()

	for x := 0; x < w; x++ {
		if size >= h {
			var r, g, b, a float64
			for y := 0; y < h; y++ {
				col := img.RGBAAt(x, y)
				r += float64(col.R)
				g += float64(col.G)
				b += float64(col.B)
				a += float64(col.A)
			}

			factor := 1.0 / float64(h)
			col := color.RGBA{
				R: uint8(math.Round(r * factor)),
				G: uint8(math.Round(g * factor)),
				B: uint8(math.Round(b * factor)),
				A: uint8(math.Round(a * factor)),
			}
			for y := 0; y < h; y++ {
				result.SetRGBA(x, y, col)
			}
			continue
		}

		var r, g, b, a float64
		for y := 0; y <= size; y++ {
			col := img.RGBAAt(x, y)
			r += float64(col.R)
			g += float64(col.G)
			b += float64(col.B)
			a += float64(col.A)
		}

		samples := size + 1
		y := 0
		for {
			factor := 1.0 / float64(samples)
			col := color.RGBA{
				R: uint8(math.Round(r * factor)),
				G: uint8(math.Round(g * factor)),
				B: uint8(math.Round(b * factor)),
				A: uint8(math.Round(a * factor)),
			}
			result.SetRGBA(x, y, col)

			if y >= h-1 {
				break
			}

			if top := y - size; top >= 0 {
				col = img.RGBAAt(x, top)
				r -= float64(col.R)
				g -= float64(col.G)
				b -= float64(col.B)
				a -= float64(col.A)
				samples--
			}

			y++

			if bottom := y + size; bottom < h {
				col = img.RGBAAt(x, bottom)
				r += float64(col.R)
				g += float64(col.G)
				b += float64(col.B)
				a += float64(col.A)
				samples++
			}
		}
	}

	return result
}

func triangleLR(tri []BackendVec, y float64) (l, r float64, outside bool) {
	a, b, c := tri[0], tri[1], tri[2]

	// sort by y
	if a[1] > b[1] {
		a, b = b, a
	}
	if b[1] > c[1] {
		b, c = c, b
		if a[1] > b[1] {
			a, b = b, a
		}
	}

	// check general bounds
	if y <= a[1] {
		return a[0], a[0], true
	}
	if y > c[1] {
		return c[0], c[0], true
	}

	// find left and right x at y
	if y >= a[1] && y <= b[1] && a[1] < b[1] {
		r0 := (y - a[1]) / (b[1] - a[1])
		l = (b[0]-a[0])*r0 + a[0]
		r1 := (y - a[1]) / (c[1] - a[1])
		r = (c[0]-a[0])*r1 + a[0]
	} else {
		r0 := (y - b[1]) / (c[1] - b[1])
		l = (c[0]-b[0])*r0 + b[0]
		r1 := (y - a[1]) / (c[1] - a[1])
		r = (c[0]-a[0])*r1 + a[0]
	}
	if l > r {
		l, r = r, l
	}

	return
}

func (b *SoftwareBackend) fillTriangleNoAA(tri []BackendVec, fn func(x, y int)) {
	minY := int(math.Floor(math.Min(math.Min(tri[0][1], tri[1][1]), tri[2][1])))
	maxY := int(math.Ceil(math.Max(math.Max(tri[0][1], tri[1][1]), tri[2][1])))
	if minY < 0 {
		minY = 0
	} else if minY >= b.h {
		return
	}
	if maxY < 0 {
		return
	} else if maxY >= b.h {
		maxY = b.h - 1
	}
	for y := minY; y <= maxY; y++ {
		l, r, out := triangleLR(tri, float64(y)+0.5)
		if out {
			continue
		}
		if l < 0 {
			l = 0
		} else if l > float64(b.w) {
			continue
		}
		if r < 0 {
			continue
		} else if r > float64(b.w) {
			r = float64(b.w)
		}
		if l >= r {
			continue
		}
		fl, cr := int(math.Floor(l)), int(math.Ceil(r))
		for x := fl; x <= cr; x++ {
			fx := float64(x) + 0.5
			if fx < l || fx >= r {
				continue
			}
			fn(x, y)
		}
	}
}

type msaaPixel struct {
	ix, iy int
	fx, fy float64
	tx, ty float64
}

func (b *SoftwareBackend) fillTriangleMSAA(tri []BackendVec, msaaLevel int, msaaPixels []msaaPixel, fn func(x, y int)) []msaaPixel {
	msaaStep := 1.0 / float64(msaaLevel+1)

	minY := int(math.Floor(math.Min(math.Min(tri[0][1], tri[1][1]), tri[2][1])))
	maxY := int(math.Ceil(math.Max(math.Max(tri[0][1], tri[1][1]), tri[2][1])))
	if minY < 0 {
		minY = 0
	} else if minY >= b.h {
		return msaaPixels
	}
	if maxY < 0 {
		return msaaPixels
	} else if maxY >= b.h {
		maxY = b.h - 1
	}

	for y := minY; y <= maxY; y++ {
		var l, r [5]float64
		allOut := true
		minL, maxR := math.MaxFloat64, 0.0

		sy := float64(y) + msaaStep*0.5
		for step := 0; step <= msaaLevel; step++ {
			var out bool
			l[step], r[step], out = triangleLR(tri, sy)
			if l[step] < 0 {
				l[step] = 0
			} else if l[step] > float64(b.w) {
				l[step] = float64(b.w)
				out = true
			}
			if r[step] < 0 {
				r[step] = 0
				out = true
			} else if r[step] > float64(b.w) {
				r[step] = float64(b.w)
			}
			if r[step] <= l[step] {
				out = true
			}
			if !out {
				allOut = false
				minL = math.Min(minL, l[step])
				maxR = math.Max(maxR, r[step])
			}
			sy += msaaStep
		}

		if allOut {
			continue
		}

		fl, cr := int(math.Floor(minL)), int(math.Ceil(maxR))
		for x := fl; x <= cr; x++ {
			sy = float64(y) + msaaStep*0.5
			allIn := true
		check:
			for stepy := 0; stepy <= msaaLevel; stepy++ {
				sx := float64(x) + msaaStep*0.5
				for stepx := 0; stepx <= msaaLevel; stepx++ {
					if sx < l[stepy] || sx >= r[stepy] {
						allIn = false
						break check
					}
					sx += msaaStep
				}
				sy += msaaStep
			}

			if allIn {
				fn(x, y)
				continue
			}

			sy = float64(y) + msaaStep*0.5
			for stepy := 0; stepy <= msaaLevel; stepy++ {
				sx := float64(x) + msaaStep*0.5
				for stepx := 0; stepx <= msaaLevel; stepx++ {
					if sx >= l[stepy] && sx < r[stepy] {
						msaaPixels = addMSAAPixel(msaaPixels, msaaPixel{ix: x, iy: y, fx: sx, fy: sy})
					}
					sx += msaaStep
				}
				sy += msaaStep
			}
		}
	}

	return msaaPixels
}

func addMSAAPixel(msaaPixels []msaaPixel, px msaaPixel) []msaaPixel {
	for _, px2 := range msaaPixels {
		if px == px2 {
			return msaaPixels
		}
	}
	return append(msaaPixels, px)
}

func quadArea(quad [4]BackendVec) float64 {
	leftv := BackendVec{quad[1][0] - quad[0][0], quad[1][1] - quad[0][1]}
	topv := BackendVec{quad[3][0] - quad[0][0], quad[3][1] - quad[0][1]}
	return math.Abs(leftv[0]*topv[1] - leftv[1]*topv[0])
}

func (b *SoftwareBackend) fillQuadNoAA(quad [4]BackendVec, fn func(x, y int, tx, ty float64)) {
	minY := int(math.Floor(math.Min(math.Min(quad[0][1], quad[1][1]), math.Min(quad[2][1], quad[3][1]))))
	maxY := int(math.Ceil(math.Max(math.Max(quad[0][1], quad[1][1]), math.Max(quad[2][1], quad[3][1]))))
	if minY < 0 {
		minY = 0
	} else if minY >= b.h {
		return
	}
	if maxY < 0 {
		return
	} else if maxY >= b.h {
		maxY = b.h - 1
	}

	leftv := BackendVec{quad[1][0] - quad[0][0], quad[1][1] - quad[0][1]}
	leftLen := math.Sqrt(leftv[0]*leftv[0] + leftv[1]*leftv[1])
	leftv[0] /= leftLen
	leftv[1] /= leftLen
	topv := BackendVec{quad[3][0] - quad[0][0], quad[3][1] - quad[0][1]}
	topLen := math.Sqrt(topv[0]*topv[0] + topv[1]*topv[1])
	topv[0] /= topLen
	topv[1] /= topLen

	tri1 := [3]BackendVec{quad[0], quad[1], quad[2]}
	tri2 := [3]BackendVec{quad[0], quad[2], quad[3]}
	for y := minY; y <= maxY; y++ {
		lf1, rf1, out1 := triangleLR(tri1[:], float64(y)+0.5)
		lf2, rf2, out2 := triangleLR(tri2[:], float64(y)+0.5)
		if out1 && out2 {
			continue
		}
		l := math.Min(lf1, lf2)
		r := math.Max(rf1, rf2)
		if l < 0 {
			l = 0
		} else if l > float64(b.w) {
			continue
		}
		if r < 0 {
			continue
		} else if r > float64(b.w) {
			r = float64(b.w)
		}
		if l >= r {
			continue
		}

		tfy := float64(y) + 0.5 - quad[0][1]
		fl, cr := int(math.Floor(l)), int(math.Ceil(r))
		for x := fl; x <= cr; x++ {
			fx := float64(x) + 0.5
			if fx < l || fx >= r {
				continue
			}
			tfx := fx - quad[0][0]

			var tx, ty float64
			if math.Abs(leftv[0]) > math.Abs(leftv[1]) {
				tx = (tfy - tfx*(leftv[1]/leftv[0])) / (topv[1] - topv[0]*(leftv[1]/leftv[0]))
				ty = (tfx - topv[0]*tx) / leftv[0]
			} else {
				tx = (tfx - tfy*(leftv[0]/leftv[1])) / (topv[0] - topv[1]*(leftv[0]/leftv[1]))
				ty = (tfy - topv[1]*tx) / leftv[1]
			}

			fn(x, y, tx/topLen, ty/leftLen)
		}
	}
}

func (b *SoftwareBackend) fillQuadMSAA(quad [4]BackendVec, msaaLevel int, msaaPixels []msaaPixel, fn func(x, y int, tx, ty float64)) []msaaPixel {
	msaaStep := 1.0 / float64(msaaLevel+1)

	minY := int(math.Floor(math.Min(math.Min(quad[0][1], quad[1][1]), math.Min(quad[2][1], quad[3][1]))))
	maxY := int(math.Ceil(math.Max(math.Max(quad[0][1], quad[1][1]), math.Max(quad[2][1], quad[3][1]))))
	if minY < 0 {
		minY = 0
	} else if minY >= b.h {
		return msaaPixels
	}
	if maxY < 0 {
		return msaaPixels
	} else if maxY >= b.h {
		maxY = b.h - 1
	}

	leftv := BackendVec{quad[1][0] - quad[0][0], quad[1][1] - quad[0][1]}
	leftLen := math.Sqrt(leftv[0]*leftv[0] + leftv[1]*leftv[1])
	leftv[0] /= leftLen
	leftv[1] /= leftLen
	topv := BackendVec{quad[3][0] - quad[0][0], quad[3][1] - quad[0][1]}
	topLen := math.Sqrt(topv[0]*topv[0] + topv[1]*topv[1])
	topv[0] /= topLen
	topv[1] /= topLen

	tri1 := [3]BackendVec{quad[0], quad[1], quad[2]}
	tri2 := [3]BackendVec{quad[0], quad[2], quad[3]}
	for y := minY; y <= maxY; y++ {
		var l, r [5]float64
		allOut := true
		minL, maxR := math.MaxFloat64, 0.0

		sy := float64(y) + msaaStep*0.5
		for step := 0; step <= msaaLevel; step++ {
			lf1, rf1, out1 := triangleLR(tri1[:], sy)
			lf2, rf2, out2 := triangleLR(tri2[:], sy)
			l[step] = math.Min(lf1, lf2)
			r[step] = math.Max(rf1, rf2)
			out := out1 || out2

			if l[step] < 0 {
				l[step] = 0
			} else if l[step] > float64(b.w) {
				l[step] = float64(b.w)
				out = true
			}
			if r[step] < 0 {
				r[step] = 0
				out = true
			} else if r[step] > float64(b.w) {
				r[step] = float64(b.w)
			}
			if r[step] <= l[step] {
				out = true
			}
			if !out {
				allOut = false
				minL = math.Min(minL, l[step])
				maxR = math.Max(maxR, r[step])
			}
			sy += msaaStep
		}

		if allOut {
			continue
		}

		fl, cr := int(math.Floor(minL)), int(math.Ceil(maxR))
		for x := fl; x <= cr; x++ {
			sy = float64(y) + msaaStep*0.5
			allIn := true
		check:
			for stepy := 0; stepy <= msaaLevel; stepy++ {
				sx := float64(x) + msaaStep*0.5
				for stepx := 0; stepx <= msaaLevel; stepx++ {
					if sx < l[stepy] || sx >= r[stepy] {
						allIn = false
						break check
					}
					sx += msaaStep
				}
				sy += msaaStep
			}

			if allIn {
				tfx := float64(x) + 0.5 - quad[0][0]
				tfy := float64(y) + 0.5 - quad[0][1]

				var tx, ty float64
				if math.Abs(leftv[0]) > math.Abs(leftv[1]) {
					tx = (tfy - tfx*(leftv[1]/leftv[0])) / (topv[1] - topv[0]*(leftv[1]/leftv[0]))
					ty = (tfx - topv[0]*tx) / leftv[0]
				} else {
					tx = (tfx - tfy*(leftv[0]/leftv[1])) / (topv[0] - topv[1]*(leftv[0]/leftv[1]))
					ty = (tfy - topv[1]*tx) / leftv[1]
				}

				fn(x, y, tx/topLen, ty/leftLen)
				continue
			}

			sy = float64(y) + msaaStep*0.5
			for stepy := 0; stepy <= msaaLevel; stepy++ {
				sx := float64(x) + msaaStep*0.5
				for stepx := 0; stepx <= msaaLevel; stepx++ {
					if sx >= l[stepy] && sx < r[stepy] {
						tfx := sx - quad[0][0]
						tfy := sy - quad[0][1]

						var tx, ty float64
						if math.Abs(leftv[0]) > math.Abs(leftv[1]) {
							tx = (tfy - tfx*(leftv[1]/leftv[0])) / (topv[1] - topv[0]*(leftv[1]/leftv[0]))
							ty = (tfx - topv[0]*tx) / leftv[0]
						} else {
							tx = (tfx - tfy*(leftv[0]/leftv[1])) / (topv[0] - topv[1]*(leftv[0]/leftv[1]))
							ty = (tfy - topv[1]*tx) / leftv[1]
						}

						msaaPixels = addMSAAPixel(msaaPixels, msaaPixel{ix: x, iy: y, fx: sx, fy: sy, tx: tx / topLen, ty: ty / leftLen})
					}
					sx += msaaStep
				}
				sy += msaaStep
			}
		}
	}

	return msaaPixels
}

func (b *SoftwareBackend) fillQuad(pts [4]BackendVec, fn func(x, y, tx, ty float64) color.RGBA) {
	b.clearStencil()

	if b.MSAA > 0 {
		var msaaPixelBuf [500]msaaPixel
		msaaPixels := msaaPixelBuf[:0]

		msaaPixels = b.fillQuadMSAA(pts, b.MSAA, msaaPixels, func(x, y int, tx, ty float64) {
			if b.clip.AlphaAt(x, y).A == 0 {
				return
			}
			if b.stencil.AlphaAt(x, y).A > 0 {
				return
			}
			b.stencil.SetAlpha(x, y, color.Alpha{A: 255})
			col := fn(float64(x)+0.5, float64(y)+0.5, tx, ty)
			if col.A > 0 {
				b.Image.SetRGBA(x, y, mix(col, b.Image.RGBAAt(x, y)))
			}
		})

		samples := (b.MSAA + 1) * (b.MSAA + 1)

		for i, px := range msaaPixels {
			if px.ix < 0 || b.clip.AlphaAt(px.ix, px.iy).A == 0 || b.stencil.AlphaAt(px.ix, px.iy).A > 0 {
				continue
			}
			b.stencil.SetAlpha(px.ix, px.iy, color.Alpha{A: 255})

			var mr, mg, mb, ma int
			for j, px2 := range msaaPixels[i:] {
				if px2.ix != px.ix || px2.iy != px.iy {
					continue
				}

				col := fn(px2.fx, px2.fy, px2.tx, px2.ty)
				mr += int(col.R)
				mg += int(col.G)
				mb += int(col.B)
				ma += int(col.A)

				msaaPixels[i+j].ix = -1
			}

			combined := color.RGBA{
				R: uint8(mr / samples),
				G: uint8(mg / samples),
				B: uint8(mb / samples),
				A: uint8(ma / samples),
			}
			b.Image.SetRGBA(px.ix, px.iy, mix(combined, b.Image.RGBAAt(px.ix, px.iy)))
		}

	} else {
		b.fillQuadNoAA(pts, func(x, y int, tx, ty float64) {
			if b.clip.AlphaAt(x, y).A == 0 {
				return
			}
			if b.stencil.AlphaAt(x, y).A > 0 {
				return
			}
			b.stencil.SetAlpha(x, y, color.Alpha{A: 255})
			col := fn(float64(x)+0.5, float64(y)+0.5, tx, ty)
			if col.A > 0 {
				b.Image.SetRGBA(x, y, mix(col, b.Image.RGBAAt(x, y)))
			}
		})
	}
}

func iterateTriangles(pts []BackendVec, fn func(tri []BackendVec)) {
	if len(pts) == 4 {
		var buf [3]BackendVec
		buf[0] = pts[0]
		buf[1] = pts[1]
		buf[2] = pts[2]
		fn(buf[:])
		buf[1] = pts[2]
		buf[2] = pts[3]
		fn(buf[:])
		return
	}
	for i := 3; i <= len(pts); i += 3 {
		fn(pts[i-3 : i])
	}
}

func (b *SoftwareBackend) fillTrianglesNoAA(pts []BackendVec, fn func(x, y float64) color.RGBA) {
	iterateTriangles(pts[:], func(tri []BackendVec) {
		b.fillTriangleNoAA(tri, func(x, y int) {
			if b.clip.AlphaAt(x, y).A == 0 {
				return
			}
			if b.stencil.AlphaAt(x, y).A > 0 {
				return
			}
			b.stencil.SetAlpha(x, y, color.Alpha{A: 255})
			col := fn(float64(x), float64(y))
			if col.A > 0 {
				b.Image.SetRGBA(x, y, mix(col, b.Image.RGBAAt(x, y)))
			}
		})
	})
}

func (b *SoftwareBackend) fillTrianglesMSAA(pts []BackendVec, msaaLevel int, fn func(x, y float64) color.RGBA) {
	var msaaPixelBuf [500]msaaPixel
	msaaPixels := msaaPixelBuf[:0]

	iterateTriangles(pts[:], func(tri []BackendVec) {
		msaaPixels = b.fillTriangleMSAA(tri, msaaLevel, msaaPixels, func(x, y int) {
			if b.clip.AlphaAt(x, y).A == 0 {
				return
			}
			if b.stencil.AlphaAt(x, y).A > 0 {
				return
			}
			b.stencil.SetAlpha(x, y, color.Alpha{A: 255})
			col := fn(float64(x), float64(y))
			if col.A > 0 {
				b.Image.SetRGBA(x, y, mix(col, b.Image.RGBAAt(x, y)))
			}
		})
	})

	samples := (msaaLevel + 1) * (msaaLevel + 1)

	for i, px := range msaaPixels {
		if px.ix < 0 || b.clip.AlphaAt(px.ix, px.iy).A == 0 || b.stencil.AlphaAt(px.ix, px.iy).A > 0 {
			continue
		}
		b.stencil.SetAlpha(px.ix, px.iy, color.Alpha{A: 255})

		var mr, mg, mb, ma int
		for j, px2 := range msaaPixels[i:] {
			if px2.ix != px.ix || px2.iy != px.iy {
				continue
			}

			col := fn(px2.fx, px2.fy)
			mr += int(col.R)
			mg += int(col.G)
			mb += int(col.B)
			ma += int(col.A)

			msaaPixels[i+j].ix = -1
		}

		combined := color.RGBA{
			R: uint8(mr / samples),
			G: uint8(mg / samples),
			B: uint8(mb / samples),
			A: uint8(ma / samples),
		}
		b.Image.SetRGBA(px.ix, px.iy, mix(combined, b.Image.RGBAAt(px.ix, px.iy)))
	}
}

func (b *SoftwareBackend) fillTriangles(pts []BackendVec, fn func(x, y float64) color.RGBA) {
	b.clearStencil()

	if b.MSAA > 0 {
		b.fillTrianglesMSAA(pts, b.MSAA, fn)
	} else {
		b.fillTrianglesNoAA(pts, fn)
	}
}

type SoftwareImage struct {
	mips    []image.Image
	deleted bool
}

func (b *SoftwareBackend) LoadImage(img image.Image) (BackendImage, error) {
	bimg := &SoftwareImage{mips: make([]image.Image, 1, 10)}
	bimg.Replace(img)
	return bimg, nil
}

func halveImage(img image.Image) (*image.RGBA, int, int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	w = w / 2
	h = h / 2
	rimg := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := y * 2
		for x := 0; x < w; x++ {
			sx := x * 2
			r1, g1, b1, a1 := img.At(sx, sy).RGBA()
			r2, g2, b2, a2 := img.At(sx+1, sy).RGBA()
			r3, g3, b3, a3 := img.At(sx, sy+1).RGBA()
			r4, g4, b4, a4 := img.At(sx+1, sy+1).RGBA()
			mixr := uint8((int(r1) + int(r2) + int(r3) + int(r4)) / 1024)
			mixg := uint8((int(g1) + int(g2) + int(g3) + int(g4)) / 1024)
			mixb := uint8((int(b1) + int(b2) + int(b3) + int(b4)) / 1024)
			mixa := uint8((int(a1) + int(a2) + int(a3) + int(a4)) / 1024)
			rimg.Set(x, y, color.RGBA{R: mixr, G: mixg, B: mixb, A: mixa})
		}
	}
	return rimg, w, h
}

func (b *SoftwareBackend) DrawImage(dimg BackendImage, sx, sy, sw, sh float64, pts [4]BackendVec, alpha float64) {
	simg := dimg.(*SoftwareImage)
	if simg.deleted {
		return
	}

	bounds := simg.mips[0].Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	factor := float64(w*h) / (sw * sh)
	area := quadArea(pts) * factor
	mip := simg.mips[0]
	closest := math.MaxFloat64
	mipW, mipH := w, h
	for _, img := range simg.mips {
		bounds := img.Bounds()
		w, h := bounds.Dx(), bounds.Dy()
		dist := math.Abs(float64(w*h) - area)
		if dist < closest {
			closest = dist
			mip = img
			mipW = w
			mipH = h
		}
	}

	mipScaleX := float64(mipW) / float64(w)
	mipScaleY := float64(mipH) / float64(h)
	sx *= mipScaleX
	sy *= mipScaleY
	sw *= mipScaleX
	sh *= mipScaleY

	b.fillQuad(pts, func(x, y, tx, ty float64) color.RGBA {
		imgx := sx + sw*tx
		imgy := sy + sh*ty
		imgxf := math.Floor(imgx)
		imgyf := math.Floor(imgy)
		return toRGBA(mip.At(int(imgxf), int(imgyf)))

		// rx := imgx - imgxf
		// ry := imgy - imgyf
		// ca := mip.At(int(imgxf), int(imgyf))
		// cb := mip.At(int(imgxf+1), int(imgyf))
		// cc := mip.At(int(imgxf), int(imgyf+1))
		// cd := mip.At(int(imgxf+1), int(imgyf+1))
		// ctop := lerp(ca, cb, rx)
		// cbtm := lerp(cc, cd, rx)
		// b.Image.Set(x, y, lerp(ctop, cbtm, ry))
	})
}

func (img *SoftwareImage) Width() int {
	return img.mips[0].Bounds().Dx()
}

func (img *SoftwareImage) Height() int {
	return img.mips[0].Bounds().Dy()
}

func (img *SoftwareImage) Size() (w, h int) {
	b := img.mips[0].Bounds()
	return b.Dx(), b.Dy()
}

func (img *SoftwareImage) Delete() {
	img.deleted = true
}

func (img *SoftwareImage) Replace(src image.Image) error {
	img.mips = img.mips[:1]
	img.mips[0] = src

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	for w > 1 && h > 1 {
		src, w, h = halveImage(src)
		img.mips = append(img.mips, src)
	}

	return nil
}

type SoftwareImagePattern struct {
	data BackendImagePatternData
}

func (b *SoftwareBackend) LoadImagePattern(data BackendImagePatternData) BackendImagePattern {
	return &SoftwareImagePattern{
		data: data,
	}
}

func (ip *SoftwareImagePattern) Delete()                              {}
func (ip *SoftwareImagePattern) Replace(data BackendImagePatternData) { ip.data = data }

func (b *SoftwareBackend) Clear(pts [4]BackendVec) {
	iterateTriangles(pts[:], func(tri []BackendVec) {
		b.fillTriangleNoAA(tri, func(x, y int) {
			if b.clip.AlphaAt(x, y).A == 0 {
				return
			}
			b.Image.SetRGBA(x, y, color.RGBA{})
		})
	})
}

func (b *SoftwareBackend) Fill(style *BackendFillStyle, pts []BackendVec, tf BackendMat, canOverlap bool) {
	ffn := fillFunc(style)

	var triBuf [500]BackendVec
	if tf != BackendMatIdentity {
		ptsOld := pts
		if len(pts) < len(triBuf) {
			pts = triBuf[:len(pts)]
		} else {
			pts = make([]BackendVec, len(pts))
		}
		for i, pt := range ptsOld {
			pts[i] = pt.MulMat(tf)
		}
	}

	if style.Blur > 0 {
		b.activateBlurTarget()
		b.fillTriangles(pts, ffn)
		b.drawBlurred(style.Blur)
	} else {
		b.fillTriangles(pts, ffn)
	}
}

func (b *SoftwareBackend) FillImageMask(style *BackendFillStyle, mask *image.Alpha, pts [4]BackendVec) {
	ffn := fillFunc(style)

	mw := float64(mask.Bounds().Dx())
	mh := float64(mask.Bounds().Dy())
	b.fillQuad(pts, func(x, y, sx2, sy2 float64) color.RGBA {
		sxi := int(mw * sx2)
		syi := int(mh * sy2)
		a := mask.AlphaAt(sxi, syi)
		if a.A == 0 {
			return color.RGBA{}
		}
		col := ffn(x, y)
		return alphaColor(col, a)
	})
}

func fillFunc(style *BackendFillStyle) func(x, y float64) color.RGBA {
	if lg := style.LinearGradient; lg != nil {
		lg := lg.(*SoftwareLinearGradient)
		from := BackendVec{style.Gradient.X0, style.Gradient.Y0}
		dir := BackendVec{style.Gradient.X1 - style.Gradient.X0, style.Gradient.Y1 - style.Gradient.Y0}
		dirlen := math.Sqrt(dir[0]*dir[0] + dir[1]*dir[1])
		dir[0] /= dirlen
		dir[1] /= dirlen
		return func(x, y float64) color.RGBA {
			pos := BackendVec{x - from[0], y - from[1]}
			r := (pos[0]*dir[0] + pos[1]*dir[1]) / dirlen
			return lg.data.ColorAt(r)
		}
	} else if rg := style.RadialGradient; rg != nil {
		rg := rg.(*SoftwareRadialGradient)
		from := BackendVec{style.Gradient.X0, style.Gradient.Y0}
		to := BackendVec{style.Gradient.X1, style.Gradient.Y1}
		radFrom := style.Gradient.RadFrom
		radTo := style.Gradient.RadTo
		return func(x, y float64) color.RGBA {
			pos := BackendVec{x, y}
			oa := 0.5 * math.Sqrt(
				math.Pow(-2.0*from[0]*from[0]+2.0*from[0]*to[0]+2.0*from[0]*pos[0]-2.0*to[0]*pos[0]-2.0*from[1]*from[1]+2.0*from[1]*to[1]+2.0*from[1]*pos[1]-2.0*to[1]*pos[1]+2.0*radFrom*radFrom-2.0*radFrom*radTo, 2.0)-
					4.0*(from[0]*from[0]-2.0*from[0]*pos[0]+pos[0]*pos[0]+from[1]*from[1]-2.0*from[1]*pos[1]+pos[1]*pos[1]-radFrom*radFrom)*
						(from[0]*from[0]-2.0*from[0]*to[0]+to[0]*to[0]+from[1]*from[1]-2.0*from[1]*to[1]+to[1]*to[1]-radFrom*radFrom+2.0*radFrom*radTo-radTo*radTo))
			ob := (from[0]*from[0] - from[0]*to[0] - from[0]*pos[0] + to[0]*pos[0] + from[1]*from[1] - from[1]*to[1] - from[1]*pos[1] + to[1]*pos[1] - radFrom*radFrom + radFrom*radTo)
			oc := (from[0]*from[0] - 2.0*from[0]*to[0] + to[0]*to[0] + from[1]*from[1] - 2.0*from[1]*to[1] + to[1]*to[1] - radFrom*radFrom + 2.0*radFrom*radTo - radTo*radTo)
			o1 := (-oa + ob) / oc
			o2 := (oa + ob) / oc
			if math.IsNaN(o1) && math.IsNaN(o2) {
				return color.RGBA{}
			}
			o := math.Max(o1, o2)
			return rg.data.ColorAt(o)
		}
	} else if ip := style.ImagePattern; ip != nil {
		ip := ip.(*SoftwareImagePattern)
		img := ip.data.Image.(*SoftwareImage)
		mip := img.mips[0] // todo select the right mip size
		w, h := img.Size()
		fw, fh := float64(w), float64(h)
		rx := ip.data.Repeat == BackendRepeat || ip.data.Repeat == BackendRepeatX
		ry := ip.data.Repeat == BackendRepeat || ip.data.Repeat == BackendRepeatY
		return func(x, y float64) color.RGBA {
			pos := BackendVec{x, y}
			tfptx := pos[0]*ip.data.Transform[0] + pos[1]*ip.data.Transform[1] + ip.data.Transform[2]
			tfpty := pos[0]*ip.data.Transform[3] + pos[1]*ip.data.Transform[4] + ip.data.Transform[5]

			if !rx && (tfptx < 0 || tfptx >= fw) {
				return color.RGBA{}
			}
			if !ry && (tfpty < 0 || tfpty >= fh) {
				return color.RGBA{}
			}

			mx := int(math.Floor(tfptx)) % w
			if mx < 0 {
				mx += w
			}
			my := int(math.Floor(tfpty)) % h
			if my < 0 {
				my += h
			}

			return toRGBA(mip.At(mx, my))
		}
	}
	return func(x, y float64) color.RGBA {
		return style.Color
	}
}

func (b *SoftwareBackend) clearStencil() {
	p := b.stencil.Pix
	for i := range p {
		p[i] = 0
	}
}

func (b *SoftwareBackend) ClearClip() {
	p := b.clip.Pix
	for i := range p {
		p[i] = 255
	}
}

func (b *SoftwareBackend) Clip(pts []BackendVec) {
	b.clearStencil()

	iterateTriangles(pts[:], func(tri []BackendVec) {
		b.fillTriangleNoAA(tri, func(x, y int) {
			b.stencil.SetAlpha(x, y, color.Alpha{A: 255})
		})
	})

	p := b.clip.Pix
	p2 := b.stencil.Pix
	for i := range p {
		if p2[i] == 0 {
			p[i] = 0
		}
	}
}

func toRGBA(src color.Color) color.RGBA {
	ir, ig, ib, ia := src.RGBA()
	return color.RGBA{
		R: uint8(ir >> 8),
		G: uint8(ig >> 8),
		B: uint8(ib >> 8),
		A: uint8(ia >> 8),
	}
}

func mix(src, dest color.Color) color.RGBA {
	ir1, ig1, ib1, ia1 := src.RGBA()
	r1 := float64(ir1) / 65535.0
	g1 := float64(ig1) / 65535.0
	b1 := float64(ib1) / 65535.0
	a1 := float64(ia1) / 65535.0

	ir2, ig2, ib2, ia2 := dest.RGBA()
	r2 := float64(ir2) / 65535.0
	g2 := float64(ig2) / 65535.0
	b2 := float64(ib2) / 65535.0
	a2 := float64(ia2) / 65535.0

	r := (r1-r2)*a1 + r2
	g := (g1-g2)*a1 + g2
	b := (b1-b2)*a1 + b2
	a := math.Max((a1-a2)*a1+a2, a2)

	return color.RGBA{
		R: uint8(math.Round(r * 255.0)),
		G: uint8(math.Round(g * 255.0)),
		B: uint8(math.Round(b * 255.0)),
		A: uint8(math.Round(a * 255.0)),
	}
}

func alphaColor(col color.Color, alpha color.Alpha) color.RGBA {
	ir, ig, ib, _ := col.RGBA()
	a2 := float64(alpha.A) / 255.0
	r := float64(ir) * a2 / 65535.0
	g := float64(ig) * a2 / 65535.0
	b := float64(ib) * a2 / 65535.0
	return color.RGBA{
		R: uint8(r * 255.0),
		G: uint8(g * 255.0),
		B: uint8(b * 255.0),
		A: 255,
	}
}

func lerp(col1, col2 color.Color, ratio float64) color.RGBA {
	ir1, ig1, ib1, ia1 := col1.RGBA()
	r1 := float64(ir1) / 65535.0
	g1 := float64(ig1) / 65535.0
	b1 := float64(ib1) / 65535.0
	a1 := float64(ia1) / 65535.0

	ir2, ig2, ib2, ia2 := col2.RGBA()
	r2 := float64(ir2) / 65535.0
	g2 := float64(ig2) / 65535.0
	b2 := float64(ib2) / 65535.0
	a2 := float64(ia2) / 65535.0

	r := (r1-r2)*ratio + r2
	g := (g1-g2)*ratio + g2
	b := (b1-b2)*ratio + b2
	a := (a1-a2)*ratio + a2

	return color.RGBA{
		R: uint8(math.Round(r * 255.0)),
		G: uint8(math.Round(g * 255.0)),
		B: uint8(math.Round(b * 255.0)),
		A: uint8(math.Round(a * 255.0)),
	}
}
