package main

import (
	"fmt"
	"math"
	"time"
)

const (
	screenWidth  = 80
	screenHeight = 30
	fov          = 60.0 * (math.Pi / 180.0)
	zCamera      = 4.0
	scaleModel   = 1.0
)

type Vec3 struct {
	x, y, z float64
}

func rotateX(p Vec3, angle float64) Vec3 {
	s, c := math.Sin(angle), math.Cos(angle)
	return Vec3{
		x: p.x,
		y: p.y*c - p.z*s,
		z: p.y*s + p.z*c,
	}
}

func rotateY(p Vec3, angle float64) Vec3 {
	s, c := math.Sin(angle), math.Cos(angle)
	return Vec3{
		x: p.x*c + p.z*s,
		y: p.y,
		z: -p.x*s + p.z*c,
	}
}

func rotateZ(p Vec3, angle float64) Vec3 {
	s, c := math.Sin(angle), math.Cos(angle)
	return Vec3{
		x: p.x*c - p.y*s,
		y: p.x*s + p.y*c,
		z: p.z,
	}
}

func project(p Vec3) (int, int, bool) {
	z := p.z + zCamera
	if z <= 0.1 {
		return 0, 0, false
	}
	scale := 1.0 / math.Tan(fov/2.0)
	x := (p.x * scale / z)
	y := (p.y * scale / z)

	sx := int((x+1.0)*0.5*float64(screenWidth-1) + 0.5)
	sy := int((1.0-(y+1.0)*0.5)*float64(screenHeight-1) + 0.5)

	if sx < 0 || sx >= screenWidth || sy < 0 || sy >= screenHeight {
		return 0, 0, false
	}
	return sx, sy, true
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func makeBuffer() [][]rune {
	buf := make([][]rune, screenHeight)
	for i := range buf {
		row := make([]rune, screenWidth)
		for j := range row {
			row[j] = ' '
		}
		buf[i] = row
	}
	return buf
}

func drawBuffer(buf [][]rune) {
	clearScreen()
	for _, row := range buf {
		fmt.Println(string(row))
	}
}

func putPixel(buf [][]rune, x, y int, ch rune) {
	if x >= 0 && x < screenWidth && y >= 0 && y < screenHeight {
		buf[y][x] = ch
	}
}

func drawLine(buf [][]rune, x0, y0, x1, y1 int, ch rune) {
	dx := int(math.Abs(float64(x1 - x0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -int(math.Abs(float64(y1 - y0)))
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy

	for {
		putPixel(buf, x0, y0, ch)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func main() {
	phi := (1.0 + math.Sqrt(5)) / 2.0 // золотое сечение φ[web:53][web:62]

	// 12 вершин правильного икосаэдра с рёбрами одинаковой длины, центр в (0,0,0).[web:53][web:62][web:61]
	rawVertices := []Vec3{
		{0, -1, -phi}, {0, -1, phi}, {0, 1, -phi}, {0, 1, phi},
		{-1, -phi, 0}, {-1, phi, 0}, {1, -phi, 0}, {1, phi, 0},
		{-phi, 0, -1}, {phi, 0, -1}, {-phi, 0, 1}, {phi, 0, 1},
	}

	// Нормализация размеров (просто масштабируем).[web:53][web:62]
	vertices := make([]Vec3, len(rawVertices))
	for i, v := range rawVertices {
		vertices[i] = Vec3{v.x * scaleModel, v.y * scaleModel, v.z * scaleModel}
	}

	// 20 треугольных граней по индексам вершин; каждое ребро — общая сторона двух граней.[web:61][web:67]
	faces := [][3]int{
		{0, 1, 4}, {0, 4, 8}, {0, 8, 2}, {0, 2, 9}, {0, 9, 6},
		{1, 10, 4}, {4, 10, 5}, {4, 5, 8}, {8, 5, 3}, {8, 3, 2},
		{2, 3, 7}, {2, 7, 9}, {9, 7, 11}, {9, 11, 6}, {6, 11, 1},
		{1, 11, 10}, {10, 11, 7}, {10, 7, 5}, {5, 7, 3}, {6, 1, 0},
	}

	// Строим список рёбер из граней (уникальные пары).
	edgeMap := map[[2]int]struct{}{}
	for _, f := range faces {
		idx := [3]int{f[0], f[1], f[2]}
		for i := 0; i < 3; i++ {
			a := idx[i]
			b := idx[(i+1)%3]
			if a > b {
				a, b = b, a
			}
			edgeMap[[2]int{a, b}] = struct{}{}
		}
	}
	var edges [][2]int
	for e := range edgeMap {
		edges = append(edges, e)
	}

	// Центры граней (для вывода цифр 1..20 — приблизительно посередине).[web:59][web:60]
	faceCenters := make([]Vec3, len(faces))
	for i, f := range faces {
		a := vertices[f[0]]
		b := vertices[f[1]]
		c := vertices[f[2]]
		faceCenters[i] = Vec3{
			(a.x + b.x + c.x) / 3.0,
			(a.y + b.y + c.y) / 3.0,
			(a.z + b.z + c.z) / 3.0,
		}
	}

	var angle float64

	for {
		buf := makeBuffer()

		// вращённые и проецированные вершины
		projVerts := make([]struct {
			x, y int
			ok   bool
		}, len(vertices))
		for i, v := range vertices {
			p := v
			p = rotateX(p, angle*0.8)
			p = rotateY(p, angle*1.1)
			p = rotateZ(p, angle*0.6)

			x, y, ok := project(p)
			projVerts[i] = struct {
				x, y int
				ok   bool
			}{x, y, ok}
		}

		// рёбра
		for _, e := range edges {
			a, b := projVerts[e[0]], projVerts[e[1]]
			if !a.ok || !b.ok {
				continue
			}
			drawLine(buf, a.x, a.y, b.x, b.y, '#')
		}

		// числа граней 1..20 по центрам
		for i, c := range faceCenters {
			p := c
			p = rotateX(p, angle*0.8)
			p = rotateY(p, angle*1.1)
			p = rotateZ(p, angle*0.6)

			x, y, ok := project(p)
			if !ok {
				continue
			}
			label := i + 1 // номер грани (1..20)
			ch := rune('0' + (label % 10))
			putPixel(buf, x, y, ch)
		}

		drawBuffer(buf)

		angle += 0.05
		time.Sleep(30 * time.Millisecond)
	}
}
