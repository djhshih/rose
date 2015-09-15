package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

var (
	connHost    string
	connPort    string
	connAddress string
	inputFile   string
	showHelp    bool
)

var (
	table  string
	srcId  string
	destId string
)

var (
	elog *log.Logger
)

const (
	programVersion = "0.1"
	protocolName   = "ROSE/0.1"
)

const (
	entryDelim    = "\n"
	commentPrefix = "#"
)

func init() {
	flag.StringVar(&connHost, "host", "127.0.0.1", "rosed address for connection")
	flag.StringVar(&connPort, "port", "12053", "rosed port for connection")

	flag.StringVar(&inputFile, "input", "-", "input file of identifiers [default: stdin]")

	flag.BoolVar(&showHelp, "help", false, "show help message")
}

func main() {

	elog = log.New(os.Stderr, "Error: ", 0)

	flag.Parse()
	connAddress = connHost + ":" + connPort

	if showHelp {
		fmt.Println("rose - Identifier conversion client (version " + programVersion + ")")
		fmt.Println()

		fmt.Println("Usage: rose [options] <table> <srcId> <destId>")
		fmt.Println()

		fmt.Println("Required arguments:")
		fmt.Println("   table   name of mapping table")
		fmt.Println("   srcId   source identifier type from which to convert")
		fmt.Println("   destId  destination identifier type to which to convert")
		fmt.Println()

		fmt.Println("Optional arguments:")
		flag.PrintDefaults()

		return
	}

	if flag.NArg() < 3 {
		elog.Fatal("Required arguments are missing")
	} else {
		table = flag.Arg(0)
		srcId = flag.Arg(1)
		destId = flag.Arg(2)
	}

	// Read input file
	var inputIds []string
	if inputFile == "-" {
		inputIds = readInput(os.Stdin)
	} else {
		f, err := os.Open(inputFile)
		if err != nil {
			elog.Fatal("Cannot read input file", inputFile)
		} else {
			defer f.Close()
			inputIds = readInput(f)
		}
	}

	// Connect to server
	raddr, _ := net.ResolveTCPAddr("tcp", connAddress)
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		elog.Fatal("Cannot connect to server:", err)
	}
	defer conn.Close()

	// Send request to server
	request := fmt.Sprintf("map %s %s %s", table, srcId, destId)
	request += entryDelim + strings.Join(inputIds, entryDelim)
	conn.Write([]byte(request))
	conn.CloseWrite()

	// Read response from server
	scanner := bufio.NewScanner(conn)
	// check header
	if scanner.Scan() {
		tokens := strings.Split(scanner.Text(), " ")
		if len(tokens) < 3 || tokens[0] != protocolName {
			elog.Println("Unrecognized protocol")
		} else if tokens[1] != "200" {
			elog.Println("Server response:", tokens[1], strings.Join(tokens[2:], " "))
		} else {
			// print identifiers to stdout
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}
	}

}

func readInput(r io.Reader) (lines []string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, commentPrefix) {
			// ignore comments
		} else {
			lines = append(lines, line)
		}
	}
	return
}
