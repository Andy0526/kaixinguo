//go:build darwin

// Package imageio decodes the pictures Go's image packages can't, such as
// HEIC covers, with macOS's ImageIO, called through purego (no cgo).
package imageio

import (
	"errors"
	"image"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

type cgRect struct{ X, Y, W, H float64 }

var (
	loadOnce sync.Once
	loadErr  error

	cfDataCreate                     func(alloc uintptr, bytes *byte, length int) uintptr
	cfRelease                        func(uintptr)
	cgImageSourceCreateWithData      func(data, opts uintptr) uintptr
	cgImageSourceCreateImageAtIndex  func(src uintptr, index int, opts uintptr) uintptr
	cgImageGetWidth                  func(uintptr) int
	cgImageGetHeight                 func(uintptr) int
	cgImageRelease                   func(uintptr)
	cgColorSpaceCreateWithName       func(name uintptr) uintptr
	cgColorSpaceRelease              func(uintptr)
	cgBitmapContextCreate            func(data unsafe.Pointer, w, h, bitsPerComponent, bytesPerRow int, space uintptr, info uint32) uintptr
	cgContextSetInterpolationQuality func(ctx uintptr, quality int32)
	cgContextDrawImage               func(ctx uintptr, r cgRect, img uintptr)
	cgContextRelease                 func(uintptr)
	srgbName                         uintptr
)

func load() {
	const flags = purego.RTLD_LAZY | purego.RTLD_GLOBAL
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", flags)
	if err != nil {
		loadErr = err
		return
	}
	io, err := purego.Dlopen("/System/Library/Frameworks/ImageIO.framework/ImageIO", flags)
	if err != nil {
		loadErr = err
		return
	}
	cg, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", flags)
	if err != nil {
		loadErr = err
		return
	}
	purego.RegisterLibFunc(&cfDataCreate, cf, "CFDataCreate")
	purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")
	purego.RegisterLibFunc(&cgImageSourceCreateWithData, io, "CGImageSourceCreateWithData")
	purego.RegisterLibFunc(&cgImageSourceCreateImageAtIndex, io, "CGImageSourceCreateImageAtIndex")
	purego.RegisterLibFunc(&cgImageGetWidth, cg, "CGImageGetWidth")
	purego.RegisterLibFunc(&cgImageGetHeight, cg, "CGImageGetHeight")
	purego.RegisterLibFunc(&cgImageRelease, cg, "CGImageRelease")
	purego.RegisterLibFunc(&cgColorSpaceCreateWithName, cg, "CGColorSpaceCreateWithName")
	purego.RegisterLibFunc(&cgColorSpaceRelease, cg, "CGColorSpaceRelease")
	purego.RegisterLibFunc(&cgBitmapContextCreate, cg, "CGBitmapContextCreate")
	purego.RegisterLibFunc(&cgContextSetInterpolationQuality, cg, "CGContextSetInterpolationQuality")
	purego.RegisterLibFunc(&cgContextDrawImage, cg, "CGContextDrawImage")
	purego.RegisterLibFunc(&cgContextRelease, cg, "CGContextRelease")
	sym, err := purego.Dlsym(cg, "kCGColorSpaceSRGB")
	if err != nil {
		loadErr = err
		return
	}
	srgbName = **(**uintptr)(unsafe.Pointer(&sym)) // the CFStringRef the symbol holds
}

// Decode decodes a picture of any format ImageIO reads into sRGB pixels,
// scaled down to at most maxWidth pixels wide when maxWidth > 0.
func Decode(data []byte, maxWidth int) (*image.RGBA, error) {
	if len(data) == 0 {
		return nil, errors.New("imageio: no data")
	}
	if loadOnce.Do(load); loadErr != nil {
		return nil, loadErr
	}
	cfData := cfDataCreate(0, &data[0], len(data))
	if cfData == 0 {
		return nil, errors.New("imageio: CFDataCreate failed")
	}
	defer cfRelease(cfData)
	src := cgImageSourceCreateWithData(cfData, 0)
	if src == 0 {
		return nil, errors.New("imageio: unknown format")
	}
	defer cfRelease(src)
	img := cgImageSourceCreateImageAtIndex(src, 0, 0)
	if img == 0 {
		return nil, errors.New("imageio: no image")
	}
	defer cgImageRelease(img)

	w, h := cgImageGetWidth(img), cgImageGetHeight(img)
	if w <= 0 || h <= 0 {
		return nil, errors.New("imageio: empty image")
	}
	if maxWidth > 0 && w > maxWidth {
		h = max(1, h*maxWidth/w)
		w = maxWidth
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	space := cgColorSpaceCreateWithName(srgbName)
	defer cgColorSpaceRelease(space)
	// Premultiplied RGBA in memory order, as image.RGBA keeps it.
	const premultipliedLast, byteOrder32Big = 1, 4 << 12
	ctx := cgBitmapContextCreate(unsafe.Pointer(&out.Pix[0]), w, h, 8, out.Stride, space, premultipliedLast|byteOrder32Big)
	if ctx == 0 {
		return nil, errors.New("imageio: CGBitmapContextCreate failed")
	}
	cgContextSetInterpolationQuality(ctx, 3) // high
	cgContextDrawImage(ctx, cgRect{W: float64(w), H: float64(h)}, img)
	cgContextRelease(ctx)
	return out, nil
}
