package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	CONN_HOST    = "127.0.0.1"
	CONN_PORT    = "12053"
	CONN_ADDRESS = CONN_HOST + ":" + CONN_PORT
	CONN_TYPE    = "tcp"
)

func main() {
	// Create listener
	ln, err := net.Listen(CONN_TYPE, CONN_ADDRESS)
	if err != nil {
		fmt.Println("Error creating listener:", err.Error())
		os.Exit(1)
	}
	defer ln.Close()

	// Listen for incoming connections
	fmt.Println("Listening on " + CONN_ADDRESS)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			os.Exit(1)
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	// Read request
	scanner := bufio.NewScanner(conn)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var response string

	// first line is the command
	tokens := strings.Split(lines[0], " ")
	command := tokens[0]
	switch command {
	case "convert":
		n := len(tokens) - 1
		if n == 4 {
			convertIdentifier(tokens[1], tokens[2], tokens[3], tokens[4])
		} else if n == 3 {
			convertIdentifier(tokens[1], tokens[2], tokens[1], tokens[3])
		} else {
			response = "Error, expecting: convert <srcSpecies> <srcId> <destSpecies> <destId>\n"
		}
	case "unload":
		unloadDatabases()
		response = "Error: Unloaded databases.\n"
	case "help":
		response = "convert <srcSpecies> <srcId> <destSpecies> <destId>\n" +
			"convert <srcSpecies> <srcId> <destId>\n" +
			"unload\n" +
			"help\n"
	default:
		response = "Error: Unknown command.\n"
	}

	// Send response back
	conn.Write([]byte(response))
	conn.Close()

	// log error instead of sending response back
}

func convertIdentifier(srcSpecies, srcId, destSpecies, destId string) {

}

func unloadDatabases() {

}
