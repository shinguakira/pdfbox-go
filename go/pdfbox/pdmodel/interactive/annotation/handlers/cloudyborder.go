package handlers

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The angles the cloudy curls are built from.
//
// Port of the private constants of CloudyBorder.
const (
	angle180Deg = math.Pi
	angle90Deg  = math.Pi / 2
	angle34Deg  = 34 * math.Pi / 180
	angle30Deg  = 30 * math.Pi / 180
	angle12Deg  = 12 * math.Pi / 180
)

// point is one point of a cloudy path.
//
// Java uses java.awt.geom.Point2D.Double; the port uses a plain pair, since
// nothing here needs the interface.
type point struct {
	x, y float64
}

// distance returns how far this point is from the other.
func (p point) distance(other point) float64 {
	return geom.Distance(p.x, p.y, other.x, other.y)
}

// cloudyBorder draws the border of an annotation whose /BE style is cloudy.
//
// Port of the package-private class CloudyBorder.
type cloudyBorder struct {
	output    annotation.AppearanceContentStream
	annotRect *common.PDRectangle
	intensity float64
	lineWidth float64

	rectWithDiff  *common.PDRectangle
	outputStarted bool
	bboxMinX      float64
	bboxMinY      float64
	bboxMaxX      float64
	bboxMaxY      float64
}

// newCloudyBorder returns a border that draws into the given content stream.
func newCloudyBorder(stream annotation.AppearanceContentStream, intensity, lineWidth float64,
	rect *common.PDRectangle) *cloudyBorder {
	return &cloudyBorder{
		output:    stream,
		intensity: intensity,
		lineWidth: lineWidth,
		annotRect: rect,
	}
}

// createCloudyRectangle draws a cloudy rectangle, inset by the given
// differences.
func (b *cloudyBorder) createCloudyRectangle(rd *common.PDRectangle) error {
	b.rectWithDiff = b.applyRectDiff(rd, float32(b.lineWidth/2))
	left := float64(b.rectWithDiff.LowerLeftX())
	bottom := float64(b.rectWithDiff.LowerLeftY())
	right := float64(b.rectWithDiff.UpperRightX())
	top := float64(b.rectWithDiff.UpperRightY())

	if err := b.cloudyRectangleImpl(left, bottom, right, top, false); err != nil {
		return err
	}
	return b.finish()
}

// createCloudyPolygon draws a cloudy polygon along the given path.
func (b *cloudyBorder) createCloudyPolygon(path [][]float32) error {
	polygon := make([]point, len(path))
	for i, array := range path {
		switch len(array) {
		case 2:
			polygon[i] = point{float64(array[0]), float64(array[1])}
		case 6:
			// TODO Curve segments are not yet supported in cloudy border.
			polygon[i] = point{float64(array[4]), float64(array[5])}
		}
	}
	if err := b.cloudyPolygonImpl(polygon, false); err != nil {
		return err
	}
	return b.finish()
}

// createCloudyEllipse draws a cloudy ellipse, inset by the given differences.
func (b *cloudyBorder) createCloudyEllipse(rd *common.PDRectangle) error {
	b.rectWithDiff = b.applyRectDiff(rd, 0)
	left := float64(b.rectWithDiff.LowerLeftX())
	bottom := float64(b.rectWithDiff.LowerLeftY())
	right := float64(b.rectWithDiff.UpperRightX())
	top := float64(b.rectWithDiff.UpperRightY())

	if err := b.cloudyEllipseImpl(left, bottom, right, top); err != nil {
		return err
	}
	return b.finish()
}

// bbox returns the bounding box of what was drawn.
func (b *cloudyBorder) bbox() *common.PDRectangle { return b.rectangle() }

// rectangle returns the bounding box of what was drawn.
func (b *cloudyBorder) rectangle() *common.PDRectangle {
	return common.NewPDRectangleOf(float32(b.bboxMinX), float32(b.bboxMinY),
		float32(b.bboxMaxX-b.bboxMinX), float32(b.bboxMaxY-b.bboxMinY))
}

// matrix returns the matrix that moves the bounding box to the origin.
func (b *cloudyBorder) matrix() *util.Matrix {
	return util.NewMatrixOf(1, 0, 0, 1, float32(-b.bboxMinX), float32(-b.bboxMinY))
}

