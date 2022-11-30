package main

import (
	"encoding/json"
	"log"

	"google.golang.org/protobuf/types/known/structpb"
)

// TODO: customise me to your needs

// This struct maps to the shape of the contents given to the pact test
// We default to using a JSON structure here but it can be whatever you like
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

// Converts a protobuf Struct (essentially an arbitrary structure)
// to a configuration item.
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
