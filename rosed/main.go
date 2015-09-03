package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	CONN_HOST    = "127.0.0.1"
	CONN_PORT    = "12053"
	CONN_ADDRESS = CONN_HOST + ":" + CONN_PORT
	CONN_TYPE    = "tcp"
)

const protocolName = "ROSE/0.1"

var (
	ilog *log.Logger
	wlog *log.Logger
	elog *log.Logger
)

var tables map[string]*Table
var tablePaths map[string]string

func main() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	initTables()

	//// TEST

	name := "ensembl_ids_hsapiens_head"
	loadTable(name)

	table := tables[name]
	table.Print()

	stable := NewSortedTable(table, "ensembl_gene_id")
	fmt.Println(stable.Slice("ensembl_gene_id"))
	fmt.Println(stable.Slice("ensembl_transcript_id"))

	xs := []Identifier{"ENSG00000210100", "ENSG00000210077", "ENGSG00000111111", "ENSG00000281540"}
	fmt.Println(stable.Map(xs, "ensembl_transcript_id"))
	fmt.Println(stable.Map(xs, "ensembl_peptide_id"))

	////

	// Create listener
	ln, err := net.Listen(CONN_TYPE, CONN_ADDRESS)
	if err != nil {
		elog.Println("Cannot create listener:", err)
		os.Exit(1)
	}
	defer ln.Close()

	// Listen for incoming connections
	ilog.Println("Listening on", CONN_ADDRESS)
	for {
		conn, err := ln.Accept()
		if err != nil {
			elog.Println("Cannot accept:", err)
			os.Exit(1)
		} else {
			ilog.Println("Accepted connection from", conn.RemoteAddr())
		}
		go handleRequest(conn)
	}
}

func initTables() {
	tables = make(map[string]*Table)
	tablePaths = make(map[string]string)

	paths := []string{
		"test.tsv",
		"/Users/davids/data/biomart/release-81/ensembl_ids_hsapiens_head.tsv",
		"/Users/davids/data/biomart/release-81/ensembl_ids_hsapiens.tsv",
		"/Users/davids/data/biomart/release-81/ensembl_ids_mmusculus.tsv",
		"/Users/davids/data/biomart/release-81/compara_homologs_hsapiens-mmusculus.tsv",
	}
	addTablePaths(paths)
}

