package fish

// Shapes are what a Species is drawn from. Coordinates are x in cells and y
// in half-rows, so one unit is about as wide as it is tall and circles come
// out round.

// Shape reports whether the point x, y is inside it.
type Shape func(x, y float64) bool

// Ellipse is centred on cx, cy with radii rx and ry.
func Ellipse(cx, cy, rx, ry float64) Shape {
	return func(x, y float64) bool {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		return dx*dx+dy*dy <= 1
	}
}

// Circle is centred on cx, cy with radius r.
func Circle(cx, cy, r float64) Shape { return Ellipse(cx, cy, r, r) }

// Rect runs from x0, y0 to x1, y1.
func Rect(x0, y0, x1, y1 float64) Shape {
	return func(x, y float64) bool { return x >= x0 && x <= x1 && y >= y0 && y <= y1 }
}

// Polygon runs through the points x0, y0, x1, y1, and so on.
func Polygon(pts ...float64) Shape {
	return func(x, y float64) bool {
		in := false
		n := len(pts) / 2
		for i, j := 0, n-1; i < n; j, i = i, i+1 {
			xi, yi, xj, yj := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
			if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				in = !in
			}
		}
		return in
	}
}

// Both is where all of ss overlap.
func Both(ss ...Shape) Shape {
	return func(x, y float64) bool {
		for _, s := range ss {
			if !s(x, y) {
				return false
			}
		}
		return true
	}
}

// Either is everywhere any of ss is.
func Either(ss ...Shape) Shape {
	return func(x, y float64) bool {
		for _, s := range ss {
			if s(x, y) {
				return true
			}
		}
		return false
	}
}

// Minus is s with cut taken out of it.
func Minus(s, cut Shape) Shape {
	return func(x, y float64) bool { return s(x, y) && !cut(x, y) }
}

// Layer is one shape of a drawing, in one colour. Later layers are drawn over
// earlier ones.
type Layer struct {
	Colour string // "#rrggbb", or "" to cut a hole that shows the background
	Shape  Shape
	Tail   bool // part of the tail, which swishes
}

// Fill is a layer of the body.
func Fill(colour string, s Shape) Layer { return Layer{Colour: colour, Shape: s} }

// TailFill is a layer of the tail.
func TailFill(colour string, s Shape) Layer { return Layer{Colour: colour, Shape: s, Tail: true} }
