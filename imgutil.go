package imgutil

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"

	"gocv.io/x/gocv"
	"golang.org/x/image/draw"
)

type Image struct{ Img gocv.Mat }

func NewCVImage(imagePath string) (Image, error) {

	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	if img.Empty() {
		return Image{}, fmt.Errorf("imgutil.NewImage(): Failed to load image %s", imagePath)
	}
	i := Image{Img: img}
	return i, nil

}

func NewImageFromBytes(bytebuf []byte) (Image, error) {
	img, err := gocv.IMDecode(bytebuf, gocv.IMReadColor)
	if err != nil {
		return Image{}, fmt.Errorf("imgutil.NewImage(): Failed to IMDecode image %s", err)
	}
	if img.Empty() {
		return Image{}, fmt.Errorf("imgutil.NewImage(): Failed to IMDecode image %s", err)
	}
	i := Image{Img: img}
	return i, nil
}

func (img Image) PutText(text string) {
	color := color.RGBA{255, 0, 140, 0}

	var thickness, x, y int
	var fontscale float64
	//TODO: Calulate the following from image size some way or another:
	if img.Img.Rows() <= 256 { // Thumb
		thickness = 2
		fontscale = 0.6
		x = 10
		y = 25
	} else if img.Img.Rows() <= 400 {
		thickness = 1
		fontscale = 0.3
		x = 10
		y = 10
	} else if img.Img.Rows() <= 720 {
		thickness = 2
		fontscale = 0.8
		x = 10
		y = 50
	} else if img.Img.Rows() <= 2992 {
		thickness = 3
		fontscale = 3
		x = 10
		y = 100
	}

	pt := image.Point{x, y}
	gocv.PutText(&img.Img, text, pt, gocv.FontItalic, fontscale, color, thickness)
}

func (img Image) Close() {
	err := img.Img.Close()
	if err != nil {
		fmt.Printf("img.Img.Close() failed: Bad matrix\n")
	}
}

func (img Image) GetMeanBrightness() (float64, error) {

	hsv := gocv.NewMat()
	defer hsv.Close()
	if img.Img.Empty() {

		return -1.0, fmt.Errorf("GetMeanBrightness: Bad matrix")
	}

	gocv.CvtColor(img.Img, &hsv, gocv.ColorBGRToHSV)
	return hsv.Mean().Val3 / 255.0, nil
}

func ScaleJpegBufr(inputBufr []byte, sizeX int, sizeY int) ([]byte, error) {
	reader := bytes.NewReader(inputBufr)
	// Decode the image (from PNG to image.Image):
	src, err := jpeg.Decode(reader)
	if err != nil {
		return []byte{}, fmt.Errorf("ScaleJpegBufr.Decode(): %v", err)
	}

	// Set the expected size that you want:
	dst := image.NewRGBA(image.Rect(0, 0, sizeX, sizeY))
	draw.BiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	outBuf := new(bytes.Buffer)
	err = jpeg.Encode(outBuf, dst, nil)
	if err != nil {
		return []byte{}, fmt.Errorf("ScaleJpegBufr.jpeg.Encode: %v", err)
	}

	return outBuf.Bytes(), nil
}
func ScaleJpegFile(inputPath string, outputPath string, sizeX int, sizeY int) error {

	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("ResizeJpeg.os.Open(): %v", err)
	}

	defer input.Close()

	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("ResizeJpeg.os.Create(): %v", err)
	}

	defer output.Close()

	// Decode the image (from PNG to image.Image):
	src, err := jpeg.Decode(input)
	if err != nil {
		return fmt.Errorf("ResizeJpeg.jpeg.Decode(): %v", err)
	}

	// Set the size:
	dst := image.NewRGBA(image.Rect(0, 0, sizeX, sizeY))
	// Resize:
	// Slow:
	draw.BiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	// Slower:
	//draw.CatmullRom.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	// Encode to `output`:
	err = jpeg.Encode(output, dst, nil)
	if err != nil {
		return fmt.Errorf("ResizeJpeg.jpeg.Encode: %v", err)
	}
	return nil

}

// Wow. This is soo much easier in python-opencv
func (img Image) BlueMask(imagePath string) error {
	lowerBlue := gocv.NewMatFromScalar(gocv.NewScalar(102.0, 31.0, 160.0, 0.0), gocv.MatTypeCV8UC3)
	upperBlue := gocv.NewMatFromScalar(gocv.NewScalar(115.0, 255.0, 255.0, 0.0), gocv.MatTypeCV8UC3)
	hsv := gocv.NewMat()
	defer hsv.Close()
	if img.Img.Empty() {
		return fmt.Errorf("BlueMask: Bad matrix")

	}

	gocv.CvtColor(img.Img, &hsv, gocv.ColorBGRToHSV)

	channels, rows, cols := hsv.Channels(), hsv.Rows(), hsv.Cols()
	lowerChans := gocv.Split(lowerBlue)
	lowerMask := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC3)
	lowerMaskChans := gocv.Split(lowerMask)
	// split HSV lower bounds into H, S, V channels
	upperChans := gocv.Split(upperBlue)
	upperMask := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC3)
	upperMaskChans := gocv.Split(upperMask)

	// copy HSV values to upper and lower masks
	for c := 0; c < channels; c++ {
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				lowerMaskChans[c].SetUCharAt(i, j, lowerChans[c].GetUCharAt(0, 0))
				upperMaskChans[c].SetUCharAt(i, j, upperChans[c].GetUCharAt(0, 0))
			}
		}
	}

	gocv.Merge(lowerMaskChans, &lowerMask)
	gocv.Merge(upperMaskChans, &upperMask)
	// global mask

	mask := gocv.NewMat()
	defer mask.Close()
	gocv.InRange(hsv, lowerMask, upperMask, &mask)

	window := gocv.NewWindow("Hello")
	window.ResizeWindow(640, 480)
	window.IMShow(mask)
	window.WaitKey(-1)
	return nil
}
