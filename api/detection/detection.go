package detection

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"sync"

	"gocv.io/x/gocv"
)

const (
	YoloWidth  = 640
	YoloHeight = 640
	NumAnchors = 8400
	PersonId   = 4
)

type Detection struct {
	Box        image.Rectangle
	Confidence float32
}

type Detector struct {
	net            gocv.Net
	scoreThreshold float64
	nmsThreshold   float64
}

func NewDetector(modelPath string, scoreThreshold, nmsThreshold float64) (*Detector, error) {
	net := gocv.ReadNetFromONNX(modelPath)
	if net.Empty() {
		return nil, ErrFailedToLoadModel
	}

	net.SetPreferableBackend(gocv.NetBackendDefault)
	net.SetPreferableTarget(gocv.NetTargetCPU)

	return &Detector{
		net:            net,
		scoreThreshold: scoreThreshold,
		nmsThreshold:   nmsThreshold,
	}, nil
}

type SharedDetector struct {
	Mtx      sync.Mutex
	Detector *Detector
}

func NewSharedDetector(modelPath string, scoreThreshold, nmsThreshold float64) *SharedDetector {
	detector, err := NewDetector(modelPath, scoreThreshold, nmsThreshold)
	if err != nil {
		log.Println(err)
		return nil
	}

	return &SharedDetector{
		Detector: detector,
	}
}

var MainDetector *SharedDetector

func (d *Detector) Close() error {
	return d.net.Close()
}

func (d *Detector) Detect(img *gocv.Mat) ([]Detection, error) {
	if img.Empty() {
		return nil, ErrInputFrameIsEmpty
	}

	blob := gocv.BlobFromImage(*img, 1.0/255.0, image.Pt(YoloWidth, YoloHeight), gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	d.net.SetInput(blob, "")
	outputs := d.net.Forward("")
	defer outputs.Close()

	data, err := outputs.DataPtrFloat32()
	if err != nil {
		return nil, ErrFailedToGetPointer
	}

	imgWidth := float32(img.Cols())
	imgHeight := float32(img.Rows())
	xScale := imgWidth / YoloWidth
	yScale := imgHeight / YoloHeight

	var bboxes []image.Rectangle
	var confidences []float32

	for c := 0; c < NumAnchors; c++ {
		personScore := data[PersonId*NumAnchors+c]

		if personScore > float32(d.scoreThreshold) {
			cx := data[0*NumAnchors+c] * xScale
			cy := data[1*NumAnchors+c] * yScale
			w := data[2*NumAnchors+c] * xScale
			h := data[3*NumAnchors+c] * yScale

			x := int(cx - w/2)
			y := int(cy - h/2)

			bboxes = append(bboxes, image.Rect(x, y, x+int(w), y+int(h)))
			confidences = append(confidences, personScore)
		}
	}

	if len(bboxes) == 0 {
		return []Detection{}, nil
	}

	indices := gocv.NMSBoxes(bboxes, confidences, float32(d.scoreThreshold), float32(d.nmsThreshold))

	results := make([]Detection, 0, len(indices))
	for _, idx := range indices {
		if idx >= len(bboxes) {
			continue
		}
		results = append(results, Detection{
			Box:        bboxes[idx],
			Confidence: confidences[idx],
		})
	}

	return results, nil
}

func DrawOverlay(img *gocv.Mat, detections []Detection, colorCode color.RGBA) {
	for _, det := range detections {
		gocv.Rectangle(img, det.Box, colorCode, 2)
		text := fmt.Sprintf("Person: %.1f%%", det.Confidence*100)
		gocv.PutText(img, text, image.Pt(det.Box.Min.X, det.Box.Min.Y-10), gocv.FontHersheySimplex, 0.5, colorCode, 2)
	}
}
