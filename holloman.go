

package holloman

import (
    "compress/gzip"
    "encoding/binary"
    "bytes"
    "image"
    "math"
    "fmt"
    "os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
    "github.com/bamiaux/rez"
	// "gopkg.in/gographics/imagick.v2/imagick"
)
const (
	M_PI = 3.141592653589793115997963468544185161590576171875
	ORDER_ALPHABET = "  cdefghijkmnopqrstuvwxyz"
)


// HilbertCurve represents the structure to hold the Hilbert curve and its order
type HilbertCurve struct {
    Order uint32
    X     []uint32
    Y     []uint32
}

/**
func init(){
 	imagick.Initialize()
}

func Close(){
	imagick.Terminate()
}
**/

// LoadHilbertCurve reads a gzipped Hilbert curve from a file and returns it
func LoadHilbertCurve(filename string) (*HilbertCurve, error) {
    // Open the file for reading
    file, err := os.Open(filename)
    if err != nil {
        return nil, fmt.Errorf("error opening file: %w", err)
    }
    defer file.Close()

    // Create a gzip reader
    gzReader, err := gzip.NewReader(file)
    if err != nil {
        return nil, fmt.Errorf("error creating gzip reader: %w", err)
    }
    defer gzReader.Close()

    // Create a new HilbertCurve struct
    curve := &HilbertCurve{}

    // Read the order
    err = binary.Read(gzReader, binary.LittleEndian, &curve.Order)
    if err != nil {
        return nil, fmt.Errorf("error reading order: %w", err)
    }

    // Calculate size based on order
    size := uint32(1 << (2 * curve.Order))

    // Initialize slices
    curve.X = make([]uint32, size)
    curve.Y = make([]uint32, size)

    // Read X coordinates
    err = binary.Read(gzReader, binary.LittleEndian, curve.X)
    if err != nil {
        return nil, fmt.Errorf("error reading X coordinates: %w", err)
    }

    // Read Y coordinates
    err = binary.Read(gzReader, binary.LittleEndian, curve.Y)
    if err != nil {
        return nil, fmt.Errorf("error reading Y coordinates: %w", err)
    }

    return curve, nil
}


func HilbertCurveOrder(n int64) int {
    if n <= 0 {
        return 0
    }
    
    // Subtract 1 to handle perfect square cases
    n--
    order := 0
    
    // Find position of highest set bit
    for n > 0 {
        n >>= 1
        order++
    }
    
    // Return ceiling of order/2
    return (order + 1) >> 1
}

func (curve *HilbertCurve) MapPoint(uint64 i, int order) ( x,y uint32, err error){
	if order > curve.Order {
		return 0,0, fmt.Errorf("requested order (%d) exceeds curve data order (%d)", order, curve.Order)
	}

	// rotate
	x := curve.Y[ i ];
    y := curve.X[ i ];

    return x, y, nil
}

func (curve *HilbertCurve) MapBuffer(buffer []byte) (outputBuffer []byte, order int32, err error){

	// is the curve large enough?
	order = int32(HilbertCurveOrder(int64(len(buffer))))
	if order > int32(curve.Order) {
		return nil, 0, fmt.Errorf("buffer too large, max order %d, it requires a curve of at least order %d", curve.Order, order)
	}
 	stride := uint32(1 << order)  // 2^order
    total_points := stride * stride
    in_len := uint32(len(buffer))

    tmpBuffer := make([]byte, total_points);

        for i := uint32(0); i < in_len; i++ {
        	// rotates the image
            x := curve.Y[ i ];
            y := curve.X[ i ];
            index := (y * stride) + x;
            //printf("%ld (%d,%d) %d %ld %ld\n", i+offset, x, y, index, bytes_read, offset);
            // ensure that the indexes are within the bounds of output_buffer
            if (i < in_len && index < total_points) {
                tmpBuffer[index] = buffer[i];
            }
        }

    if zerolog.GlobalLevel() == zerolog.DebugLevel {
        savePGM("/tmp/debug.pgm", tmpBuffer, int(stride))
    }
    im := image.NewGray(image.Rect(0, 0, int(stride), int(stride)))
    im.Pix = tmpBuffer
	output_im2 := image.NewGray(image.Rect(0, 0, 4, 4))

//	mw := imagick.NewMagickWand()

	err = rez.Convert(output_im2, im, rez.NewLanczosFilter(3))
	if err != nil {
		log.Error().Msg(err.Error())
	}

	outputBuffer = output_im2.Pix
	//outputBuffer, err = lancozDownSample(tmpBuffer, int(stride) , 2 )
    // if zerolog.GlobalLevel() == zerolog.DebugLevel {
	//     PrintImage4x4(outputBuffer)
    // }

	return outputBuffer, order, nil 
}

