package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type pactv4 struct {
	Consumer        application
	Provider        application
	Interactions    []interface{}     `json:"-"` // This is polymorphic, so we need a custom marshaller
	RawInteractions []json.RawMessage `json:"interactions"`
}

func (p *pactv4) UnmarshalJSON(b []byte) error {
	type pact pactv4
	err := json.Unmarshal(b, (*pact)(p))
	if err != nil {
		return err
	}

	for _, raw := range p.RawInteractions {
		var v interaction
		err = json.Unmarshal(raw, &v)
		if err != nil {
			return err
		}
		var i interface{}
		switch v.Type {
		case "Asynchronous/Messages":
			i = &asyncMessageInteraction{}
		case "Synchronous/Messages":
			i = &syncMessageInteraction{}
		case "Synchronous/HTTP":
			i = &httpInteraction{}
		default:
			return fmt.Errorf("unknown interaction type: '%s'", v.Type)
		}

		log.Println("identified narrow type:", i)
		err = json.Unmarshal(raw, i)
		if err != nil {
			return err
		}
		log.Println("unmarshalled into narrow type:", i)
		p.Interactions = append(p.Interactions, i)
	}
	return nil
}

type interaction struct {
	Type string
	Key  string
}

type application struct {
	Name string
}

type httpInteraction struct {
	interaction
	Request  httpRequest
	Response httpResponse
}

type syncMessageInteraction struct {
	interaction
	Request  messageRequest
	Response []syncMessageResponse
}

type asyncMessageInteraction struct {
	interaction
	Contents contents
}

type messageRequest struct {
	Contents contents
}

type httpResponse struct {
	Body bodyContent
}

// NOTE: only mapping parts of the spec required. Excluding headers, query etc.
//       If you need additional fields please update and submit a PR
type httpRequest struct {
	Body bodyContent
}

type syncMessageResponse struct {
	Contents contents
}

type contents struct {
	Content string
}

type bodyContent struct {
	Content         string // TODO: should be interface{} ?
	ContentType     string
	ContentTypeHint string
	Encoded         bool
}
