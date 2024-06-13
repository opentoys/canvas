package canvas

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Backend is used by the canvas to actually do the final
// drawing. This enables the backend to be implemented by
// various methods (OpenGL, but also other APIs or software)
type Backend interface {
	Size() (int, int)

	LoadImage(img image.Image) (BackendImage, error)
	LoadImagePattern(data BackendImagePatternData) BackendImagePattern
	LoadLinearGradient(data BackendGradient) BackendLinearGradient
	LoadRadialGradient(data BackendGradient) BackendRadialGradient

	Clear(pts [4]BackendVec)
	Fill(style *BackendFillStyle, pts []BackendVec, tf BackendMat, canOverlap bool)
	DrawImage(dimg BackendImage, sx, sy, sw, sh float64, pts [4]BackendVec, alpha float64)
	FillImageMask(style *BackendFillStyle, mask *image.Alpha, pts [4]BackendVec) // pts must have four points

	ClearClip()
	Clip(pts []BackendVec)

	GetImageData(x, y, w, h int) *image.RGBA
	PutImageData(img *image.RGBA, x, y int)

	CanUseAsImage(b Backend) bool
	AsImage() BackendImage // can return nil if not supported
}

// FillStyle is the color and other details on how to fill
type BackendFillStyle struct {
	Color          color.RGBA
	Blur           float64
	LinearGradient BackendLinearGradient
	RadialGradient BackendRadialGradient
	Gradient       struct {
		X0, Y0  float64
		X1, Y1  float64
		RadFrom float64
		RadTo   float64
	}
	ImagePattern BackendImagePattern
}

type BackendGradient []BackendGradientStop

func (g BackendGradient) ColorAt(pos float64) color.RGBA {
	if len(g) == 0 {
		return color.RGBA{}
	} else if len(g) == 1 {
		return g[0].Color
	}
	beforeIdx, afterIdx := -1, -1
	for i, stop := range g {
		if stop.Pos > pos {
			afterIdx = i
			break
		}
		beforeIdx = i
	}
	if beforeIdx == -1 {
		return g[0].Color
	} else if afterIdx == -1 {
		return g[len(g)-1].Color
	}
	before, after := g[beforeIdx], g[afterIdx]
	p := (pos - before.Pos) / (after.Pos - before.Pos)
	var c [4]float64
	c[0] = (float64(after.Color.R)-float64(before.Color.R))*p + float64(before.Color.R)
	c[1] = (float64(after.Color.G)-float64(before.Color.G))*p + float64(before.Color.G)
	c[2] = (float64(after.Color.B)-float64(before.Color.B))*p + float64(before.Color.B)
	c[3] = (float64(after.Color.A)-float64(before.Color.A))*p + float64(before.Color.A)
	return color.RGBA{
		R: uint8(math.Round(c[0])),
		G: uint8(math.Round(c[1])),
		B: uint8(math.Round(c[2])),
		A: uint8(math.Round(c[3])),
	}
}

type BackendGradientStop struct {
	Pos   float64
	Color color.RGBA
}

type BackendLinearGradient interface {
	Delete()
	Replace(data BackendGradient)
}

type BackendRadialGradient interface {
	Delete()
	Replace(data BackendGradient)
}

type BackendImage interface {
	Width() int
	Height() int
	Size() (w, h int)
	Delete()
	Replace(src image.Image) error
}

type BackendImagePatternData struct {
	Image     BackendImage
	Transform [9]float64
	Repeat    BackendImagePatternRepeat
}

type BackendImagePatternRepeat uint8

// Image pattern repeat constants
const (
	BackendRepeat BackendImagePatternRepeat = iota
	BackendRepeatX
	BackendRepeatY
	BackendNoRepeat
)

type BackendImagePattern interface {
	Delete()
	Replace(data BackendImagePatternData)
}

type BackendVec [2]float64

func (v BackendVec) String() string {
	return fmt.Sprintf("[%f,%f]", v[0], v[1])
}

