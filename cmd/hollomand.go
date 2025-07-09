package main

//
// Copyright 2025 (c) By Rick Wesson & Support Intelligence, Inc.
// Licenced under the RLL 1.0

/**
  we havn't implemented the REST client, use python or yara to test it.
**/

// go:generate  protoc --go_out=.  --go-grpc_out=. holloman.proto
import (
	"context"
	_ "embed"
	"strings"
	"bytes"
	"sync"
	"flag"
	"fmt"
	"net"
	"os"
	"io"
	"syscall"
	"time"
	"net/http"
    "crypto/sha1"
	"encoding/json"
	"path/filepath"

	"github.com/OneOfOne/xxhash"
	"github.com/hosom/gomagic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	hh "github.com/wessorh/HuntingHash"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/glaslos/ssdeep"
	"github.com/glaslos/tlsh"
	"github.com/eciavatta/sdhash"
)

const (
	BUFFER_LEN_MIN = 64
)

var (
	curveFile   string
	damonize    *bool
	dna         *bool
	location    string
	ep          string // execution pattern (client, server, stand_alone)
	filename    string
	verbose     *bool
	ssdf        *bool
	rest_port   string
	do_sdhash   *bool
	do_tlsh		*bool
	dir			string

	//go:embed LICENSE.md
	LICENCE string
)

type HollomanServer struct {
	hh.HollomanServer

	curve 	*hh.HilbertCurve
	m     	*magic.Magic
	mu 		sync.Mutex // mutex prevents cgo memory access errors on calls to libmagic
}


func NewServer(curve *hh.HilbertCurve, dna bool) (s *HollomanServer, err error) {

	s = new(HollomanServer)
	s.curve = curve
	if err != nil {
		return nil, err
	}
	if !dna {
		s.m, err = magic.Open(magic.MAGIC_NONE)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
	}
	return s, nil
}
func restCapabilities(hs *HollomanServer) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {
		cp, _ := hs.Capabilities(context.Background(), nil)
		log.Debug().Msgf("/holloman/v2/capabilities %v", cp)
		js, err := json.Marshal(cp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}

	return http.HandlerFunc(fn)
}

func restClusterBuffer(hs *HollomanServer) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {
        log.Info().Msgf("%v", r)
		breq := new(hh.BufferRequest)
		r.ParseMultipartForm(32 << 20) // limit your max input length!
		var buf bytes.Buffer
		// in your case file would be fileupload
		file, header, err := r.FormFile("holloman-data")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		name := strings.Split(header.Filename, ".")
		log.Debug().Msgf("File name %s\n", name[0])
		breq.Label=name[0]
		// Copy the file data to my buffer
		io.Copy(&buf, file)

		breq.Buffer = buf.Bytes()
		resp, err := hs.ClusterBuffer(context.Background(), breq)
		log.Debug().Msgf("/clusterBuffer %v", resp)
		js, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}

	return http.HandlerFunc(fn)
}


func restServer(hs *HollomanServer) {

	http.Handle("/holloman/v2/capabilities", restCapabilities(hs))
	http.Handle("/holloman/v2/hh128", restClusterBuffer(hs))

	log.Error().Msgf("server: %v", http.ListenAndServe(rest_port, nil))
}

func init() {

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	flag.StringVar(&curveFile, "curve", "hilbert_curve.dat.gz", "pre-generated hilbert curve (gzip compressed)")
	flag.StringVar(&location, "grpc", ":50051", "location to listen :port or /path/to/unix.socket")
	flag.StringVar(&filename, "f", "", "file to generate an identifier for")
	flag.StringVar(&rest_port, "rest-port", ":50005", "port to listen for REST transactions")
	flag.StringVar(&dir, "d", "", "recursive process all fines in directory")

	server := flag.Bool("S", false, "Server")
	client := flag.Bool("C", false, "Client")
	help := flag.Bool("h", false, "help")
	ssdf = flag.Bool("ssdeep", false, "enable ssdeep results")
	dna = flag.Bool("dna", false, "the server should only be used for DNA clustering")
	verbose = flag.Bool("v", false, "verbose")
	licence := flag.Bool("license", false, "print licence")
    do_tlsh = flag.Bool("tlsh", false, "calculate TLSH")
    do_sdhash = flag.Bool("sdhash", false, "calculate TLSH")

	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	if *licence {
		fmt.Println(LICENCE)
		os.Exit(0)
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	ep = "stand_alone"
	if *server {
		ep = "server"
	} else if *client {
		ep = "client"
	} else if len(rest_port) > 0 {
		ep = "rest_server"
	} else {
		flag.Usage()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Info().Msgf("ep: %s", ep)
	}

	log.Debug().Msgf("ep: %s", ep)
}


func dirIsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func producer(path string, info os.FileInfo, err error) error {
	return nil
}

func processDirectory(dir string){
			// walk the dir until it is empty
			for {
				err := filepath.Walk(dir, producer)

				if err != nil {
					log.Error().Msg(err.Error())
					continue
				}

				is_empty, err := dirIsEmpty(dir)

				if is_empty || err != nil {
					log.Info().Msgf("%s is empty\n", dir)
					break
				}

				// default exit loop until we write better dir empty code...
				break
			}
}


func readStdIn() []byte {
    var result []byte
    buffer := make([]byte, 1024)

    for {
        n, err := os.Stdin.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
            return nil
        }
        result = append(result, buffer[:n]...)
    }

    return result
}

