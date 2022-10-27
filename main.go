package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/google/uuid"
	plugin "github.com/pact-foundation/example-plugin-go/io_pact_plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	initLogging()
	port, err := GetFreePort()
	if err != nil {
		log.Fatal("ERROR unable to find a free port:", err)
	}

	// TODO: proper handling of startup/shutdown
	startServer(serverDetails{
		Port:      port,
		ServerKey: uuid.NewString(),
	})
}

type serverDetails struct {
	Port      int
	ServerKey string
}

// The shape of the JSON object given to the pact test
type configuration struct {
	Request  configurationRequest
	Response configurationResponse
}

type configurationRequest struct {
	Body string
}

type configurationResponse struct {
	Body string
}

type pact struct {
	Consumer     string
	Provider     string
	Interactions []interaction
}

type interaction struct {
	Request  request
	Response []response
}

type request struct {
	Contents contents
}
type response struct {
	Contents contents
}
type contents struct {
	Content string
}

func initLogging() {
	dir, _ := os.Getwd()
	log.SetOutput(&lumberjack.Logger{
		Filename:   path.Join(dir, "log", "plugin.log"),
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	})
}

func startServer(details serverDetails) {
	log.Println("starting server on port", details.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", details.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf(`{"port": %d, "serverKey": "%s"}%s`, details.Port, details.ServerKey, "\n")

	var opts []grpc.ServerOption

	os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "99")
	os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "info")

	// TODO: options as flags?
	grpcServer := grpc.NewServer(opts...)
	plugin.RegisterPactPluginServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}

func newServer() *mattPluginServer {
	s := &mattPluginServer{}
	return s
}

type mattPluginServer struct {
	plugin.UnimplementedPactPluginServer
}

// // Check that the plugin loaded OK. Returns the catalogue entries describing what the plugin provides
func (m *mattPluginServer) InitPlugin(ctx context.Context, req *plugin.InitPluginRequest) (*plugin.InitPluginResponse, error) {
	log.Println("Received InitPlugin request:", req.Implementation, req.Version)

	return &plugin.InitPluginResponse{
		Catalogue: []*plugin.CatalogueEntry{
			{
				Key:  "matt",
				Type: plugin.CatalogueEntry_CONTENT_MATCHER,
				Values: map[string]string{
					"content-types": "text/matt;application/matt",
				},
			},
			{
				Key:  "matt",
				Type: plugin.CatalogueEntry_TRANSPORT,
			},
		},
	}, nil
}

// Request to configure/setup the interaction for later verification. Data returned will be persisted in the pact file.

// Validate the request
// Setup the pact interaction (including parsing matching rules and setting up generators)
func (m *mattPluginServer) ConfigureInteraction(ctx context.Context, req *plugin.ConfigureInteractionRequest) (*plugin.ConfigureInteractionResponse, error) {
	log.Println("Received ConfigureInteraction request:", req.ContentType, req.ContentsConfig)

	// req.ContentsConfig <- protobuf struct, equivalent to what can be represented in JSON

	// TODO: extract the actual request part and put into below
	config, err := protoStructToConfigMap(req.ContentsConfig)

	log.Println("Parsed ContentsConfig:", config.Request.Body, config.Response.Body)

	if err != nil {
		log.Println("ERROR unmarshalling ContentsConfig from JSON:", err)
	}

	var interactions = make([]*plugin.InteractionResponse, 0)
	if config.Request.Body != "" {
		interactions = append(interactions, &plugin.InteractionResponse{
			Contents: &plugin.Body{
				ContentType: "application/matt",
				Content:     wrapperspb.Bytes([]byte(generateMattMessage(config.Request.Body))),
			},
			PartName: "request",
		})
	}
	if config.Response.Body != "" {
		interactions = append(interactions, &plugin.InteractionResponse{
			Contents: &plugin.Body{
				ContentType: "application/matt",
				Content:     wrapperspb.Bytes([]byte(generateMattMessage(config.Response.Body))),
			},
			PartName: "response",
		})
	}

	return &plugin.ConfigureInteractionResponse{
		Interaction: interactions,
	}, nil
}

// Request to perform a comparison of some contents (matching request)
func (m *mattPluginServer) CompareContents(ctx context.Context, req *plugin.CompareContentsRequest) (*plugin.CompareContentsResponse, error) {
	log.Println("Received CompareContents request:", req)
	var mismatch string

	actual := parseMattMessage(string(req.Actual.Content.Value))
	expected := parseMattMessage(string(req.Expected.Content.Value))

	if actual != expected {
		mismatch = fmt.Sprintf("expected body '%s' is not equal to actual body '%s'", expected, actual)
		log.Println("Mismatch found:", mismatch)

		return &plugin.CompareContentsResponse{
			Results: map[string]*plugin.ContentMismatches{
				// "foo.bar.baz...." // hierarchical
				// "column:1" // tabular
				"$": {
					Mismatches: []*plugin.ContentMismatch{
						{
							Expected: wrapperspb.Bytes([]byte(expected)),
							Actual:   wrapperspb.Bytes([]byte(actual)),
							Mismatch: mismatch,
							Path:     "$",
						},
					},
				},
			},
		}, nil
	}

	return &plugin.CompareContentsResponse{}, nil

}

