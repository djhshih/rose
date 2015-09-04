package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	connHost       string
	connPort       string
	connAddress    string
	tablePathsFile string
)

const protocolName = "ROSE/0.1"

const (
	fieldDelim = "\t"
	idDelim    = ","
	entryDelim = "\n"
	tableDelim = "/"
)

var (
	ilog *log.Logger
	wlog *log.Logger
	elog *log.Logger
)

var tables map[string]*Table
var tablePaths map[string]string

func init() {
	tables = make(map[string]*Table)
	tablePaths = make(map[string]string)

	flag.StringVar(&connHost, "host", "127.0.0.1", "local host address to listen on")
	flag.StringVar(&connPort, "port", "12053", "local host port to listen on")
	connAddress = connHost + ":" + connPort

	flag.StringVar(&tablePathsFile, "tables", "", "file with paths to tables")
}

func main() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	initTables()

	// Create listener
	ln, err := net.Listen("tcp", connAddress)
	if err != nil {
		elog.Fatal("Cannot create listener:", err)
	}
	defer ln.Close()

	// Listen for incoming connections
	ilog.Println("Listening on", connAddress)
	for {
		conn, err := ln.Accept()
		if err != nil {
			elog.Println("Cannot accept connection:", err)
		} else {
			go handleRequest(conn)
			ilog.Println("Accepted connection from", conn.RemoteAddr())
		}
	}
}

func initTables() {
	var paths []string

	if tablePathsFile != "" {
		f, err := os.Open(tablePathsFile)
		if err != nil {
			elog.Println(err)
		} else {
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				paths = append(paths, scanner.Text())
			}
		}
	}

	tableDirs := os.Getenv("ROSE_TABLES_PATH")
	if tableDirs != "" {
		tableExts := os.Getenv("ROSE_TABLES_EXT")
		if tableExts == "" {
			tableExts = "tsv"
		}
		for _, dir := range strings.Split(tableDirs, string(os.PathListSeparator)) {
			for _, ext := range strings.Split(tableExts, string(os.PathListSeparator)) {
				pattern := dir + string(os.PathSeparator) + "*." + ext
				matches, _ := filepath.Glob(pattern)
				for _, fn := range matches {
					paths = append(paths, fn)
				}
			}
		}
	}

	ilog.Println("Table paths:" + entryDelim + strings.Join(paths, entryDelim))
	addTablePaths(paths)

	if len(tablePaths) == 0 {
		elog.Fatal("No tables are available for loading")
	}
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
						response += string(id) + entryDelim
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
		response += strings.Join(getTables(), entryDelim) + entryDelim
	default:
		response = initResponse(400)
		elog.Println("Unknown command")
	}

}

func mapIdentifiers(xs []Identifier, tableName, srcId, destId string) (ys []Identifier, err error) {
	if err = loadTable(tableName); err == nil {
		sorted := tables[tableName].Sorted(srcId)
		ys = sorted.Map(xs, destId)
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
		if e := reloadTable(k); e != nil {
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

// getTables return a list of available tables
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
		// add path only if target is a valid file
		if info, err := os.Stat(path); err != nil {
			elog.Println("Path is not found or not accessible:", path)
		} else if !info.Mode().IsRegular() {
			elog.Println("Cannot add table path, not a file:", path)
		} else {
			tablePaths[name] = path
		}
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