func client() {
	var buffer []byte

	// Create a new client
	client, err := NewHollomanClient(location)
	if err != nil {
		log.Fatal().Msgf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Example: Get capabilities
	/**
	capabilities, err := client.GetCapabilities("fast", 5)
	if err != nil {
		log.Fatal().Msgf("Failed to get capabilities: %v", err)
	}
	log.Info().Msgf("Capabilities received: %v", capabilities)
**/
	// Example: Cluster buffer
	if filename == "-" {
		buffer = readStdIn()
	}else{
		buffer, err = getMmappedBuffer(filename)
		if err != nil {
			log.Fatal().Msgf(err.Error())
		}
		defer syscall.Munmap(buffer)
	}
	//sampleBuffer := []byte("sample data")
	rsp, err := client.ClusterBuffer(buffer, filename)
	if err != nil {
		log.Fatal().Msgf("Failed to cluster buffer: %v", err)
	}
	log.Debug().Msgf("Cluster response received: HOrder=%d, Id=%s, Magic=%s",
		rsp.HOrder, rsp.Id, rsp.Magic)
	fmt.Printf("%s\t%s\t%s\t%s\t%s\n", filename, rsp.Id, rsp.Ssdeep, rsp.Tlsh, rsp.Sdhash)

}

func server(srvr *HollomanServer) {
	lis, err := net.Listen("tcp", location)
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(1024*10e7), grpc.MaxSendMsgSize(1024*10e7), withServerUnaryInterceptor())
	hh.RegisterHollomanServer(s, srvr)

	log.Debug().Msgf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatal().Msgf("failed to serve: %v", err)
	}
}

func serverInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (h interface{}, err error) {
	if *verbose {
		start := time.Now()
		// Calls the handler
		h, err = handler(ctx, req)

		log.Info().Msgf("Request - Method:%s\tDuration:%s\t\tError:%v",
			info.FullMethod, time.Since(start), err)
	} else {
		h, err = handler(ctx, req)
	}
	return h, err
}
func (server *HollomanServer) Capabilities(context.Context, *hh.ServiceCapabilities) (*hh.ServiceCapabilities, error) {
	cah := new(hh.ServiceCapabilities)
	cah.Acceleration = "none"
	cah.MaxOrder = int32(server.curve.Order)

	cah.Magic = "filemagic"
	if *dna {
		cah.Magic = "dna/iching"
	}
	cah.Ssdeep = false
	if *ssdf {
		cah.Ssdeep = true
	}

	return cah, nil
}