// Request to generate the content using any defined generators

// If there are no generators, this should just return back the given data
func (m *mattPluginServer) GenerateContent(ctx context.Context, req *plugin.GenerateContentRequest) (*plugin.GenerateContentResponse, error) {
	log.Println("Received GenerateContent request:", req.Contents, req.Generators, req.PluginConfiguration)

	var config configuration
	err := json.Unmarshal(req.Contents.Content.Value, &config)

	if err != nil {
		log.Println("ERROR:", err)
	}

	return &plugin.GenerateContentResponse{
		Contents: &plugin.Body{
			ContentType: "application/matt",
			Content:     wrapperspb.Bytes([]byte(generateMattMessage(config.Response.Body))),
		},
	}, nil

}

// Updated catalogue. This will be sent when the core catalogue has been updated (probably by a plugin loading).
func (m *mattPluginServer) UpdateCatalogue(ctx context.Context, cat *plugin.Catalogue) (*emptypb.Empty, error) {
	log.Println("Received UpdateCatalogue request:", cat.Catalogue)

	return &emptypb.Empty{}, nil

}

// Start a mock server
func (m *mattPluginServer) StartMockServer(ctx context.Context, req *plugin.StartMockServerRequest) (*plugin.StartMockServerResponse, error) {
	log.Println("Received StartMockServer request:", req)
	var err error
	port := int(req.Port)

	id := uuid.NewString()
	if port == 0 {
		port, err = GetFreePort()
		if err != nil {
			log.Println("ERROR unable to find a free port:", err)
			return &plugin.StartMockServerResponse{
				Response: &plugin.StartMockServerResponse_Error{
					Error: err.Error(),
				},
			}, err
		}
	}
	go startTCPServer(id, port)

	return &plugin.StartMockServerResponse{
		Response: &plugin.StartMockServerResponse_Details{
			Details: &plugin.MockServerDetails{
				Key:     id,
				Port:    uint32(port),
				Address: fmt.Sprintf("tcp://%s:%d", req.HostInterface, port),
			},
		},
	}, nil

	// TODO: parse the interactions and then store for future responses

}

// Shutdown a running mock server
func (m *mattPluginServer) ShutdownMockServer(ctx context.Context, req *plugin.ShutdownMockServerRequest) (*plugin.ShutdownMockServerResponse, error) {
	log.Println("Received ShutdownMockServer request:", req)

	ok := true
	err := stopTCPServer(req.ServerKey)
	if err != nil {
		ok = false
	}

	return &plugin.ShutdownMockServerResponse{ // duplicate / same info to GetMockServerResults
		Ok: ok,
		Results: []*plugin.MockServerResult{
			{
				Error: err.Error(),
			},
		},
	}, nil

}

// Get the matching results from a running mock server
func (m *mattPluginServer) GetMockServerResults(ctx context.Context, req *plugin.MockServerRequest) (*plugin.MockServerResults, error) {
	log.Println("Received GetMockServerResults request:", req)

	// TODO: error if server not called, or mismatches found
	return &plugin.MockServerResults{}, nil

}

// Prepare an interaction for verification. This should return any data required to construct any request
// so that it can be amended before the verification is run

// Example: authentication headers

var expectedMessage = ""

// If no modification is necessary, this should simply send the unmodified request back to the framework
// 2022/10/27 23:06:42 Received PrepareInteractionForVerification request: pact:"{\"consumer\":{\"name\":\"matttcpconsumer\"},\"interactions\":[{\"description\":\"Matt message\",\"key\":\"f27f2917655cb542\",\"pending\":false,\"request\":{\"contents\":{\"content\":\"MATThellotcpMATT\",\"contentType\":\"application/matt\",\"contentTypeHint\":\"DEFAULT\",\"encoded\":false}},\"response\":[{\"contents\":{\"content\":\"MATTtcpworldMATT\",\"contentType\":\"application/matt\",\"contentTypeHint\":\"DEFAULT\",\"encoded\":false}}],\"transport\":\"matt\",\"type\":\"Synchronous/Messages\"}],\"metadata\":{\"pactRust\":{\"ffi\":\"0.3.13\",\"mockserver\":\"0.9.4\",\"models\":\"0.4.5\"},\"pactSpecification\":{\"version\":\"4.0\"},\"plugins\":[{\"configuration\":{},\"name\":\"matt\",\"version\":\"0.0.1\"}]},\"provider\":{\"name\":\"matttcpprovider\"}}" interactionKey:"f27f2917655cb542" config:{fields:{key:"host" value:{string_value:"localhost"}} fields:{key:"port" value:{number_value:8444}}}
func (m *mattPluginServer) PrepareInteractionForVerification(ctx context.Context, req *plugin.VerificationPreparationRequest) (*plugin.VerificationPreparationResponse, error) {
	log.Println("Received PrepareInteractionForVerification request:", req)

	var p pact
	err := json.Unmarshal([]byte(req.Pact), &p)
	if err != nil {
		log.Println("ERROR extracting payload for verification:", err)
	}

	expectedMessage = parseMattMessage(p.Interactions[0].Response[0].Contents.Content)

	return &plugin.VerificationPreparationResponse{
		Response: &plugin.VerificationPreparationResponse_InteractionData{
			InteractionData: &plugin.InteractionData{
				Body: &plugin.Body{
					ContentType: "application/matt",
					Content:     wrapperspb.Bytes([]byte(generateMattMessage(expectedMessage))), // <- TODO: this needs to come from the pact struct
				},
			},
		},
	}, nil

}

