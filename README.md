# rose

> What's in a name? that which we call a rose  
> By any other name would smell as sweet.  
> -- William Shakespeare

This package faciliates identifier conversion. It consists of a client and a
server program for converting identifiers based on conversion tables in
tab-delimited format. Such conversion tables can be retrieved from
resources including BioMart UCSC Genome Browser.

## Requirement

To compile this package, you will need (go)[https://golang.org/].

## Installation

    go install ./rose
    go install ./rosed

## Usage

First, you need to start the `rosed` server, which will search for tables found
in paths specified by the `ROSE_TABLES_PATH` environmental variable and restrict
the search to files with the extensions matching those specified in
`ROSE_TABLES_EXT`.

    ROSE_TABLES_PATH=test rosed

`rosed` will also record paths to available tables listed in an input file
specified by the `--tables` command line argument. This file should contain
absolute paths.

Then, you can use the client program `rose` to send queries to `rosed`. By
default, `rose` reads identifiers (on separate lines) from `stdin`. You may
also specify an input file:

    rose --input test/ids.txt test id1 id2

If the request is successful, `rose` will output (to stdout) each mapped
identifier on a line, where blank lines represent identifiers which could not be
mapped. (All error messages are output to stderr.)

## Remarks

This conversion task is split into a client and a server program in order to
optimize efficiency. The server will load tables and create indices as needed and
keep them in memory to avoid disk IO. Given a query list of size *m* and a
conversion data table with *n* entries, the complexity of the mapping operation
is *(m + n) log(n)* and subsequent mapping operations on the same table from the
same identifier type (to any identifier type) is *m log(n)*.

