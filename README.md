# rose

> What's in a name? that which we call a rose  
> By any other name would smell as sweet.  
> -- William Shakespeare

This package facilitates name identifier conversion. It consists of a client
(`rose`) and a server (`rosed`) program for converting identifiers from one type
to another based on conversion tables stored in tab-delimited format.

Existing conversion tables can be retrieved from such public databases using, for example, [apiarius](https://github.com/djhshih/apiarius).

## Requirement

To compile this package, you will need [go](https://golang.org).

## Installation

Assuming your `GOPATH` has been setup, you can simply download and install this
package by

    go get github.com/djhshih/rose
    go get github.com/djhshih/rosed

If you choose to clone this project to a non-canonical location, you can also
install the programs within the project directory by

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

For more details on command line arguments, see `rose --help` and `rosed
--help`.

## Remarks

### Efficiency

This conversion task is split into a client and a server program in order to
optimize efficiency. The server will load tables and create indices as needed
and keep them in memory to avoid disk read operations. Given a query list
of size *m* and a conversion data table with *n* entries, the complexity of the
mapping operation is *O( (m + n) log(n) )* and subsequent mapping operations on
the same table from the same identifier type (to any identifier type) is *O( m
log(n) )*.

### Mapping

The mapped identifiers are provided in the same order as the source
identifiers so that there is a one-to-one correspondence between each original
identifier and the mapped identifier. This output format works best for
one-to-one mapping relationships or many-to-one mapping relationships. In the
case of one-to-many mapping, the mapped identifiers for the same source
identifiers are concatenated by a comma character.