// rectDifference returns the difference between the annotation rectangle and
// the bounding box.
func (b *cloudyBorder) rectDifference() *common.PDRectangle {
	if b.annotRect == nil {
		d := float32(b.lineWidth / 2)
		return common.NewPDRectangleOf(d, d, float32(b.lineWidth), float32(b.lineWidth))
	}
	re := b.rectWithDiff
	if re == nil {
		re = b.annotRect
	}
	left := re.LowerLeftX() - float32(b.bboxMinX)
	bottom := re.LowerLeftY() - float32(b.bboxMinY)
	right := float32(b.bboxMaxX) - re.UpperRightX()
	top := float32(b.bboxMaxY) - re.UpperRightY()
	return common.NewPDRectangleOf(left, bottom, right-left, top-bottom)
}

// cosine returns dx over the hypotenuse, and zero where that is zero.
func cosine(dx, hypot float64) float64 {
	if hypot == 0 {
		return 0
	}
	return dx / hypot
}

// sine returns dy over the hypotenuse, and zero where that is zero.
func sine(dy, hypot float64) float64 {
	if hypot == 0 {
		return 0
	}
	return dy / hypot
}

// cloudyRectangleImpl draws the rectangle as a cloudy polygon.
func (b *cloudyBorder) cloudyRectangleImpl(left, bottom, right, top float64,
	isEllipse bool) error {
	w := right - left
	h := top - bottom

	if b.intensity <= 0.0 {
		if err := b.output.AddRect(float32(left), float32(bottom),
			float32(w), float32(h)); err != nil {
			return err
		}
		b.bboxMinX = left
		b.bboxMinY = bottom
		b.bboxMaxX = right
		b.bboxMaxY = top
		return nil
	}

	// Make a polygon with direction equal to the positive angle direction.
	var polygon []point
	switch {
	case w < 1.0:
		polygon = []point{{left, bottom}, {left, top}, {left, bottom}}
	case h < 1.0:
		polygon = []point{{left, bottom}, {right, bottom}, {left, bottom}}
	default:
		polygon = []point{
			{left, bottom}, {right, bottom}, {right, top}, {left, top}, {left, bottom},
		}
	}
	return b.cloudyPolygonImpl(polygon, isEllipse)
}

// cloudyPolygonImpl draws the curls along the given polygon.
func (b *cloudyBorder) cloudyPolygonImpl(vertices []point, isEllipse bool) error {
	polygon := removeZeroLengthSegments(vertices)
	getPositivePolygon(polygon)
	numPoints := len(polygon)
	if numPoints < 2 {
		return nil
	}
	if b.intensity <= 0.0 {
		if err := b.moveToPoint(polygon[0]); err != nil {
			return err
		}
		for i := 1; i < numPoints; i++ {
			if err := b.lineToPoint(polygon[i]); err != nil {
				return err
			}
		}
		return nil
	}

	cloudRadius := b.polygonCloudRadius()
	if isEllipse {
		cloudRadius = b.ellipseCloudRadius()
	}
	if cloudRadius < 0.5 {
		cloudRadius = 0.5
	}
	k := math.Cos(angle34Deg)
	advIntermDefault := 2 * k * cloudRadius
	advCornerDefault := k * cloudRadius
	array := make([]float64, 2)
	anglePrev := 0.0

	// The number of curls per polygon segment is hardly ever an integer,
	// so the length of some curls must be adjustable. We adjust the angle
	// of the trailing arc of corner curls and the leading arc of the first
	// intermediate curl.
	// In each polygon segment, we have n intermediate curls plus one half of a
	// corner curl at each end. One of the n intermediate curls is adjustable.
	// Thus the number of fixed (or unadjusted) intermediate curls is n - 1.

	// Find the adjusted angle `alpha` for the first corner curl.
	n0 := computeParamsPolygon(advIntermDefault, advCornerDefault, k, cloudRadius,
		polygon[numPoints-2].distance(polygon[0]), array)
	alphaPrev := angle34Deg
	if n0 == 0 {
		alphaPrev = array[0]
	}

	for j := 0; j+1 < numPoints; j++ {
		pt := polygon[j]
		ptNext := polygon[j+1]
		length := pt.distance(ptNext)
		if length == 0 {
			alphaPrev = angle34Deg
			continue
		}
		// n is the number of intermediate curls in the current polygon segment.
		n := computeParamsPolygon(advIntermDefault, advCornerDefault, k,
			cloudRadius, length, array)
		if n < 0 {
			if !b.outputStarted {
				if err := b.moveToPoint(pt); err != nil {
					return err
				}
			}
			continue
		}
		alpha := array[0]
		dx := array[1]
		angleCur := math.Atan2(ptNext.y-pt.y, ptNext.x-pt.x)
		if j == 0 {
			ptPrev := polygon[numPoints-2]
			anglePrev = math.Atan2(pt.y-ptPrev.y, pt.x-ptPrev.x)
		}
		cos := cosine(ptNext.x-pt.x, length)
		sin := sine(ptNext.y-pt.y, length)
		x := pt.x
		y := pt.y

		if err := b.addCornerCurl(anglePrev, angleCur, cloudRadius, pt.x, pt.y, alpha,
			alphaPrev, !b.outputStarted); err != nil {
			return err
		}

		// Proceed to the center point of the first intermediate curl.
		adv := 2*k*cloudRadius + 2*dx
		x += adv * cos
		y += adv * sin

		// Create the first intermediate curl.
		numInterm := n
		if n >= 1 {
			if err := b.addFirstIntermediateCurl(angleCur, cloudRadius, alpha, x, y); err != nil {
				return err
			}
			x += advIntermDefault * cos
			y += advIntermDefault * sin
			numInterm = n - 1
		}

		// Create one intermediate curl and replicate it along the polygon segment.
		template, err := b.intermediateCurlTemplate(angleCur, cloudRadius)
		if err != nil {
			return err
		}
		for i := 0; i < numInterm; i++ {
			if err := b.outputCurlTemplate(template, x, y); err != nil {
				return err
			}
			x += advIntermDefault * cos
			y += advIntermDefault * sin
		}

		anglePrev = angleCur
		alphaPrev = angle34Deg
		if n == 0 {
			alphaPrev = alpha
		}
	}
	return nil
}

