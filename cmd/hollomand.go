package main
//
// Copyright 2025 (c) By Rick Wesson & Support Intelligence, Inc.
// Licenced under the RLL 1.0

// go:generate  protoc protoc --go_out=.  --go-grpc_out=. holloman.proto
import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
  _ "embed"

	"github.com/OneOfOne/xxhash"
	"github.com/hosom/gomagic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"main/iidf/holloman"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	// "google.golang.org/grpc/metadata"
	// "google.golang.org/grpc/peer"
	// "google.golang.org/grpc/reflection"

	"github.com/glaslos/ssdeep"
)

const ()

var (
	curveFile 	string
	damonize  	*bool
	dna       	*bool
	location  	string
	ep        	string // execution pattern (client, server, stand_alone)
	filename  	string
	verbose   	*bool
	ssdf		*bool

	// go:embed LICENCE.txt
	LICENCE		[]byte
)

type HollomanServer struct {
	holloman.HollomanServer

	curve *holloman.HilbertCurve
	m     *magic.Magic
}

func NewServer(curve *holloman.HilbertCurve, dna bool) (s *HollomanServer, err error) {

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

func init() {


	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	flag.StringVar(&curveFile, "curve", "hilbert_curve.dat.gz", "pre-generated hilbert curve (gzip compressed)")
	flag.StringVar(&location, "listen", ":50051", "location to listen :port or /path/to/unix.socket")
	flag.StringVar(&filename, "f", "", "file to generate an identifier for")

	server := flag.Bool("S", false, "Server")
	client := flag.Bool("C", false, "Client")
	help := flag.Bool("h", false, "help")
	ssdf = flag.Bool("ssdeep", false, "enable ssdeep results")
	dna = flag.Bool("dna", false, "the server should only be used for DNA clustering")
	verbose = flag.Bool("v", false, "verbose")

	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *server == false && *client == false {
		ep = "stand_alone"
	} else if *server {
		ep = "server"
	} else if *client {
		ep = "client"
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

func client() {
	// Create a new client
	client, err := NewHollomanClient(location)
	if err != nil {
		log.Fatal().Msgf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Example: Get capabilities
	capabilities, err := client.GetCapabilities("fast", 5)
	if err != nil {
		log.Fatal().Msgf("Failed to get capabilities: %v", err)
	}
	log.Info().Msgf("Capabilities received: %v", capabilities)

	// Example: Cluster buffer
	buffer, err := getMmappedBuffer(filename)
	if err != nil {
		log.Fatal().Msgf(err.Error())
	}
	defer syscall.Munmap(buffer)
	//sampleBuffer := []byte("sample data")
	response, err := client.ClusterBuffer(buffer)
	if err != nil {
		log.Fatal().Msgf("Failed to cluster buffer: %v", err)
	}
	log.Info().Msgf("Cluster response received: HOrder=%d, Id=%s, Magic=%s",
		response.HOrder, string(response.Id), response.Magic)

}

func server(srvr *HollomanServer) {
	lis, err := net.Listen("tcp", location)
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(1024*10e7), grpc.MaxSendMsgSize(1024*10e7), withServerUnaryInterceptor())
	holloman.RegisterHollomanServer(s, srvr)

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
func (server *HollomanServer) Capabilities(context.Context, *holloman.ServiceCapabilities) (*holloman.ServiceCapabilities, error) {
	cah := new(holloman.ServiceCapabilities)
	cah.Acceleration = "none"
	cah.MaxOrder = int32(server.curve.Order)
	
	cah.Magic="filemagic"
	if *dna {
		cah.Magic="dna/iching"
	}
	cah.Ssdeep=false
	if(*ssdf){
		cah.Ssdeep=true
	}

	return cah, nil
}

func (server *HollomanServer) ClusterBuffer(ctx context.Context, req *holloman.BufferRequest) (br *holloman.BufferResponse, err error) {

	var voxel []byte
	br = new(holloman.BufferResponse)
	voxel, br.HOrder, err = server.curve.MapBuffer(req.Buffer)
	if *dna {
		br.Magic = "dna/iching"
		br.Id = []byte(fmt.Sprintf("%c.%032x", holloman.ORDER_ALPHABET[br.HOrder], voxel))
	} else {
		br.Magic, err  = server.m.Buffer(req.Buffer)
		_ = err
		ch32 := xxhash.ChecksumString32(fmt.Sprintf("%-60.60s", br.Magic))
		br.Id = []byte(fmt.Sprintf("%c%08x.%32.32x", holloman.ORDER_ALPHABET[br.HOrder], ch32, voxel))
	}

	if *ssdf {
		//preform ssdeep hash on buffer
		s, err := ssdeep.FuzzyBytes(req.Buffer)
		if err != nil {
			br.Ssdeep = err.Error()
		}
		br.Ssdeep = s
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
		curve, err := holloman.LoadHilbertCurve(curveFile)
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

	case "client":
		client()

	case "stand_alone":
		//read the compressed curve
		curve, err := holloman.LoadHilbertCurve(curveFile)
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
		ch32 := xxhash.ChecksumString32(fmt.Sprintf("%-60.60s", mbuff))

		if *dna {
			fmt.Printf("This program is uable to cluster DNA in standalone mode\n")
		} else {
			if *verbose {
				fmt.Printf("magic: %s\n", mbuff)
			}
			fmt.Printf("%s %c%08x.%32.32x\n", filename, holloman.ORDER_ALPHABET[order], ch32, voxel)
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
	client holloman.HollomanClient
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

	client := holloman.NewHollomanClient(conn)
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
func (c *HollomanClient) GetCapabilities(acceleration string, maxOrder int32) (*holloman.ServiceCapabilities, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	capabilities := &holloman.ServiceCapabilities{
		Acceleration: acceleration,
		MaxOrder:     maxOrder,
	}

	return c.client.Capabilities(ctx, capabilities)
}

// ClusterBuffer calls the ClusterBuffer RPC
func (c *HollomanClient) ClusterBuffer(buffer []byte) (*holloman.BufferResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	request := &holloman.BufferRequest{
		Buffer: buffer,
	}

	return c.client.ClusterBuffer(ctx, request)
}
