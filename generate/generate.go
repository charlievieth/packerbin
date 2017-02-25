package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	DecodedLen int
	BinaryFile string
	GoFile     string
	BinaryName string
	BuildTags  string
	Sha256Sum  [sha256.Size]byte
	Sha1Sum    [sha1.Size]byte
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s: [OPTIONS] PACKER_BINARY GO_FILENAME\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Executable()
	os.Exit(1)
}

func init() {
	flag.StringVar(&BuildTags, "tags", "", "build tags")
	flag.StringVar(&BinaryName, "name", "", "packer binary name")
	flag.Usage = Usage
}

func EncodeBinary(filename string, out *bytes.Buffer) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	DecodedLen = len(b)
	Sha256Sum = sha256.Sum256(b)
	Sha1Sum = sha1.Sum(b)

	e := base64.NewEncoder(base64.RawStdEncoding, out)
	w, err := gzip.NewWriterLevel(e, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	if err := e.Close(); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "invalid flags:", args)
		Usage()
	}
	BinaryFile = args[0]
	GoFile = args[1]
	if BinaryName == "" {
		BinaryName = "packer"
	}

	f, err := os.OpenFile(GoFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating file (%s): %s\n", GoFile, err)
		os.Exit(1)
	}
	defer f.Close()

	var buf bytes.Buffer
	if err := EncodeBinary(BinaryFile, &buf); err != nil {
		fmt.Fprintf(os.Stderr, "encoding binary file (%s): %s\n", BinaryFile, err)
		os.Exit(1)
	}

	var out bytes.Buffer

	const Header = `// MACHINE GENERATED DO NOT EDIT!

// This source file contains an encoded packer executable (packerBinary).
// The packer source code is available at: https://github.com/mitchellh/packer

`
	out.WriteString(Header)

	if BuildTags != "" {
		fmt.Fprintf(&out, "// +build %s\n\n", BuildTags)
	}

	const Format = `package packerbin
// length of decoded binary
const decodedLen = %d

// name of the packer binary (on Unix "packer", on Windows "packer.exe")
const packerFilename = "%s"

// hex encoded sha1 sum of the packer binary
const packerSHA1 = "%s"

// hex encoded sha256 sum of the packer binary
const packerSHA256 = "%s"

// packer executable compressed with gzip and base64 encoded.
const packerBinary =` + "`\n" // append backtick '`' and newline

	fmt.Fprintf(&out, Format,
		DecodedLen,
		BinaryName,
		hex.EncodeToString(Sha1Sum[0:]),
		hex.EncodeToString(Sha256Sum[0:]),
	)

	scratch := make([]byte, 80)
	for buf.Len() > 0 {
		n, err := buf.Read(scratch)
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		out.Write(scratch[:n])
		out.WriteByte('\n')
		if err != nil {
			break
		}
	}
	out.WriteString("`\n")

	if err := FormatAST(&out, f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func FormatAST(buf *bytes.Buffer, w io.Writer) error {
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, "packer.go", buf, parser.ParseComments)
	if err != nil {
		return err
	}
	return printer.Fprint(w, fset, af)
}
