package main

import (
    "compress/gzip"
    "encoding/binary"
    "flag"
    "fmt"
    "io"
    "os"
    "runtime"
    "strings"
    "sync"
    "time"
)

type HilbertCurve struct {
    order uint32
    x     []uint32
    y     []uint32
}

const (
    batchSize  = 1024 * 1024
    bufferSize = 1024 * 1024
)

type Worker struct {
    start, end uint32
    x, y       []uint32
    wg         *sync.WaitGroup
}

func (w *Worker) generatePoints(order uint32) {
    defer w.wg.Done()
    
    for i := w.start; i < w.end; i++ {
        gray := i ^ (i >> 1)
        var x, y uint32

        if order <= 16 {
            mask := uint32(1)
            for j := uint32(0); j < order; j++ {
                bit := (gray >> (2 * j)) & 3
                if (bit >> 1) & 1 != 0 {
                    x |= mask
                }
                if bit & 1 != 0 {
                    y |= mask
                }
                mask <<= 1
            }
        } else {
            for j := uint32(0); j < order; j++ {
                bit := (gray >> (2 * j)) & 3
                x |= ((bit >> 1) & 1) << j
                y |= (bit & 1) << j
            }
        }

        w.x[i] = x
        w.y[i] = y
    }
}

func generateHilbertCurve(order uint32) *HilbertCurve {
    size := uint32(1 << (2 * order))
    curve := &HilbertCurve{
        order: order,
        x:     make([]uint32, size),
        y:     make([]uint32, size),
    }

    numWorkers := runtime.NumCPU()
    pointsPerWorker := size / uint32(numWorkers)
    
    var wg sync.WaitGroup
    workers := make([]Worker, numWorkers)

    for i := 0; i < numWorkers; i++ {
        start := uint32(i) * pointsPerWorker
        end := start + pointsPerWorker
        if i == numWorkers-1 {
            end = size
        }

        workers[i] = Worker{
            start: start,
            end:   end,
            x:     curve.x,
            y:     curve.y,
            wg:    &wg,
        }
        wg.Add(1)
        go workers[i].generatePoints(order)
    }

    wg.Wait()
    return curve
}

type bufferedWriter struct {
    w   io.Writer
    buf []byte
    pos int
}

func newBufferedWriter(w io.Writer, size int) *bufferedWriter {
    return &bufferedWriter{
        w:   w,
        buf: make([]byte, size),
    }
}

func (bw *bufferedWriter) Write(p []byte) (n int, err error) {
    n = len(p)
    for len(p) > 0 {
        remaining := len(bw.buf) - bw.pos
        if remaining == 0 {
            if err := bw.Flush(); err != nil {
                return 0, err
            }
        }
        copyLen := min(remaining, len(p))
        copy(bw.buf[bw.pos:], p[:copyLen])
        bw.pos += copyLen
        p = p[copyLen:]
    }
    return n, nil
}

func (bw *bufferedWriter) Flush() error {
    if bw.pos == 0 {
        return nil
    }
    _, err := bw.w.Write(bw.buf[:bw.pos])
    bw.pos = 0
    return err
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func (curve *HilbertCurve) saveHilbertCurve(filename string, compress bool) error {
    file, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("error opening file for writing: %v", err)
    }
    defer file.Close()

    var w io.Writer = file
    if compress {
        gw := gzip.NewWriter(file)
        defer gw.Close()
        w = gw
    }

    writer := newBufferedWriter(w, bufferSize)
    defer writer.Flush()

    header := make([]byte, 4)
    binary.LittleEndian.PutUint32(header, curve.order)
    if _, err := writer.Write(header); err != nil {
        return fmt.Errorf("error writing order: %v", err)
    }

    size := uint32(1 << (2 * curve.order))
    buf := make([]byte, batchSize*4)

    for i := uint32(0); i < size; i += batchSize {
        end := min(int(i+batchSize), int(size))
        pos := 0
        for j := i; j < uint32(end); j++ {
            binary.LittleEndian.PutUint32(buf[pos:], curve.x[j])
            pos += 4
        }
        if _, err := writer.Write(buf[:pos]); err != nil {
            return fmt.Errorf("error writing x coordinates: %v", err)
        }
    }

    for i := uint32(0); i < size; i += batchSize {
        end := min(int(i+batchSize), int(size))
        pos := 0
        for j := i; j < uint32(end); j++ {
            binary.LittleEndian.PutUint32(buf[pos:], curve.y[j])
            pos += 4
        }
        if _, err := writer.Write(buf[:pos]); err != nil {
            return fmt.Errorf("error writing y coordinates: %v", err)
        }
    }

    return nil
}