// computeParamsPolygon works out the curl count and the adjustment angle of one
// polygon segment.
func computeParamsPolygon(advInterm, advCorner, k, r, length float64, array []float64) int {
	if length == 0 {
		array[0] = angle34Deg
		array[1] = 0
		return -1
	}

	// n is the number of intermediate curls in the current polygon segment
	n := int(math.Ceil((length - 2*advCorner) / advInterm))

	// Fitting error along polygon segment
	e := length - (2*advCorner + float64(n)*advInterm)

	// Fitting error per each adjustable half curl
	dx := e / 2

	// Convert fitting error to an angle that can be used to control arcs.
	arg := (k*r + dx) / r
	alpha := 0.0
	if arg >= -1.0 && arg <= 1.0 {
		alpha = math.Acos(arg)
	}

	array[0] = alpha
	array[1] = dx
	return n
}

// addCornerCurl draws the curl at a corner of the polygon.
func (b *cloudyBorder) addCornerCurl(anglePrev, angleCur, radius, cx, cy, alpha,
	alphaPrev float64, addMoveTo bool) error {
	a := anglePrev + angle180Deg + alphaPrev
	bAngle := anglePrev + angle180Deg + alphaPrev - 22*math.Pi/180
	if err := b.getArcSegment(a, bAngle, cx, cy, radius, radius, nil, addMoveTo); err != nil {
		return err
	}
	a = bAngle
	bAngle = angleCur - alpha
	return b.getArc(a, bAngle, radius, radius, cx, cy, nil, false)
}

// addFirstIntermediateCurl draws the adjustable curl after a corner.
func (b *cloudyBorder) addFirstIntermediateCurl(angleCur, r, alpha, cx, cy float64) error {
	a := angleCur + angle180Deg
	if err := b.getArcSegment(a+alpha, a+alpha-angle30Deg, cx, cy, r, r, nil, false); err != nil {
		return err
	}
	if err := b.getArcSegment(a+alpha-angle30Deg, a+angle90Deg, cx, cy, r, r, nil, false); err != nil {
		return err
	}
	return b.getArcSegment(a+angle90Deg, a+angle180Deg-angle34Deg, cx, cy, r, r, nil, false)
}

// intermediateCurlTemplate builds the points of one repeatable curl, centred on
// the origin.
func (b *cloudyBorder) intermediateCurlTemplate(angleCur, r float64) ([]point, error) {
	points := []point{}
	a := angleCur + angle180Deg
	if err := b.getArcSegment(a+angle34Deg, a+angle12Deg, 0, 0, r, r, &points, false); err != nil {
		return nil, err
	}
	if err := b.getArcSegment(a+angle12Deg, a+angle90Deg, 0, 0, r, r, &points, false); err != nil {
		return nil, err
	}
	if err := b.getArcSegment(a+angle90Deg, a+angle180Deg-angle34Deg,
		0, 0, r, r, &points, false); err != nil {
		return nil, err
	}
	return points, nil
}

