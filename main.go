package main

import (
	"log"

	"github.com/google/uuid"
)

// This file contains the plugin entrypoint
// You probably don't need to change this

func main() {
	initLogging()

	port, err := GetFreePort()
	if err != nil {
		log.Fatal("ERROR unable to find a free port:", err)
	}

	// Start the Plugin Server
	startPluginServer(serverDetails{
		Port:      port,
		ServerKey: uuid.NewString(),
	})
}
