package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

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
	log.Println("Shutting down TCP Server")

	// TODO: properly handle this, and send a signal to the handler to stop listening
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