// outputCurlTemplate writes one curl of the template at the given centre.
func (b *cloudyBorder) outputCurlTemplate(template []point, x, y float64) error {
	n := len(template)
	i := 0
	if n%3 == 1 {
		a := template[0]
		if err := b.moveTo(a.x+x, a.y+y); err != nil {
			return err
		}
		i++
	}
	for ; i+2 < n; i += 3 {
		a := template[i]
		bp := template[i+1]
		c := template[i+2]
		if err := b.curveTo(a.x+x, a.y+y, bp.x+x, bp.y+y, c.x+x, c.y+y); err != nil {
			return err
		}
	}
	return nil
}

// applyRectDiff insets the annotation rectangle by the given differences, each
// at least min.
func (b *cloudyBorder) applyRectDiff(rd *common.PDRectangle, min float32) *common.PDRectangle {
	rectLeft := b.annotRect.LowerLeftX()
	rectBottom := b.annotRect.LowerLeftY()
	rectRight := b.annotRect.UpperRightX()
	rectTop := b.annotRect.UpperRightY()

	// Normalize
	rectLeft = minFloat32(rectLeft, rectRight)
	rectBottom = minFloat32(rectBottom, rectTop)
	rectRight = maxFloat32(rectLeft, rectRight)
	rectTop = maxFloat32(rectBottom, rectTop)

	rdLeft := min
	rdBottom := min
	rdRight := min
	rdTop := min
	if rd != nil {
		rdLeft = maxFloat32(rd.LowerLeftX(), min)
		rdBottom = maxFloat32(rd.LowerLeftY(), min)
		rdRight = maxFloat32(rd.UpperRightX(), min)
		rdTop = maxFloat32(rd.UpperRightY(), min)
	}

	rectLeft += rdLeft
	rectBottom += rdBottom
	rectRight -= rdRight
	rectTop -= rdTop
	return common.NewPDRectangleOf(rectLeft, rectBottom, rectRight-rectLeft, rectTop-rectBottom)
}

// reversePolygon turns the polygon around, in place.
func reversePolygon(points []point) {
	length := len(points)
	n := length / 2
	for i := 0; i < n; i++ {
		j := length - i - 1
		points[i], points[j] = points[j], points[i]
	}
}

// getPositivePolygon turns the polygon so that it runs anticlockwise.
func getPositivePolygon(points []point) {
	if polygonDirection(points) < 0 {
		reversePolygon(points)
	}
}

// polygonDirection returns twice the signed area of the polygon, which is
// positive for an anticlockwise one.
func polygonDirection(points []point) float64 {
	a := 0.0
	length := len(points)
	for i := 0; i < length; i++ {
		j := (i + 1) % length
		a += points[i].x*points[j].y - points[i].y*points[j].x
	}
	return a
}

// getArc draws an arc as a run of quarter-circle segments, into out where it is
// given and into the content stream otherwise.
func (b *cloudyBorder) getArc(startAng, endAng, rx, ry, cx, cy float64,
	out *[]point, addMoveTo bool) error {
	angleIncr := math.Pi / 2
	startx := rx*math.Cos(startAng) + cx
	starty := ry*math.Sin(startAng) + cy

	angleTodo := endAng - startAng
	for angleTodo < 0 {
		angleTodo += 2 * math.Pi
	}
	sweep := angleTodo
	angleDone := 0.0

	if addMoveTo {
		if out != nil {
			*out = append(*out, point{startx, starty})
		} else if err := b.moveTo(startx, starty); err != nil {
			return err
		}
	}

	for angleTodo > angleIncr {
		if err := b.getArcSegment(startAng+angleDone,
			startAng+angleDone+angleIncr, cx, cy, rx, ry, out, false); err != nil {
			return err
		}
		angleDone += angleIncr
		angleTodo -= angleIncr
	}
	if angleTodo > 0 {
		return b.getArcSegment(startAng+angleDone, startAng+sweep, cx, cy, rx, ry, out, false)
	}
	return nil
}

