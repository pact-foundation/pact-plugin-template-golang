package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/google/uuid"
	plugin "github.com/mefellows/pact-matt-plugin/io_pact_plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

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

func startPluginServer(details serverDetails) {
	log.Println("starting server on port", details.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", details.Port))
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

	log.Println("Parsed ContentsConfig:", config.Request.Body, config.Response.Body, err)

	if err != nil {
		log.Println("ERROR unmarshalling ContentsConfig from JSON:", err)
		return &plugin.ConfigureInteractionResponse{
			Error: err.Error(),
		}, nil
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

	err := stopTCPServer(req.ServerKey)
	if err != nil {
		return &plugin.ShutdownMockServerResponse{ // duplicate / same info to GetMockServerResults
			Ok: false,
			Results: []*plugin.MockServerResult{
				{
					Error: err.Error(),
				},
			},
		}, nil
	}

	return &plugin.ShutdownMockServerResponse{ // duplicate / same info to GetMockServerResults
		Ok:      true,
		Results: []*plugin.MockServerResult{},
	}, nil

}

// Get the matching results from a running mock server
func (m *mattPluginServer) GetMockServerResults(ctx context.Context, req *plugin.MockServerRequest) (*plugin.MockServerResults, error) {
	log.Println("Received GetMockServerResults request:", req)

	// TODO: error if server not called, or mismatches found
	return &plugin.MockServerResults{}, nil

}

var expectedMessage = ""

// Prepare an interaction for verification. This should return any data required to construct any request
// so that it can be amended before the verification is run
// Example: authentication headers
// If no modification is necessary, this should simply send the unmodified request back to the framework
func (m *mattPluginServer) PrepareInteractionForVerification(ctx context.Context, req *plugin.VerificationPreparationRequest) (*plugin.VerificationPreparationResponse, error) {
	// 2022/10/27 23:06:42 Received PrepareInteractionForVerification request: pact:"{\"consumer\":{\"name\":\"matttcpconsumer\"},\"interactions\":[{\"description\":\"Matt message\",\"key\":\"f27f2917655cb542\",\"pending\":false,\"request\":{\"contents\":{\"content\":\"MATThellotcpMATT\",\"contentType\":\"application/matt\",\"contentTypeHint\":\"DEFAULT\",\"encoded\":false}},\"response\":[{\"contents\":{\"content\":\"MATTtcpworldMATT\",\"contentType\":\"application/matt\",\"contentTypeHint\":\"DEFAULT\",\"encoded\":false}}],\"transport\":\"matt\",\"type\":\"Synchronous/Messages\"}],\"metadata\":{\"pactRust\":{\"ffi\":\"0.3.13\",\"mockserver\":\"0.9.4\",\"models\":\"0.4.5\"},\"pactSpecification\":{\"version\":\"4.0\"},\"plugins\":[{\"configuration\":{},\"name\":\"matt\",\"version\":\"0.0.1\"}]},\"provider\":{\"name\":\"matttcpprovider\"}}" interactionKey:"f27f2917655cb542" config:{fields:{key:"host" value:{string_value:"localhost"}} fields:{key:"port" value:{number_value:8444}}}
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
	port := l.Addr().(*net.TCPAddr).Port
	defer l.Close()
	return port, nil
}
