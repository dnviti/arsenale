package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
)

func renderWorldTile(z int, x int, y int) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, tileSize, tileSize))
	drawGraticule(img, z, x, y)

	for _, polygon := range worldPolygons {
		points := make([]pixelPoint, 0, len(polygon))
		for _, coord := range polygon {
			projected := projectLatLon(coord[0], coord[1], z)
			points = append(points, pixelPoint{
				x: projected.x - float64(x*tileSize),
				y: projected.y - float64(y*tileSize),
			})
		}
		fillPolygon(img, points, landFill)
		drawPolygonStroke(img, points, landStroke)
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawGraticule(img *image.NRGBA, z int, x int, y int) {
	tileOffsetX := float64(x * tileSize)
	tileOffsetY := float64(y * tileSize)

	for _, latitude := range graticuleLatitudes {
		projected := projectLatLon(latitude, 0, z)
		drawHorizontalLine(img, projected.y-tileOffsetY, gridLine)
	}
	for _, longitude := range graticuleLongitudes {
		projected := projectLatLon(0, longitude, z)
		drawVerticalLine(img, projected.x-tileOffsetX, gridLine)
	}
}

func drawHorizontalLine(img *image.NRGBA, y float64, tone color.NRGBA) {
	row := int(math.Round(y))
	if row < 0 || row >= tileSize {
		return
	}
	for col := 0; col < tileSize; col++ {
		blendPixel(img, col, row, tone)
	}
}

func drawVerticalLine(img *image.NRGBA, x float64, tone color.NRGBA) {
	col := int(math.Round(x))
	if col < 0 || col >= tileSize {
		return
	}
	for row := 0; row < tileSize; row++ {
		blendPixel(img, col, row, tone)
	}
}

func drawPolygonStroke(img *image.NRGBA, points []pixelPoint, tone color.NRGBA) {
	if len(points) < 2 {
		return
	}
	for index := range points {
		start := points[index]
		end := points[(index+1)%len(points)]
		clippedStart, clippedEnd, ok := clipLine(start, end)
		if !ok {
			continue
		}
		drawLine(img, clippedStart, clippedEnd, tone)
	}
}

func drawLine(img *image.NRGBA, start pixelPoint, end pixelPoint, tone color.NRGBA) {
	deltaX := end.x - start.x
	deltaY := end.y - start.y
	steps := int(math.Ceil(math.Max(math.Abs(deltaX), math.Abs(deltaY))))
	if steps == 0 {
		blendPixel(img, int(math.Round(start.x)), int(math.Round(start.y)), tone)
		return
	}
	for step := 0; step <= steps; step++ {
		progress := float64(step) / float64(steps)
		x := start.x + deltaX*progress
		y := start.y + deltaY*progress
		blendPixel(img, int(math.Round(x)), int(math.Round(y)), tone)
	}
}

func fillPolygon(img *image.NRGBA, points []pixelPoint, tone color.NRGBA) {
	if len(points) < 3 {
		return
	}
	intersections := make([]float64, 0, len(points))
	for row := 0; row < tileSize; row++ {
		scanY := float64(row) + 0.5
		intersections = intersections[:0]
		for index := range points {
			start := points[index]
			end := points[(index+1)%len(points)]
			if crossesScanline(start.y, end.y, scanY) {
				progress := (scanY - start.y) / (end.y - start.y)
				intersections = append(intersections, start.x+progress*(end.x-start.x))
			}
		}
		if len(intersections) < 2 {
			continue
		}
		sort.Float64s(intersections)
		for index := 0; index+1 < len(intersections); index += 2 {
			startX := int(math.Ceil(intersections[index]))
			endX := int(math.Floor(intersections[index+1]))
			if endX < 0 || startX >= tileSize {
				continue
			}
			if startX < 0 {
				startX = 0
			}
			if endX >= tileSize {
				endX = tileSize - 1
			}
			for col := startX; col <= endX; col++ {
				blendPixel(img, col, row, tone)
			}
		}
	}
}