func (v BackendVec) Add(v2 BackendVec) BackendVec {
	return BackendVec{v[0] + v2[0], v[1] + v2[1]}
}

func (v BackendVec) Sub(v2 BackendVec) BackendVec {
	return BackendVec{v[0] - v2[0], v[1] - v2[1]}
}

func (v BackendVec) Mul(v2 BackendVec) BackendVec {
	return BackendVec{v[0] * v2[0], v[1] * v2[1]}
}

func (v BackendVec) Mulf(f float64) BackendVec {
	return BackendVec{v[0] * f, v[1] * f}
}

func (v BackendVec) MulMat(m BackendMat) BackendVec {
	return BackendVec{
		m[0]*v[0] + m[2]*v[1] + m[4],
		m[1]*v[0] + m[3]*v[1] + m[5]}
}

func (v BackendVec) MulMat2(m BackendMat2) BackendVec {
	return BackendVec{m[0]*v[0] + m[2]*v[1], m[1]*v[0] + m[3]*v[1]}
}

func (v BackendVec) Div(v2 BackendVec) BackendVec {
	return BackendVec{v[0] / v2[0], v[1] / v2[1]}
}

func (v BackendVec) Divf(f float64) BackendVec {
	return BackendVec{v[0] / f, v[1] / f}
}

func (v BackendVec) Dot(v2 BackendVec) float64 {
	return v[0]*v2[0] + v[1]*v2[1]
}

func (v BackendVec) Len() float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1])
}

func (v BackendVec) LenSqr() float64 {
	return v[0]*v[0] + v[1]*v[1]
}

func (v BackendVec) Norm() BackendVec {
	return v.Mulf(1.0 / v.Len())
}

func (v BackendVec) Atan2() float64 {
	return math.Atan2(v[1], v[0])
}

func (v BackendVec) Angle() float64 {
	return math.Pi*0.5 - math.Atan2(v[1], v[0])
}

func (v BackendVec) AngleTo(v2 BackendVec) float64 {
	return math.Acos(v.Norm().Dot(v2.Norm()))
}

type BackendMat [6]float64

func (m *BackendMat) String() string {
	return fmt.Sprintf("[%f,%f,0,\n %f,%f,0,\n %f,%f,1,]", m[0], m[2], m[4], m[1], m[3], m[5])
}

var BackendMatIdentity = BackendMat{
	1, 0,
	0, 1,
	0, 0}

func BackendMatTranslate(v BackendVec) BackendMat {
	return BackendMat{
		1, 0,
		0, 1,
		v[0], v[1]}
}

func BackendMatScale(v BackendVec) BackendMat {
	return BackendMat{
		v[0], 0,
		0, v[1],
		0, 0}
}

func BackendMatRotate(radians float64) BackendMat {
	s, c := math.Sincos(radians)
	return BackendMat{
		c, s,
		-s, c,
		0, 0}
}

func (m BackendMat) Mul(m2 BackendMat) BackendMat {
	return BackendMat{
		m[0]*m2[0] + m[1]*m2[2],
		m[0]*m2[1] + m[1]*m2[3],
		m[2]*m2[0] + m[3]*m2[2],
		m[2]*m2[1] + m[3]*m2[3],
		m[4]*m2[0] + m[5]*m2[2] + m2[4],
		m[4]*m2[1] + m[5]*m2[3] + m2[5]}
}

func (m BackendMat) Invert() BackendMat {
	identity := 1.0 / (m[0]*m[3] - m[2]*m[1])

	return BackendMat{
		m[3] * identity,
		-m[1] * identity,
		-m[2] * identity,
		m[0] * identity,
		(m[2]*m[5] - m[3]*m[4]) * identity,
		(m[1]*m[4] - m[0]*m[5]) * identity,
	}
}

type BackendMat2 [4]float64

func (m BackendMat) Mat2() BackendMat2 {
	return BackendMat2{m[0], m[1], m[2], m[3]}
}

func (m *BackendMat2) String() string {
	return fmt.Sprintf("[%f,%f,\n %f,%f]", m[0], m[2], m[1], m[3])
}