// getArcSegment draws one arc of at most a quarter circle as a cubic.
func (b *cloudyBorder) getArcSegment(startAng, endAng, cx, cy, rx, ry float64,
	out *[]point, addMoveTo bool) error {
	// Algorithm is from the FAQ of the news group comp.text.pdf
	cosA := math.Cos(startAng)
	sinA := math.Sin(startAng)
	cosB := math.Cos(endAng)
	sinB := math.Sin(endAng)
	denom := math.Sin((endAng - startAng) / 2.0)

	if denom == 0 {
		// This can happen only if endAng == startAng.
		// The arc sweep angle is zero, so we create no arc at all.
		if addMoveTo {
			xs := cx + rx*cosA
			ys := cy + ry*sinA
			if out != nil {
				*out = append(*out, point{xs, ys})
			} else if err := b.moveTo(xs, ys); err != nil {
				return err
			}
		}
		return nil
	}

	bcp := 1.333333333 * (1 - math.Cos((endAng-startAng)/2.0)) / denom
	p1x := cx + rx*(cosA-bcp*sinA)
	p1y := cy + ry*(sinA+bcp*cosA)
	p2x := cx + rx*(cosB+bcp*sinB)
	p2y := cy + ry*(sinB-bcp*cosB)
	p3x := cx + rx*cosB
	p3y := cy + ry*sinB

	if addMoveTo {
		xs := cx + rx*cosA
		ys := cy + ry*sinA
		if out != nil {
			*out = append(*out, point{xs, ys})
		} else if err := b.moveTo(xs, ys); err != nil {
			return err
		}
	}

	if out != nil {
		*out = append(*out, point{p1x, p1y}, point{p2x, p2y}, point{p3x, p3y})
		return nil
	}
	return b.curveTo(p1x, p1y, p2x, p2y, p3x, p3y)
}

// flattenEllipse breaks an ellipse into a polygon.
func flattenEllipse(left, bottom, right, top float64) []point {
	ellipse := geom.NewEllipse2D(left, bottom, right-left, top-bottom)
	const flatness = 0.50
	iterator := ellipse.FlatteningPathIterator(nil, flatness)
	coords := make([]float64, 6)
	points := []point{}

	for !iterator.IsDone() {
		switch iterator.CurrentSegment(coords) {
		case geom.SegMoveTo, geom.SegLineTo:
			points = append(points, point{coords[0], coords[1]})
		default:
			// Curve segments are not expected because the path iterator is
			// flattened. SEG_CLOSE can be ignored.
		}
		iterator.Next()
	}

	size := len(points)
	const closeTestLimit = 0.05
	if size >= 2 && points[size-1].distance(points[0]) > closeTestLimit {
		points = append(points, points[len(points)-1])
	}
	return points
}

