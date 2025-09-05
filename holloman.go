

package HuntingHash

import (
    "compress/gzip"
    "encoding/binary"
    //"bytes"
    "image"
    //"math"
    "fmt"
    "os"

	//"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
    "github.com/wessorh/rez"
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

func (curve *HilbertCurve) MapPoint(i, order int) ( x,y uint32, err error){
	if uint32(order) > curve.Order {
		return 0,0, fmt.Errorf("requested order (%d) exceeds curve data order (%d)", order, curve.Order)
	}

	// rotate
	x = curve.Y[ i ];
    y = curve.X[ i ];

    return x, y, nil
}

func (curve *HilbertCurve) MapBuffer(buffer []byte) (outputBuffer []byte, order int32, im *image.Gray, err error){

	// is the curve large enough?
	order = int32(HilbertCurveOrder(int64(len(buffer))))
	if order > int32(curve.Order) {
		return nil, 0, nil, fmt.Errorf("buffer too large, max order %d, it requires a curve of at least order %d", curve.Order, order)
	}
 	stride := uint32(1 << order)  // 2^order
    total_points := stride * stride
    in_len := uint32(len(buffer))
    im = image.NewGray(image.Rect(0, 0, int(stride), int(stride)))

    //tmpBuffer := make([]byte, total_points);

        for i := uint32(0); i < in_len; i++ {
        	// rotates the image
            x := curve.Y[ i ];
            y := curve.X[ i ];
            index := (y * stride) + x;
            //printf("%ld (%d,%d) %d %ld %ld\n", i+offset, x, y, index, bytes_read, offset);
            // ensure that the indexes are within the bounds of output_buffer
            if (i < in_len && index < total_points) {
                im.Pix[index] = buffer[i];
            }
        }

    //im.Pix = tmpBuffer
	output_im2 := image.NewGray(image.Rect(0, 0, 4, 4))

	err = rez.Convert(output_im2, im, rez.NewLanczosFilter(3))
	if err != nil {
		log.Error().Msg(err.Error())
	}

	outputBuffer = output_im2.Pix


	return outputBuffer, order, im, nil 
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



