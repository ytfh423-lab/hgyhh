package controller

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// ========== 市场波动图生成 ==========

const (
	chartWidth    = 900
	chartHeight   = 520
	chartPadLeft  = 70
	chartPadRight = 20
	chartPadTop   = 45
	chartPadBot   = 60
	chartLegendH  = 30
)

var chartCategoryColors = map[string][]color.RGBA{
	"crop": {
		{0x22, 0xc5, 0x5e, 0xff}, {0x16, 0xa3, 0x4a, 0xff}, {0x84, 0xcc, 0x16, 0xff},
		{0xea, 0xb3, 0x08, 0xff}, {0xf9, 0x73, 0x16, 0xff}, {0xa3, 0xe6, 0x35, 0xff},
		{0x0e, 0xa5, 0xe9, 0xff}, {0x64, 0x74, 0x8b, 0xff}, {0xd9, 0x77, 0x06, 0xff},
		{0x8b, 0x5c, 0xf6, 0xff}, {0xef, 0x44, 0x44, 0xff}, {0x06, 0xb6, 0xd4, 0xff},
		{0xf4, 0x3f, 0x5e, 0xff}, {0x14, 0xb8, 0xa6, 0xff}, {0xa8, 0x55, 0xf7, 0xff},
		{0x78, 0x71, 0x6c, 0xff}, {0xe1, 0x1d, 0x48, 0xff}, {0x0d, 0x96, 0x88, 0xff},
		{0xdb, 0x27, 0x77, 0xff}, {0x65, 0xa3, 0x0d, 0xff}, {0xc0, 0x26, 0xd3, 0xff},
		{0x25, 0x63, 0xeb, 0xff}, {0xdc, 0x26, 0x26, 0xff},
	},
	"fish": {
		{0x38, 0xbd, 0xf8, 0xff}, {0x06, 0xb6, 0xd4, 0xff}, {0xf9, 0x73, 0x16, 0xff},
		{0xef, 0x44, 0x44, 0xff}, {0xa8, 0x55, 0xf7, 0xff}, {0x22, 0xc5, 0x5e, 0xff},
		{0xea, 0xb3, 0x08, 0xff}, {0xe1, 0x1d, 0x48, 0xff},
	},
	"meat": {
		{0xef, 0x44, 0x44, 0xff}, {0xf9, 0x73, 0x16, 0xff}, {0xea, 0xb3, 0x08, 0xff},
		{0x22, 0xc5, 0x5e, 0xff}, {0x38, 0xbd, 0xf8, 0xff}, {0xa8, 0x55, 0xf7, 0xff},
	},
	"recipe": {
		{0xa8, 0x55, 0xf7, 0xff}, {0x8b, 0x5c, 0xf6, 0xff}, {0xdb, 0x27, 0x77, 0xff},
		{0xef, 0x44, 0x44, 0xff}, {0xf9, 0x73, 0x16, 0xff}, {0xea, 0xb3, 0x08, 0xff},
		{0x22, 0xc5, 0x5e, 0xff}, {0x06, 0xb6, 0xd4, 0xff}, {0x38, 0xbd, 0xf8, 0xff},
		{0x25, 0x63, 0xeb, 0xff}, {0x64, 0x74, 0x8b, 0xff}, {0xd9, 0x77, 0x06, 0xff},
		{0xe1, 0x1d, 0x48, 0xff}, {0x14, 0xb8, 0xa6, 0xff}, {0x84, 0xcc, 0x16, 0xff},
		{0xc0, 0x26, 0xd3, 0xff}, {0x0d, 0x96, 0x88, 0xff}, {0x78, 0x71, 0x6c, 0xff},
		{0x16, 0xa3, 0x4a, 0xff}, {0xa3, 0xe6, 0x35, 0xff}, {0xf4, 0x3f, 0x5e, 0xff},
		{0x65, 0xa3, 0x0d, 0xff},
	},
}

type chartItem struct {
	Key   string
	Label string
}

func getChartItems(category string) []chartItem {
	switch category {
	case "crop":
		var items []chartItem
		for _, c := range farmCrops {
			items = append(items, chartItem{"crop_" + c.Key, c.Emoji + c.Name})
		}
		return items
	case "fish":
		var items []chartItem
		for _, f := range fishTypes {
			items = append(items, chartItem{"fish_" + f.Key, f.Emoji + f.Name})
		}
		return items
	case "meat":
		var items []chartItem
		for _, a := range ranchAnimals {
			items = append(items, chartItem{"meat_" + a.Key, a.Emoji + a.Name})
		}
		return items
	case "recipe":
		var items []chartItem
		for _, r := range recipes {
			items = append(items, chartItem{"recipe_" + r.Key, r.Emoji + r.Name})
		}
		return items
	}
	return nil
}

func getCategoryTitle(category string) string {
	switch category {
	case "crop":
		return "🌾 作物市场波动"
	case "fish":
		return "🐟 鱼类市场波动"
	case "meat":
		return "🥩 肉类市场波动"
	case "recipe":
		return "🏭 加工品市场波动"
	}
	return "📈 市场波动"
}