func (server *HollomanServer) ClusterBuffer(ctx context.Context, req *hh.BufferRequest) (br *hh.BufferResponse, err error) {

	var voxel []byte
	br = new(hh.BufferResponse)
	if len(req.Label) > 0 {
		br.Label=req.Label
	}
	if len(req.Buffer) < 64 {
		return nil, fmt.Errorf("buffer length of %d is too small. minum length is %d", len(req.Buffer), BUFFER_LEN_MIN)
	}
	voxel, br.HOrder, err = server.curve.MapBuffer(req.Buffer)
	if *dna {
		br.Magic = "dna/iching"
		br.Id = fmt.Sprintf("%c.%032x", hh.ORDER_ALPHABET[br.HOrder], voxel)
	} else {
		server.mu.Lock()
		br.Magic, err = server.m.Buffer(req.Buffer)
		// br.Magic = magic.Buffer( server.m, req.Buffer)
		defer server.mu.Unlock()
		_ = err
		ch32 := xxhash.ChecksumString32(fmt.Sprintf("%-60.60s", br.Magic))
		br.Id = fmt.Sprintf("%c%08x.%32.32x", hh.ORDER_ALPHABET[br.HOrder], ch32, voxel)
		// preform sha1 on buffer
 		var sha = sha1.New()
 		sha.Write(req.Buffer)
 		br.Sha1 = fmt.Sprintf("%40x", sha.Sum(nil))
 		br.Label = req.Label
	}

	if *ssdf && len(req.Buffer) > 4096 {
		//preform ssdeep hash on buffer
		s, err := ssdeep.FuzzyBytes(req.Buffer)
		if err != nil {
			br.Ssdeep = err.Error()
		}
		br.Ssdeep = s
	}

	if *do_sdhash {
		f, err:= sdhash.CreateSdbfFromBytes(req.Buffer)
		if err == nil {
			sdbf := f.Compute()
			br.Sdhash = sdbf.String()
		}
	}

	if *do_tlsh && len(req.Buffer) > 256 {
		f, err := tlsh.HashBytes(req.Buffer)
		if err == nil {
			br.Tlsh = f.String()
		}
	}


	return br, nil
}

func withServerUnaryInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(serverInterceptor)
}
func main() {
	var srvr *HollomanServer

	if curveFile == "" {
		flag.Usage()
		return
	}

	if ep != "client" {
		//read the compressed curve
		curve, err := hh.LoadHilbertCurve(curveFile)
		if err != nil {
			log.Fatal().Msgf("curve file %s is invalid", curveFile)
		}
		log.Debug().Msgf("loaded order %d hilbert curve from %s", curve.Order, curveFile)

		// start server, damonize?
		srvr, err = NewServer(curve, *dna)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}
	}

	switch ep {
	case "server":
		server(srvr)

	case "rest_server":
		restServer(srvr)

	case "client":
		client()

	case "stand_alone":
		//read the compressed curve
		curve, err := hh.LoadHilbertCurve(curveFile)
		if filename == "" {
			return
		}
		buffer, err := getMmappedBuffer(filename)
		if err != nil {
			log.Fatal().Msgf(err.Error())
		}
		defer syscall.Munmap(buffer)
		voxel, order, err := curve.MapBuffer(buffer)
		mbuff, err := srvr.m.Buffer(buffer)
		// mbuff := magic.Buffer(srvr.m, buffer)
		ch32 := xxhash.ChecksumString32(fmt.Sprintf("%-60.60s", mbuff))

		if *dna {
			fmt.Printf("This program is uable to cluster DNA in standalone mode\n")
		} else {
			if *verbose {
				fmt.Printf("magic: %s\n", mbuff)
			}
			fmt.Printf("%s %c%08x.%32.32x\n", filename, hh.ORDER_ALPHABET[order], ch32, voxel)
		}

	default:
		flag.Usage()
	}
}

func getMmappedBuffer(filename string) ([]byte, error) {
	// Open the file
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	// Memory map the file
	buffer, err := syscall.Mmap(
		int(file.Fd()),
		0,
		int(stat.Size()),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mmap file: %w", err)
	}

	return buffer, nil
}

type HollomanClient struct {
	client hh.HollomanClient
	conn   *grpc.ClientConn
}

// NewHollomanClient creates a new client instance
func NewHollomanClient(serverAddr string) (*HollomanClient, error) {
	// Set up connection with the server
	conn, err := grpc.Dial(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	client := hh .NewHollomanClient(conn)
	return &HollomanClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the client connection
func (c *HollomanClient) Close() error {
	return c.conn.Close()
}

// GetCapabilities calls the Capabilities RPC
func (c *HollomanClient) GetCapabilities(acceleration string, maxOrder int32) (*hh.ServiceCapabilities, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	capabilities := &hh.ServiceCapabilities{
		Acceleration: acceleration,
		MaxOrder:     maxOrder,
	}

	return c.client.Capabilities(ctx, capabilities)
}

// ClusterBuffer calls the ClusterBuffer RPC
func (c *HollomanClient) ClusterBuffer(buffer []byte, filename string) (*hh.BufferResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	request := &hh.BufferRequest{
		Buffer: buffer,
		Label: filename,
	}

	return c.client.ClusterBuffer(ctx, request)
}