// cloudyEllipseImpl draws the curls along an ellipse.
func (b *cloudyBorder) cloudyEllipseImpl(leftOrig, bottomOrig, rightOrig, topOrig float64) error {
	if b.intensity <= 0.0 {
		return b.drawBasicEllipse(leftOrig, bottomOrig, rightOrig, topOrig)
	}

	left := leftOrig
	bottom := bottomOrig
	right := rightOrig
	top := topOrig
	width := right - left
	height := top - bottom
	cloudRadius := b.ellipseCloudRadius()

	// Omit cloudy border if the ellipse is very small.
	threshold1 := 0.50 * cloudRadius
	if width < threshold1 && height < threshold1 {
		return b.drawBasicEllipse(left, bottom, right, top)
	}

	// Draw a cloudy rectangle instead of an ellipse when the
	// width or height is very small.
	const threshold2 = 5
	if (width < threshold2 && height > 20) || (width > 20 && height < threshold2) {
		return b.cloudyRectangleImpl(left, bottom, right, top, true)
	}

	// Decrease radii (while center point does not move). This makes the
	// "tails" of the curls almost touch the ellipse outline.
	radiusAdj := math.Sin(angle12Deg)*cloudRadius - 1.50
	if width > 2*radiusAdj {
		left += radiusAdj
		right -= radiusAdj
	} else {
		mid := (left + right) / 2
		left = mid - 0.10
		right = mid + 0.10
	}
	if height > 2*radiusAdj {
		top -= radiusAdj
		bottom += radiusAdj
	} else {
		mid := (top + bottom) / 2
		top = mid + 0.10
		bottom = mid - 0.10
	}

	// Flatten the ellipse into a polygon. The segment lengths of the flattened
	// result don't need to be extremely short because the loop below is able to
	// interpolate between polygon points when it computes the center points
	// at which each curl is placed.
	flatPolygon := flattenEllipse(left, bottom, right, top)
	numPoints := len(flatPolygon)
	if numPoints < 2 {
		return nil
	}

	totLen := 0.0
	for i := 1; i < numPoints; i++ {
		totLen += flatPolygon[i-1].distance(flatPolygon[i])
	}

	k := math.Cos(angle34Deg)
	curlAdvance := 2 * k * cloudRadius
	n := int(math.Ceil(totLen / curlAdvance))
	if n < 2 {
		return b.drawBasicEllipse(leftOrig, bottomOrig, rightOrig, topOrig)
	}
	curlAdvance = totLen / float64(n)
	cloudRadius = curlAdvance / (2 * k)
	if cloudRadius < 0.5 {
		cloudRadius = 0.5
		curlAdvance = 2 * k * cloudRadius
	} else if cloudRadius < 3.0 {
		// Draw a small circle when the scaled radius becomes very small.
		// This happens also if intensity is much smaller than 1.
		return b.drawBasicEllipse(leftOrig, bottomOrig, rightOrig, topOrig)
	}

	// Construct centerPoints array, in which each point is the center point of a curl.
	// The length of each centerPoints segment ideally equals curlAdv but that
	// is not true in regions where the ellipse curvature is high.
	centerPoints := make([]point, n)
	centerPointsIndex := 0
	lengthRemain := 0.0
	comparisonToler := b.lineWidth * 0.10

	for i := 0; i+1 < numPoints; i++ {
		p1 := flatPolygon[i]
		p2 := flatPolygon[i+1]
		dx := p2.x - p1.x
		dy := p2.y - p1.y
		length := p1.distance(p2)
		if length == 0 {
			continue
		}
		lengthTodo := length + lengthRemain

		if lengthTodo >= curlAdvance-comparisonToler || i == numPoints-2 {
			cos := cosine(dx, length)
			sin := sine(dy, length)
			d := curlAdvance - lengthRemain
			for {
				x := p1.x + d*cos
				y := p1.y + d*sin
				if centerPointsIndex < len(centerPoints) {
					centerPoints[centerPointsIndex] = point{x, y}
					centerPointsIndex++
				}
				lengthTodo -= curlAdvance
				d += curlAdvance
				if lengthTodo < curlAdvance-comparisonToler {
					break
				}
			}
			lengthRemain = lengthTodo
			if lengthRemain < 0 {
				lengthRemain = 0
			}
		} else {
			lengthRemain += length
		}
	}

	// Note: centerPoints does not repeat the first point as the last point
	// to create a "closing" segment.

	// Place a curl at each point of the centerPoints array.
	// In regions where the ellipse curvature is high, the centerPoints segments
	// are shorter than the actual distance along the ellipse. Thus we must
	// again compute arc adjustments like in cloudy polygons.
	numPoints = centerPointsIndex
	anglePrev := 0.0
	alphaPrev := 0.0

	for i := 0; i < numPoints; i++ {
		idxNext := i + 1
		if i+1 >= numPoints {
			idxNext = 0
		}
		pt := centerPoints[i]
		ptNext := centerPoints[idxNext]
		if i == 0 {
			ptPrev := centerPoints[numPoints-1]
			anglePrev = math.Atan2(pt.y-ptPrev.y, pt.x-ptPrev.x)
			alphaPrev = computeParamsEllipse(ptPrev, pt, cloudRadius, curlAdvance)
		}
		angleCur := math.Atan2(ptNext.y-pt.y, ptNext.x-pt.x)
		alpha := computeParamsEllipse(pt, ptNext, cloudRadius, curlAdvance)

		if err := b.addCornerCurl(anglePrev, angleCur, cloudRadius, pt.x, pt.y, alpha,
			alphaPrev, !b.outputStarted); err != nil {
			return err
		}
		anglePrev = angleCur
		alphaPrev = alpha
	}
	return nil
}

// computeParamsEllipse works out the adjustment angle between two curl centres.
func computeParamsEllipse(pt, ptNext point, r, curlAdv float64) float64 {
	length := pt.distance(ptNext)
	if length == 0 {
		return angle34Deg
	}
	e := length - curlAdv
	arg := (curlAdv/2 + e/2) / r
	if arg < -1.0 || arg > 1.0 {
		return 0.0
	}
	return math.Acos(arg)
}

