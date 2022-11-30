package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	plugin "github.com/pact-foundation/pact-plugin-template-golang/io_pact_plugin"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var CONTENT_TYPE = "application/foo"

///////////////////////////
/// Common RPC functions //
///////////////////////////

// Initialise the plugin, and registers the plugin capabilities with the plugin catalogue
//
// When a plugin loads, it will receive an InitPluginRequest and must respond with an InitPluginResponse with the catalogue entries for the plugin. See the plugin proto file for details of these messages.
//
// For a content matcher or generator, the entries need to be:
//
// Entry Type
// The entry type must be set to CONTENT_MATCHER or CONTENT_GENERATOR.
//
// Entry Key
// This should be the name of the content type. If there is only one content matcher/generator for the plugin, it can be the plugin name.
//
// Associated values
// The values for the entry must contain a content-types key that contains all the content types the matcher or generator supports. If there are multiple content types, they must be separated with a semi-colon (;).
//
// For example, for the CSV plugin we return the following two entries:
// Docs: https://github.com/pact-foundation/pact-plugins/blob/main/docs/content-matcher-design.md#plugin-catalogue-entries-for-content-matchers-and-generators
func (m *pluginServer) InitPlugin(ctx context.Context, req *plugin.InitPluginRequest) (*plugin.InitPluginResponse, error) {
	log.Println("Received InitPlugin request:", req.Implementation, req.Version)

	// TODO: update this as required
	// NOTE: Not all plugins will implement both a CONTENT_MATCHER and TRANSPORT
	return &plugin.InitPluginResponse{
		Catalogue: []*plugin.CatalogueEntry{
			{
				Key:  "PROJECT NAME",                        // TODO: changeme!
				Type: plugin.CatalogueEntry_CONTENT_MATCHER, // TODO: changeme!
				Values: map[string]string{
					"content-types": CONTENT_TYPE,
				},
			},
			{
				Key:  "TRANSPORT NAME", // TODO: changeme!
				Type: plugin.CatalogueEntry_TRANSPORT,
			},
		},
	}, nil
}

// This will be sent when the core catalogue has been updated (probably by a plugin loading).
//
// This method is currently not implemented by the driver, and is reserved for future use
// (e.g. for discovering and communicating with other plugins)
func (m *pluginServer) UpdateCatalogue(ctx context.Context, cat *plugin.Catalogue) (*emptypb.Empty, error) {
	log.Println("Received UpdateCatalogue request:", cat.Catalogue)

	return &emptypb.Empty{}, nil
}

////////////////////////////
/// Content RPC functions //
////////////////////////////

// In the consumer test, the first thing it will do is send though an ConfigureInteractionRequest
// containing the content type and the data the user configured in the test for the interaction.
// The plugin needs to consume this data, and then return the data required to configure the interaction
// (which includes the body, matching rules, generators and additional data that needs to be persisted
// to the pact file).
//
// Docs: https://github.com/pact-foundation/pact-plugins/blob/main/docs/content-matcher-design.md#responding-to-match-contents-requests
func (m *pluginServer) ConfigureInteraction(ctx context.Context, req *plugin.ConfigureInteractionRequest) (*plugin.ConfigureInteractionResponse, error) {
	log.Println("Received ConfigureInteraction request:", req.ContentType, req.ContentsConfig)

	// Extract the incoming plugin configuration from and validate it
	// Remember - this structure is whatever you designed for your consumer interface
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
				ContentType: CONTENT_TYPE,
				Content:     wrapperspb.Bytes([]byte(config.Request.Body)), // <- ensure format is correct
			},
			PartName: "request",
		})
	}
	if config.Response.Body != "" {
		interactions = append(interactions, &plugin.InteractionResponse{
			Contents: &plugin.Body{
				ContentType: CONTENT_TYPE,
				Content:     wrapperspb.Bytes([]byte(config.Response.Body)), // <- ensure format is correct
			},
			PartName: "response",
		})
	}

	return &plugin.ConfigureInteractionResponse{
		Interaction: interactions,
	}, nil
}