// generateMarketChartPNG 生成指定类别的市场波动图
func generateMarketChartPNG(category string) ([]byte, error) {
	marketMu.RLock()
	history := make([]marketSnapshot, len(marketHistory))
	copy(history, marketHistory)
	marketMu.RUnlock()

	if len(history) < 2 {
		return nil, fmt.Errorf("数据不足，至少需要2个历史快照")
	}

	items := getChartItems(category)
	if len(items) == 0 {
		return nil, fmt.Errorf("未知类别")
	}

	colors := chartCategoryColors[category]
	if colors == nil {
		colors = chartCategoryColors["crop"]
	}

	img := image.NewRGBA(image.Rect(0, 0, chartWidth, chartHeight))

	// 背景
	bgColor := color.RGBA{0x1a, 0x1a, 0x2e, 0xff}
	fillRect(img, 0, 0, chartWidth, chartHeight, bgColor)

	// 图表区域
	plotL := chartPadLeft
	plotR := chartWidth - chartPadRight
	plotT := chartPadTop
	plotB := chartHeight - chartPadBot - chartLegendH
	plotW := plotR - plotL
	plotH := plotB - plotT

	// 绘制图表区域背景
	fillRect(img, plotL, plotT, plotR, plotB, color.RGBA{0x16, 0x16, 0x28, 0xff})

	// 计算Y轴范围
	minVal := 999
	maxVal := 0
	for _, snap := range history {
		for _, item := range items {
			if v, ok := snap.Prices[item.Key]; ok {
				if v < minVal {
					minVal = v
				}
				if v > maxVal {
					maxVal = v
				}
			}
		}
	}
	if minVal > maxVal {
		minVal, maxVal = 50, 200
	}
	yPad := (maxVal - minVal) / 5
	if yPad < 10 {
		yPad = 10
	}
	minVal -= yPad
	maxVal += yPad
	if minVal < 0 {
		minVal = 0
	}
	yRange := float64(maxVal - minVal)
	if yRange == 0 {
		yRange = 1
	}

	// 网格线 + Y轴标签
	gridColor := color.RGBA{0x33, 0x33, 0x55, 0xff}
	labelColor := color.RGBA{0x99, 0x99, 0xbb, 0xff}
	face := basicfont.Face7x13
	gridLines := 5
	for i := 0; i <= gridLines; i++ {
		yVal := minVal + int(float64(i)*yRange/float64(gridLines))
		yPx := plotB - int(float64(i)*float64(plotH)/float64(gridLines))
		drawHLine(img, plotL, plotR, yPx, gridColor)
		label := fmt.Sprintf("%d%%", yVal)
		drawString(img, face, plotL-len(label)*7-6, yPx+4, label, labelColor)
	}

	// 100%参考线（红色虚线）
	if minVal <= 100 && maxVal >= 100 {
		y100 := plotB - int(float64(100-minVal)/yRange*float64(plotH))
		for x := plotL; x < plotR; x += 6 {
			end := x + 3
			if end > plotR {
				end = plotR
			}
			drawHLine(img, x, end, y100, color.RGBA{0xff, 0x55, 0x55, 0x88})
		}
	}

	// X轴时间标签
	nSnaps := len(history)
	xStep := float64(plotW) / float64(nSnaps-1)
	for i, snap := range history {
		if i%(nSnaps/6+1) == 0 || i == nSnaps-1 {
			t := time.Unix(snap.Timestamp, 0)
			label := t.Format("01/02 15:04")
			xPx := plotL + int(float64(i)*xStep)
			drawString(img, face, xPx-len(label)*7/2, plotB+16, label, labelColor)
		}
	}

	// 绘制数据线
	for idx, item := range items {
		clr := colors[idx%len(colors)]
		var prevX, prevY int
		first := true
		for i, snap := range history {
			v, ok := snap.Prices[item.Key]
			if !ok {
				first = true
				continue
			}
			xPx := plotL + int(float64(i)*xStep)
			yPx := plotB - int(float64(v-minVal)/yRange*float64(plotH))
			if !first {
				drawLine(img, prevX, prevY, xPx, yPx, clr)
				drawLine(img, prevX, prevY+1, xPx, yPx+1, clr)
			}
			// 数据点
			fillCircle(img, xPx, yPx, 2, clr)
			first = false
			prevX = xPx
			prevY = yPx
		}
	}

	// 标题
	title := getCategoryTitle(category)
	titleColor := color.RGBA{0xff, 0xff, 0xff, 0xff}
	drawString(img, face, plotL, plotT-12, title, titleColor)

	// 图例（底部）
	legendY := chartHeight - chartLegendH + 5
	legendX := plotL
	maxLegendWidth := plotW
	currentX := legendX
	for idx, item := range items {
		clr := colors[idx%len(colors)]
		label := item.Label
		labelW := len(label)*7 + 20
		if currentX+labelW > legendX+maxLegendWidth {
			// 超出宽度不绘制更多图例
			drawString(img, face, currentX, legendY+10, "...", labelColor)
			break
		}
		fillRect(img, currentX, legendY+3, currentX+10, legendY+13, clr)
		drawString(img, face, currentX+13, legendY+12, label, labelColor)
		currentX += labelW
	}

	// 编码PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ========== 图形绘制工具函数 ==========

func fillRect(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func drawHLine(img *image.RGBA, x1, x2, y int, c color.RGBA) {
	for x := x1; x <= x2; x++ {
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
	sx, sy := 1, 1
	if x1 >= x2 {
		sx = -1
	}
	if y1 >= y2 {
		sy = -1
	}
	err := dx - dy
	for {
		if x1 >= 0 && x1 < img.Bounds().Dx() && y1 >= 0 && y1 < img.Bounds().Dy() {
			img.SetRGBA(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				px, py := cx+x, cy+y
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.SetRGBA(px, py, c)
				}
			}
		}
	}
}

func drawString(img *image.RGBA, face font.Face, x, y int, s string, c color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}