// Execute the verification for the interaction.
func (m *mattPluginServer) VerifyInteraction(ctx context.Context, req *plugin.VerifyInteractionRequest) (*plugin.VerifyInteractionResponse, error) {
	log.Println("Received VerifyInteraction request:", req)

	// expected := parseMattMessage(string(req.InteractionData.Body.Content.Value))

	// Issue the call to the provider
	host := req.Config.AsMap()["host"].(string)
	port := req.Config.AsMap()["port"].(float64)
	log.Println("Calling TCP service at host", host, "and port", port)
	actual, err := callMattServiceTCP(host, int(port), expectedMessage)
	log.Println("Received:", actual, "wanted:", expectedMessage, "err:", err)

	// Report on the results
	if actual != expectedMessage {
		return &plugin.VerifyInteractionResponse{
			Response: &plugin.VerifyInteractionResponse_Result{
				Result: &plugin.VerificationResult{
					Success: false,
					Output:  []string{""},
					Mismatches: []*plugin.VerificationResultItem{
						{
							Result: &plugin.VerificationResultItem_Mismatch{
								Mismatch: &plugin.ContentMismatch{
									Expected: wrapperspb.Bytes([]byte(expectedMessage)),
									Actual:   wrapperspb.Bytes([]byte(actual)),
									Path:     "$",
									Mismatch: fmt.Sprintf("Expected '%s' but got '%s'", expectedMessage, actual),
								},
							},
						},
					},
				},
			},
		}, nil
	}

	return &plugin.VerifyInteractionResponse{
		Response: &plugin.VerifyInteractionResponse_Result{
			Result: &plugin.VerificationResult{
				Success: true,
			},
		},
	}, nil

}

func protoStructToConfigMap(s *structpb.Struct) (configuration, error) {
	var config configuration
	bytes, err := s.MarshalJSON()

	if err != nil {
		log.Println("ERROR marshalling ContentsConfig to JSON:", err)
		return config, nil
	}

	err = json.Unmarshal(bytes, &config)

	if err != nil {
		log.Println("ERROR unmarshalling ContentsConfig from JSON:", err)
		return config, nil
	}

	return config, nil
}

var servers = map[string]net.Listener{}

func startTCPServer(id string, port int) {
	log.Println("Starting TCP server", id, "on port", port)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Println("ERROR:", err)
	}
	servers[id] = listener
	log.Println("TCP server started", id, "on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("TCP connection error:", err)
			continue
		}

		log.Println("TCP connection established with:", conn.RemoteAddr())

		go handleConnection(conn)
	}
}

func stopTCPServer(id string) error {
	// TODO: properly handle this
	return servers[id].Close()
}

func handleConnection(conn net.Conn) {
	log.Println("Handling TCP connection")
	defer conn.Close()

	s := bufio.NewScanner(conn)

	for s.Scan() {

		data := s.Text()
		log.Println("Data received from connection", data)

		if data == "" {
			continue
		}

		handleRequest(data, conn)
	}
}

func handleRequest(req string, conn net.Conn) {
	log.Println("TCP Server received request", req, "on connection", conn)

	if !isValidMessage(req) {
		log.Println("TCP Server received invalid request, erroring")
		conn.Write([]byte("ERROR\n"))
	}
	log.Println("TCP Server received valid request, responding")

	// TODO: this should come from the original request
	var expectedResponse = "tcpworld"
	conn.Write([]byte(generateMattMessage(expectedResponse)))
	conn.Write([]byte("\n"))
}

func generateMattMessage(message string) string {
	return fmt.Sprintf("MATT%sMATT", message)
}

func parseMattMessage(message string) string {
	return strings.TrimSpace(strings.ReplaceAll(message, "MATT", ""))
}

func isValidMessage(str string) bool {
	matched, err := regexp.MatchString(`^MATT.*MATT$`, str)
	if err != nil {
		return false
	}

	return matched
}

func callMattServiceTCP(host string, port int, message string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return "", err
	}

	conn.Write([]byte(generateMattMessage(message)))
	conn.Write([]byte("\n"))

	str, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil {
		return "", err
	}

	return parseMattMessage(str), nil
}

// GetFreePort Gets an available port by asking the kernal for a random port
// ready and available for use.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