// Now that the interaction has been configured, everytime the Pact mock
// server (consumer side) or verifier (provider side) encounters a content
// type associated with the plugin, the plugin will receive a
// CompareContentsRequest request and must respond with a CompareContentsResponse
// with the match results of the contents.
//
// Docs: https://github.com/pact-foundation/pact-plugins/blob/main/docs/content-matcher-design.md#match-content-requests
func (m *pluginServer) CompareContents(ctx context.Context, req *plugin.CompareContentsRequest) (*plugin.CompareContentsResponse, error) {
	log.Println("Received CompareContents request:", req)
	var mismatch string

	// Extract the actual and expected values (given as an array of bytes)
	// This is where you will need to convert and parse the protocol specific
	// information
	actual := string(req.Actual.Content.Value)
	expected := string(req.Expected.Content.Value)

	// Perform the matching logic.
	// Here we are simply checking exact values
	if compare(actual, expected) {
		mismatch = fmt.Sprintf("expected body '%s' is not equal to actual body '%s'", expected, actual)
		log.Println("Mismatch found:", mismatch)

		return &plugin.CompareContentsResponse{
			Results: map[string]*plugin.ContentMismatches{
				"$": { // <- path where the content is matched.

					// The path can be denoted however you wish. Pact uses a JSON Path-like syntax
					//
					// Examples:
					//
					//   hierarchical => "$.foo.bar.baz...."
					//   tabular =>      "column:1"

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
//
// Every time the Pact implementation needs to generate contents for a
// content associated with a plugin, it will send a GenerateContentRequest
// to the plugin. This will happen in consumer tests when the mock server
// needs to generate a response or the contents of a message, or during
// verification of the provider when the verifier needs to send a request
// to the provider.
//
// Docs: https://github.com/pact-foundation/pact-plugins/blob/main/docs/content-matcher-design.md#responding-to-generate-contents-requests
func (m *pluginServer) GenerateContent(ctx context.Context, req *plugin.GenerateContentRequest) (*plugin.GenerateContentResponse, error) {
	log.Println("Received GenerateContent request:", req.Contents, req.Generators, req.PluginConfiguration)

	// Read in the Pact test configuration
	// This is what is used in the test DSL to specify the plugin specific
	// interaction details, such as request/response content
	var config configuration
	err := json.Unmarshal(req.Contents.Content.Value, &config)

	if err != nil {
		log.Println("ERROR:", err)
	}

	// Extract the portion of the message (request/response), and return the content as it will be sent
	// over the wire
	return &plugin.GenerateContentResponse{
		Contents: &plugin.Body{
			ContentType: CONTENT_TYPE,
			Content:     wrapperspb.Bytes([]byte(config.Response.Body)),
		},
	}, nil

}

///////////////////////////////////////
/// Transport protocol RPC functions //
///////////////////////////////////////

// Start a mock server
func (m *pluginServer) StartMockServer(ctx context.Context, req *plugin.StartMockServerRequest) (*plugin.StartMockServerResponse, error) {
	log.Println("Received StartMockServer request:", req)
	var err error

	// The runtime may specify a port to start your server on
	port := int(req.Port)

	// Your server needs a unique id so that you can keep track of it
	// The pact plugin framework may start several servers, so you
	// may need to track each server separately (a task left up to the author)
	id := uuid.NewString()
	log.Println("Creating a new server with id:", id)

	// If a port hasn't been specified, find a free one
	// Return an error if one can't be allocated
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

	// TODO: start your server e.g.
	// go startTCPServer(id, port)

	// Populate the return message with your mock server details
	// return &plugin.StartMockServerResponse{
	// 	Response: &plugin.StartMockServerResponse_Details{
	// 		Details: &plugin.MockServerDetails{
	// 			Key:     id,
	// 			Port:    uint32(port),
	// 			Address: fmt.Sprintf("tcp://%s:%d", req.HostInterface, port),
	// 		},
	// 	},
	// }, nil

	return nil, nil
}

// Shutdown a running mock server
func (m *pluginServer) ShutdownMockServer(ctx context.Context, req *plugin.ShutdownMockServerRequest) (*plugin.ShutdownMockServerResponse, error) {
	log.Println("Received ShutdownMockServer request:", req)
	var err error

	// Locate your server, and shut it down e.g.
	// err := stopMyServer(req.ServerKey)

	if err != nil {
		return &plugin.ShutdownMockServerResponse{
			Ok: false,
			Results: []*plugin.MockServerResult{
				{
					Error: err.Error(),
				},
			},
		}, nil
	}

	return &plugin.ShutdownMockServerResponse{
		Ok:      true,
		Results: []*plugin.MockServerResult{},
	}, nil

}

// Get the matching results from a running mock server
func (m *pluginServer) GetMockServerResults(ctx context.Context, req *plugin.MockServerRequest) (*plugin.MockServerResults, error) {
	log.Println("Received GetMockServerResults request:", req)

	// TODO: error if server not called, or mismatches found

	return &plugin.MockServerResults{}, nil
}

var requestMessage = ""
var responseMessage = ""

// Prepare an interaction for verification. This should return any data required to construct any request
// so that it can be amended before the verification is run e.g. auth headers
// If no modification is necessary, this should simply send the unmodified request back to the framework
func (m *pluginServer) PrepareInteractionForVerification(ctx context.Context, req *plugin.VerificationPreparationRequest) (*plugin.VerificationPreparationResponse, error) {
	// 2022/10/27 23:06:42 Received PrepareInteractionForVerification request: pact:"{\"consumer\":{\"name\":\"matttcpconsumer\"},\"interactions\":[{\"description\":\"Matt message\",\"key\":\"f27f2917655cb542\",\"pending\":false,\"request\":{\"contents\":{\"content\":\"MATThellotcpMATT\",\"contentType\":\"application/matt\",\"contentTypeHint\":\"DEFAULT\",\"encoded\":false}},\"response\":[{\"contents\":{\"content\":\"MATTtcpworldMATT\",\"contentType\":\"application/matt\",\"contentTypeHint\":\"DEFAULT\",\"encoded\":false}}],\"transport\":\"matt\",\"type\":\"Synchronous/Messages\"}],\"metadata\":{\"pactRust\":{\"ffi\":\"0.3.13\",\"mockserver\":\"0.9.4\",\"models\":\"0.4.5\"},\"pactSpecification\":{\"version\":\"4.0\"},\"plugins\":[{\"configuration\":{},\"name\":\"matt\",\"version\":\"0.0.1\"}]},\"provider\":{\"name\":\"matttcpprovider\"}}" interactionKey:"f27f2917655cb542" config:{fields:{key:"host" value:{string_value:"localhost"}} fields:{key:"port" value:{number_value:8444}}}
	log.Println("Received PrepareInteractionForVerification request:", req)

	// Get a handle to the incoming Pact file and parse it
	var p pactv4
	err := json.Unmarshal([]byte(req.Pact), &p)
	if err != nil {
		log.Println("ERROR extracting payload for verification:", err)
	}

	// Find the current interaction in the Pact and extract the request and (if needed) response payloads
	// You'll need to keep track of these for later when verifying the interaction was correct
	for _, inter := range p.Interactions {
		log.Println("finding interaction by key", req.InteractionKey)

		switch i := inter.(type) {
		case *httpInteraction:
			log.Println("comparing keys", i.interaction.Key, req.InteractionKey)
			if i.Key == req.InteractionKey {
				log.Println("found HTTP interaction")
				requestMessage = i.Request.Body.Content
				responseMessage = i.Response.Body.Content
			}
		case *asyncMessageInteraction:
			log.Println("comparing keys", i.interaction.Key, req.InteractionKey)
			if i.Key == req.InteractionKey {
				log.Println("found async interaction")
				requestMessage = i.Contents.Content
			}
		case *syncMessageInteraction:
			log.Println("comparing keys", i.interaction.Key, req.InteractionKey)
			if i.Key == req.InteractionKey {
				log.Println("found sync interaction")
				requestMessage = i.Request.Contents.Content
				responseMessage = i.Response[0].Contents.Content
			}
		default:
			log.Printf("unknown interaction type: '%+v'", i)
		}
	}
	log.Println("found request body:", requestMessage)   // <- This gets sent back to the framework
	log.Println("found response body:", responseMessage) // <- This gets stored later for use in VerifyInteraction ()

	// We need to return the request message,
	return &plugin.VerificationPreparationResponse{
		Response: &plugin.VerificationPreparationResponse_InteractionData{
			InteractionData: &plugin.InteractionData{
				Body: &plugin.Body{
					ContentType: CONTENT_TYPE,
					Content:     wrapperspb.Bytes([]byte(requestMessage)), // <- TODO: ensure the right format
				},
			},
		},
	}, nil

}

// Execute the verification for the interaction.
// TODO: delete this method if you are not providing a transport for your plugin
func (m *pluginServer) VerifyInteraction(ctx context.Context, req *plugin.VerifyInteractionRequest) (*plugin.VerifyInteractionResponse, error) {
	log.Println("Received VerifyInteraction request:", req)

	actual := ""
	err := errors.New("VerifyInteraction is unimplemented")

	// Issue the call to the provider
	// These two keys should reliably exist in this structure
	host := req.Config.AsMap()["host"].(string)
	port := req.Config.AsMap()["port"].(float64)

	// Issue call
	log.Println("Calling PROJECT_NAME mock service at host", host, "and port", port)
	// e.g. actual, err := callTestProviderAPI(host, int(port), requestMessage)
	log.Println("Received:", actual, "wanted:", responseMessage, "err:", err)

	// Error invoking the API
	if err != nil {
		return &plugin.VerifyInteractionResponse{
			Response: &plugin.VerifyInteractionResponse_Result{
				Result: &plugin.VerificationResult{
					Success: false,
					Output:  []string{"Error communicating to the provider API"},
					Mismatches: []*plugin.VerificationResultItem{
						{
							Result: &plugin.VerificationResultItem_Mismatch{
								Mismatch: &plugin.ContentMismatch{
									Path:     "$",                                                 // <- ensure path is correct
									Mismatch: fmt.Sprintf("Error invoking provider API: %s", err), // <- human readible error
								},
							},
						},
					},
				},
			},
		}, nil
	}

	// Compare the expected vs actual, and print any errors
	// this logic is likely insufficient, but it should help illuminate the point
	if compare(actual, responseMessage) {
		return &plugin.VerifyInteractionResponse{
			Response: &plugin.VerifyInteractionResponse_Result{
				Result: &plugin.VerificationResult{
					Success: false,
					Output:  []string{""},
					Mismatches: []*plugin.VerificationResultItem{
						{
							Result: &plugin.VerificationResultItem_Mismatch{
								Mismatch: &plugin.ContentMismatch{
									Expected: wrapperspb.Bytes([]byte(responseMessage)),
									Actual:   wrapperspb.Bytes([]byte(actual)),
									Path:     "$",                                                                // <- ensure path is correct
									Mismatch: fmt.Sprintf("Expected '%s' but got '%s'", responseMessage, actual), // <- human readible error
								},
							},
						},
					},
				},
			},
		}, nil
	}

	// Everything was OK
	return &plugin.VerifyInteractionResponse{
		Response: &plugin.VerifyInteractionResponse_Result{
			Result: &plugin.VerificationResult{
				Success: true,
			},
		},
	}, nil
}

// TODO: review signature and return errors
func compare(actual string, expected string) bool {
	return actual != expected
}