// removeZeroLengthSegments drops the points that repeat the one before them.
func removeZeroLengthSegments(polygon []point) []point {
	np := len(polygon)
	if np <= 2 {
		return polygon
	}
	const toler = 0.50
	kept := make([]bool, np)
	kept[0] = true
	npNew := np
	ptPrev := polygon[0]

	// Don't remove the last point if it equals the first point.
	for i := 1; i < np; i++ {
		pt := polygon[i]
		if math.Abs(pt.x-ptPrev.x) < toler && math.Abs(pt.y-ptPrev.y) < toler {
			npNew--
		} else {
			kept[i] = true
		}
		ptPrev = pt
	}
	if npNew == np {
		return polygon
	}

	polygonNew := make([]point, 0, npNew)
	for i := 0; i < np; i++ {
		if kept[i] {
			polygonNew = append(polygonNew, polygon[i])
		}
	}
	return polygonNew
}

// drawBasicEllipse draws the ellipse without curls.
func (b *cloudyBorder) drawBasicEllipse(left, bottom, right, top float64) error {
	rx := math.Abs(right-left) / 2
	ry := math.Abs(top-bottom) / 2
	cx := (left + right) / 2
	cy := (bottom + top) / 2
	return b.getArc(0, 2*math.Pi, rx, ry, cx, cy, nil, true)
}

// beginOutput starts the bounding box at the first point written.
func (b *cloudyBorder) beginOutput(x, y float64) error {
	b.bboxMinX = x
	b.bboxMinY = y
	b.bboxMaxX = x
	b.bboxMaxY = y
	b.outputStarted = true

	// Set line join to bevel to avoid spikes
	return b.output.SetLineJoinStyle(2)
}

// updateBBox grows the bounding box to hold the given point.
func (b *cloudyBorder) updateBBox(x, y float64) {
	b.bboxMinX = math.Min(b.bboxMinX, x)
	b.bboxMinY = math.Min(b.bboxMinY, y)
	b.bboxMaxX = math.Max(b.bboxMaxX, x)
	b.bboxMaxY = math.Max(b.bboxMaxY, y)
}

// moveToPoint begins a subpath at the given point.
func (b *cloudyBorder) moveToPoint(p point) error { return b.moveTo(p.x, p.y) }

// moveTo begins a subpath at the given point.
func (b *cloudyBorder) moveTo(x, y float64) error {
	if b.outputStarted {
		b.updateBBox(x, y)
	} else if err := b.beginOutput(x, y); err != nil {
		return err
	}
	return b.output.MoveTo(float32(x), float32(y))
}

// lineToPoint draws a line to the given point.
func (b *cloudyBorder) lineToPoint(p point) error { return b.lineTo(p.x, p.y) }

// lineTo draws a line to the given point.
func (b *cloudyBorder) lineTo(x, y float64) error {
	if b.outputStarted {
		b.updateBBox(x, y)
	} else if err := b.beginOutput(x, y); err != nil {
		return err
	}
	return b.output.LineTo(float32(x), float32(y))
}

// curveTo draws a cubic curve to the given point.
func (b *cloudyBorder) curveTo(ax, ay, bx, by, cx, cy float64) error {
	b.updateBBox(ax, ay)
	b.updateBBox(bx, by)
	b.updateBBox(cx, cy)
	return b.output.CurveTo(float32(ax), float32(ay), float32(bx), float32(by),
		float32(cx), float32(cy))
}

// finish closes the path and pads the bounding box by half the line width.
func (b *cloudyBorder) finish() error {
	if b.outputStarted {
		if err := b.output.ClosePath(); err != nil {
			return err
		}
	}
	if b.lineWidth > 0 {
		d := b.lineWidth / 2
		b.bboxMinX -= d
		b.bboxMinY -= d
		b.bboxMaxX += d
		b.bboxMaxY += d
	}
	return nil
}

// ellipseCloudRadius returns the curl radius of an ellipse.
func (b *cloudyBorder) ellipseCloudRadius() float64 {
	// Equation deduced from Acrobat Reader's appearance streams. Circle
	// annotations have a slightly larger radius than Polygons and Squares.
	return 4.75*b.intensity + 0.5*b.lineWidth
}

// polygonCloudRadius returns the curl radius of a polygon.
func (b *cloudyBorder) polygonCloudRadius() float64 {
	// Equation deduced from Acrobat Reader's appearance streams.
	return 4*b.intensity + 0.5*b.lineWidth
}
