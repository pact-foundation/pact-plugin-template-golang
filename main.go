package main

import (
	"log"

	"github.com/google/uuid"
)

func main() {
	initLogging()

	port, err := GetFreePort()
	if err != nil {
		log.Fatal("ERROR unable to find a free port:", err)
	}

	// Start the Plugin Server
	// TODO: proper handling of startup/shutdown
	startPluginServer(serverDetails{
		Port:      port,
		ServerKey: uuid.NewString(),
	})
}