func loadHilbertCurve(filename string) (*HilbertCurve, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, fmt.Errorf("error opening file for reading: %v", err)
    }
    defer file.Close()

    var r io.Reader = file
    if isGzipped(filename) {
        gr, err := gzip.NewReader(file)
        if err != nil {
            return nil, fmt.Errorf("error creating gzip reader: %v", err)
        }
        defer gr.Close()
        r = gr
    }

    curve := &HilbertCurve{}

    var orderBuf [4]byte
    if _, err := io.ReadFull(r, orderBuf[:]); err != nil {
        return nil, fmt.Errorf("error reading order: %v", err)
    }
    curve.order = binary.LittleEndian.Uint32(orderBuf[:])

    size := uint32(1 << (2 * curve.order))
    curve.x = make([]uint32, size)
    curve.y = make([]uint32, size)

    buf := make([]byte, batchSize*4)

    for i := uint32(0); i < size; i += batchSize {
        end := min(int(i+batchSize), int(size))
        readSize := (end - int(i)) * 4
        if _, err := io.ReadFull(r, buf[:readSize]); err != nil {
            return nil, fmt.Errorf("error reading x coordinates: %v", err)
        }
        for j := 0; j < readSize/4; j++ {
            curve.x[i+uint32(j)] = binary.LittleEndian.Uint32(buf[j*4:])
        }
    }

    for i := uint32(0); i < size; i += batchSize {
        end := min(int(i+batchSize), int(size))
        readSize := (end - int(i)) * 4
        if _, err := io.ReadFull(r, buf[:readSize]); err != nil {
            return nil, fmt.Errorf("error reading y coordinates: %v", err)
        }
        for j := 0; j < readSize/4; j++ {
            curve.y[i+uint32(j)] = binary.LittleEndian.Uint32(buf[j*4:])
        }
    }

    return curve, nil
}

func isGzipped(filename string) bool {
    if strings.HasSuffix(strings.ToLower(filename), ".gz") {
        return true
    }

    file, err := os.Open(filename)
    if err != nil {
        return false
    }
    defer file.Close()

    var buf [2]byte
    if _, err := io.ReadFull(file, buf[:]); err != nil {
        return false
    }
    file.Seek(0, 0)

    return buf[0] == 0x1f && buf[1] == 0x8b
}

func formatSize(size int64) string {
    const unit = 1024
    if size < unit {
        return fmt.Sprintf("%d B", size)
    }
    div, exp := int64(unit), 0
    for n := size / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func main() {
    order := flag.Uint("order", 3, "Order of the Hilbert curve")
    filename := flag.String("file", "hilbert_curve.dat", "Output/input file name")
    compress := flag.Bool("compress", false, "Use gzip compression")
    verbose := flag.Bool("verbose", false, "Print timing and size information")
    mode := flag.String("mode", "both", "Operation mode: generate, load, or both")
    flag.Parse()

    if *order > 16 {
        fmt.Println("Warning: orders > 16 may be slow and memory intensive")
    }

    outputFile := *filename
    if *compress && !strings.HasSuffix(strings.ToLower(outputFile), ".gz") {
        outputFile += ".gz"
    }

    startTime := time.Now()
    var curve *HilbertCurve

    switch strings.ToLower(*mode) {
    case "generate", "both":
        genStart := time.Now()
        curve = generateHilbertCurve(uint32(*order))
        if *verbose {
            fmt.Printf("Generation time: %v\n", time.Since(genStart))
        }

        saveStart := time.Now()
        err := curve.saveHilbertCurve(outputFile, *compress)
        if err != nil {
            fmt.Printf("Error saving curve: %v\n", err)
            os.Exit(1)
        }
        
        if *verbose {
            saveTime := time.Since(saveStart)
            fileInfo, _ := os.Stat(outputFile)
            fmt.Printf("Save time: %v\n", saveTime)
            fmt.Printf("File size: %s\n", formatSize(fileInfo.Size()))
            if *compress {
                // Calculate total bytes without compression (8 bytes per point)
                totalPoints := uint64(1) << (2 * (*order))
                uncompressedSize := float64(totalPoints * 8)
                compressionRatio := float64(fileInfo.Size()) / uncompressedSize
                fmt.Printf("Compression ratio: %.2f%%\n", compressionRatio*100)
            }
        }

        if *mode == "generate" {
            break
        }
        fallthrough

    case "load":
        loadStart := time.Now()
        loadedCurve, err := loadHilbertCurve(outputFile)
        if err != nil {
            fmt.Printf("Error loading curve: %v\n", err)
            os.Exit(1)
        }
        if *verbose {
            fmt.Printf("Load time: %v\n", time.Since(loadStart))
        }
        curve = loadedCurve

    default:
        fmt.Printf("Invalid mode: %s\n", *mode)
        os.Exit(1)
    }

    if *verbose {
        fmt.Printf("Total time: %v\n", time.Since(startTime))
        fmt.Printf("Memory used: %.2f MB\n", float64(runtime.MemStats{}.Alloc)/(1024*1024))
    }

    fmt.Printf("Successfully processed Hilbert curve of order %d\n", curve.order)
}