// PrintImage4x4 prints a 4x4 image in hexadecimal format
func PrintImage4x4(image []uint8) {
    for i := 0; i < 4; i++ {
        for j := 0; j < 4; j++ {
            fmt.Printf("%02x ", image[i*4+j])
        }
        fmt.Println()
    }
    fmt.Println()
}

// SavePGM saves grayscale image data as a PGM file
func savePGM(filename string, data []uint8, stride int) error {
    // Open file for writing
    file, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    // Write PGM header
    header := fmt.Sprintf("P5\n%d %d\n255\n", stride, stride)
    _, err = file.WriteString(header)
    if err != nil {
        return fmt.Errorf("failed to write header: %w", err)
    }

    // Write image data row by row
    for y := 0; y < stride; y++ {
        start := y * stride
        end := start + stride
        _, err = file.Write(data[start:end])
        if err != nil {
            return fmt.Errorf("failed to write data at row %d: %w", y, err)
        }
    }

    return nil
}
// CreatePGM creates a PGM format buffer from grayscale image data
func createPGMBuffer(data []byte, stride int) (*bytes.Buffer, error) {
    buf := new(bytes.Buffer)

    // Write PGM header
    header := fmt.Sprintf("P5\n%d %d\n255\n", stride, stride)
    _, err := buf.WriteString(header)
    if err != nil {
        return nil, fmt.Errorf("failed to write header: %w", err)
    }

    // Write image data row by row
    for y := 0; y < stride; y++ {
        start := y * stride
        end := start + stride
        _, err = buf.Write(data[start:end])
        if err != nil {
            return nil, fmt.Errorf("failed to write data at row %d: %w", y, err)
        }
    }

    return buf, nil
}

// lanczosKernel implements the Lanczos kernel function
func lanczosKernel(x, radius float64) float64 {
    if x == 0 {
        return 1
    }
    sinc := math.Sin(M_PI*x) / (M_PI * x)
    return sinc * math.Sin(M_PI*x/radius) / (M_PI * x / radius)
}

// LanczosDownsample performs Lanczos downsampling on an image buffer
func lancozDownSample(inputBuffer []byte, inOrder, outOrder int) (outputBuffer []byte, err error){
    // Input validation
    if len(inputBuffer) == 0 || len(inputBuffer) <= 0 || outOrder <= 0 {
        return nil, fmt.Errorf("invalid input parameters")
    }
	inputSize   := len(inputBuffer)
	inputStride := 1 << inOrder

	outputSize   := 1 << (outOrder*2)
	outputStride := 1 << outOrder
    outputBuffer  = make([]byte, outputSize)

    scale := float64(inputSize) / float64(outputSize)
    const radius = 3 // Typical radius for Lanczos filter

    for y := 0; y < outputStride; y++ {
        for x := 0; x < outputStride; x++ {
            var sum, weightSum float64

            // Calculate center point for accurate sampling
            xCenter := (float64(x)+0.5)*scale - 0.5
            yCenter := (float64(y)+0.5)*scale - 0.5

            for dy := -radius; dy <= radius; dy++ {
                for dx := -radius; dx <= radius; dx++ {
                    xIn := xCenter + float64(dx)
                    yIn := yCenter + float64(dy)

                    // Round to nearest integer for accurate sampling
                    inputX := int(math.Round(xIn))
                    inputY := int(math.Round(yIn))

                    fmt.Printf("%d\n", (inputY*inputStride)+inputX)

                    // Check bounds to prevent out-of-bounds access
                    if inputX >= 0 && inputX < inputStride && inputY >= 0 && inputY < inputStride {
                        // Scale dx and dy for correct kernel evaluation
                        dxScaled := float64(dx) / scale
                        dyScaled := float64(dy) / scale

                        weight := lanczosKernel(dxScaled, float64(radius)) * 
                                lanczosKernel(dyScaled, float64(radius))
                        sum += weight * float64(inputBuffer[(inputY*inputStride)+inputX])
                        weightSum += weight
                    }
                }
            }

            // Handle division by zero and clamp result
            if weightSum == 0 {
                outputBuffer[y*outputStride+x] = 0
            } else {
                result := sum / weightSum
                // Clamp results between 0 and 255
                result = math.Max(0, math.Min(255, result))
                outputBuffer[y*outputStride+x] = uint8(math.Round(result))
            }
        }
    }

    return outputBuffer, nil
}

