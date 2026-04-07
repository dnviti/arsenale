package main

import (
	"image"
	"image/color"
	"math"
)

func crossesScanline(startY float64, endY float64, scanY float64) bool {
	return (startY <= scanY && endY > scanY) || (endY <= scanY && startY > scanY)
}

func projectLatLon(latitude float64, longitude float64, zoom int) pixelPoint {
	clampedLat := math.Max(math.Min(latitude, maxMercatorLat), -maxMercatorLat)
	latRadians := clampedLat * math.Pi / 180
	scale := float64(tileSize) * math.Exp2(float64(zoom))
	return pixelPoint{
		x: ((longitude + 180) / 360) * scale,
		y: (1 - math.Log(math.Tan(latRadians)+1/math.Cos(latRadians))/math.Pi) * scale / 2,
	}
}

func clipLine(start pixelPoint, end pixelPoint) (pixelPoint, pixelPoint, bool) {
	const (
		leftCode   = 1
		rightCode  = 2
		topCode    = 4
		bottomCode = 8
	)
	const (
		minCoord = -1.0
		maxCoord = float64(tileSize)
	)

	outCode := func(point pixelPoint) int {
		code := 0
		if point.x < minCoord {
			code |= leftCode
		} else if point.x > maxCoord {
			code |= rightCode
		}
		if point.y < minCoord {
			code |= topCode
		} else if point.y > maxCoord {
			code |= bottomCode
		}
		return code
	}

	codeA := outCode(start)
	codeB := outCode(end)

	for {
		if codeA|codeB == 0 {
			return start, end, true
		}
		if codeA&codeB != 0 {
			return pixelPoint{}, pixelPoint{}, false
		}

		codeOut := codeA
		if codeOut == 0 {
			codeOut = codeB
		}

		next := pixelPoint{}
		switch {
		case codeOut&topCode != 0:
			next.y = minCoord
			next.x = start.x + (end.x-start.x)*(minCoord-start.y)/(end.y-start.y)
		case codeOut&bottomCode != 0:
			next.y = maxCoord
			next.x = start.x + (end.x-start.x)*(maxCoord-start.y)/(end.y-start.y)
		case codeOut&rightCode != 0:
			next.x = maxCoord
			next.y = start.y + (end.y-start.y)*(maxCoord-start.x)/(end.x-start.x)
		default:
			next.x = minCoord
			next.y = start.y + (end.y-start.y)*(minCoord-start.x)/(end.x-start.x)
		}

		if codeOut == codeA {
			start = next
			codeA = outCode(start)
			continue
		}
		end = next
		codeB = outCode(end)
	}
}

func blendPixel(img *image.NRGBA, x int, y int, tone color.NRGBA) {
	if x < 0 || x >= tileSize || y < 0 || y >= tileSize {
		return
	}
	offset := img.PixOffset(x, y)
	dstR := float64(img.Pix[offset+0])
	dstG := float64(img.Pix[offset+1])
	dstB := float64(img.Pix[offset+2])
	dstA := float64(img.Pix[offset+3]) / 255

	srcA := float64(tone.A) / 255
	outA := srcA + dstA*(1-srcA)
	if outA <= 0 {
		img.Pix[offset+0] = 0
		img.Pix[offset+1] = 0
		img.Pix[offset+2] = 0
		img.Pix[offset+3] = 0
		return
	}

	srcR := float64(tone.R)
	srcG := float64(tone.G)
	srcB := float64(tone.B)
	outR := (srcR*srcA + dstR*dstA*(1-srcA)) / outA
	outG := (srcG*srcA + dstG*dstA*(1-srcA)) / outA
	outB := (srcB*srcA + dstB*dstA*(1-srcA)) / outA

	img.Pix[offset+0] = uint8(math.Round(outR))
	img.Pix[offset+1] = uint8(math.Round(outG))
	img.Pix[offset+2] = uint8(math.Round(outB))
	img.Pix[offset+3] = uint8(math.Round(outA * 255))
}