func initLogs(i io.Writer, w io.Writer, e io.Writer) {
	ilog = log.New(i, "Info: ", log.Ldate|log.Ltime)
	wlog = log.New(w, "Warning: ", log.Ldate|log.Ltime|log.Lshortfile)
	elog = log.New(e, "Error: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func initResponse(status int) (r string) {
	r = protocolName + " " + strconv.Itoa(status) + " "
	switch status {
	case 200:
		r += "OK"
	case 201:
		r += "Created"
	case 202:
		r += "Accepted"
	case 204:
		r += "No Content"
	case 205:
		r += "Reset Content"
	case 400:
		r += "Bad Request"
	case 401:
		r += "Unauthorized"
	case 404:
		r += "Not Found"
	case 408:
		r += "Request Timeout"
	case 500:
		r += "Internal Error"
	default:
		r += "Unknown"
	}
	r += "\n"
	if status == 200 {
		r += "\n"
	}
	return
}

func handleRequest(conn net.Conn) {
	var command string
	var inputIds []Identifier

	response := initResponse(500)
	defer func() {
		// Send response back
		conn.Write([]byte(response))
		conn.Close()
		ilog.Println("Closed connection with", conn.RemoteAddr())
	}()

	// Read request
	scanner := bufio.NewScanner(conn)
	// first line is the command
	if !scanner.Scan() {
		response = initResponse(404)
		elog.Println("Cannot read request")
		return
	} else {
		command = scanner.Text()
		for scanner.Scan() {
			inputIds = append(inputIds, Identifier(scanner.Text()))
		}
	}

	// first line is the command
	tokens := strings.Split(command, " ")
	n := len(tokens) - 1
	action := tokens[0]

	switch action {
	case "map":
		if n == 3 {
			outputIds, err := mapIdentifiers(inputIds, tokens[1], tokens[2], tokens[3])

			if err != nil {
				response = initResponse(404)
				elog.Println("Cannot map identifiers from", conn.RemoteAddr())
			} else {
				// Concatenate identifiers
				if len(outputIds) > 0 {
					response = initResponse(200)
					for _, id := range outputIds {
						response += string(id) + entryDelimiter
					}
				} else {
					response = initResponse(204)
				}
				ilog.Println("Mapped identifiers from", conn.RemoteAddr())
			}
		} else {
			response = initResponse(400)
			elog.Println("Invalid command, expecting: map <srcId> <destId> <table>")
		}
	case "load":
		if n == 0 {
			if err := loadAllTables(); err != nil {
				response = initResponse(404)
			} else {
				response = initResponse(201)
				ilog.Println("All known tables are preloaded")
			}
		} else if n == 1 {
			if err := loadTable(tokens[1]); err != nil {
				response = initResponse(404)
			} else {
				response = initResponse(201)
				ilog.Println("Table", tokens[1], "is preloaded")
			}
		} else {
			response = initResponse(400)
			elog.Println("Invalid command, expecting: load [<table> ...]")
		}
	case "unload":
		if n == 0 {
			if err := unloadAllTables(); err != nil {
				response = initResponse(404)
			} else {
				response = initResponse(204)
				ilog.Println("All extant tables are unloaded")
			}
		} else if n == 1 {
			if err := unloadTable(tokens[1]); err != nil {
				response = initResponse(404)
			} else {
				response = initResponse(204)
				ilog.Println("Table", tokens[1], "is unloaded")
			}
		} else {
			response = initResponse(400)
			elog.Println("Invalid command, expecting: unload [<table> ...]")
		}
	case "reload":
		if n == 0 {
			if err := reloadAllTables(); err != nil {
				response = initResponse(404)
			} else {
				response = initResponse(205)
				ilog.Println("All extant tables are reloaded")
			}
		} else if n == 1 {
			if err := reloadTable(tokens[1]); err != nil {
				response = initResponse(404)
			} else {
				response = initResponse(205)
				ilog.Println("Table", tokens[1], "is reloaded")
			}
		} else {
			response = initResponse(400)
			elog.Println("Invalid command, expecting: reload [<table> ...]")
		}
	case "list":
		response = initResponse(200)
		response += strings.Join(getTables(), entryDelimiter) + entryDelimiter
	default:
		response = initResponse(400)
		elog.Println("Unknown command")
	}

}

func mapIdentifiers(xs []Identifier, tableName, srcId, destId string) (ys []Identifier, err error) {
	if err = loadTable(tableName); err == nil {
		stable := NewSortedTable(tables[tableName], srcId)
		ys = stable.Map(xs, destId)
	}
	return
}

func reloadTable(name string) error {
	unloadTable(name)
	return loadTable(name)
}

func reloadAllTables() error {
	var err error
	for k := range tables {
		if e := unloadTable(k); e != nil {
			err = e
		}
	}
	return err
}

func unloadTable(name string) error {
	if _, exists := tables[name]; exists {
		delete(tables, name)
		return nil
	}
	err := errors.New("Table " + name + " does not exist")
	wlog.Println(err)
	return err
}

func unloadAllTables() error {
	var err error
	for k := range tables {
		if e := unloadTable(k); e != nil {
			err = e
		}
	}
	return err
}

func loadTable(name string) (err error) {
	// Check if table is already loaded
	if _, exists := tables[name]; exists {
		// Do not reload table
		return
	}

	// Look up path to table
	if path, ok := tablePaths[name]; ok {
		// Open and read table
		f, err := os.Open(path)
		if err != nil {
			elog.Println(err)
		} else {
			defer f.Close()
			tables[name] = NewTable(f)
			ilog.Println("Loaded table", name)
		}
	} else {
		err = errors.New("Path to table " + name + " is unknown.")
		elog.Println(err)
	}

	return
}

func loadAllTables() error {
	var err error
	for k := range tablePaths {
		if e := loadTable(k); e != nil {
			err = e
		}
	}
	return err
}

func getTables() (names []string) {
	for k := range tablePaths {
		names = append(names, k)
	}
	sort.StringSlice(names).Sort()
	return
}

func addTablePath(path string) error {
	var err error
	name := getTableName(path)
	if _, exists := tablePaths[name]; exists {
		err = errors.New("Table name conflict exists for " + name)
		wlog.Println(err)
	} else {
		tablePaths[name] = path
	}
	return err
}

func addTablePaths(paths []string) error {
	var err error
	for _, p := range paths {
		if err == nil {
			err = addTablePath(p)
		} else {
			addTablePath(p)
		}
	}
	return err
}

func getTableName(path string) string {
	i := strings.LastIndex(path, "/")
	j := strings.LastIndex(path, ".")
	if i == -1 {
		i = 0
	} else {
		i++
	}
	if j == -1 {
		j = len(path)
	}
	return path[i:j]
}
