// Package packerbin provides access to a stored packer binary.
//
// It is useful for when you need packer, but it may not be on the system and
// cannot be downloaded.
//
// Note: this package contains packer binaries downloaded from packer.io the
// source code of packer is available at https://github.com/mitchellh/packer.
package packerbin

import (
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// Version of packer encoded
const Version = "v0.12.2"

// Name of the packer executable ("packer" on Unix "packer.exe" on Windows)
const Name = packerFilename

// Sha1, returns the SHA256 sum of the encoded packer binary.
func Sha1() (sum [sha1.Size]byte) {
	s, err := hex.DecodeString(packerSHA1)
	if err != nil {
		panic(err)
	}
	copy(sum[0:], s)
	return
}

// Sha256, returns the SHA256 sum of the encoded packer binary.
func Sha256() (sum [sha256.Size]byte) {
	s, err := hex.DecodeString(packerSHA256)
	if err != nil {
		panic(err)
	}
	copy(sum[0:], s)
	return
}

func newReader() (*gzip.Reader, error) {
	d := base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(packerBinary))
	return gzip.NewReader(d)
}

// NewReader returns a reader to the encoded packer binary.
func NewReader() (io.ReadCloser, error) {
	return newReader()
}

// WriteFile, writes the packer binary to a new file and sets the executable bits.
func WriteFile(name string) error {
	r, err := newReader()
	if err != nil {
		fmt.Println("1")
		return err
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		fmt.Println("2")
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(name)
		fmt.Println("3")
		return err
	}
	return f.Close()
}

// Bytes returns the decoded bytes of the encoded packer binary.
func Bytes() ([]byte, error) {
	r, err := newReader()
	if err != nil {
		return nil, err
	}
	p := make([]byte, decodedLen)
	n := 0
	for {
		m, e := r.Read(p[n:])
		n += m
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
	}
	if n != decodedLen {
		return nil, io.ErrUnexpectedEOF
	}
	return p, nil
}
