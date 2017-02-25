package packerbin

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBytes(t *testing.T) {
	b, err := Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != decodedLen {
		t.Errorf("expected decoded len (%d): got (%d)", decodedLen, len(b))
	}
	sumSha1 := sha1.Sum(b)
	if s := hex.EncodeToString(sumSha1[0:]); s != packerSHA1 {
		t.Errorf("sha1 mismatch: expected (%s) got (%s)", packerSHA1, s)
	}
	sumSha256 := sha256.Sum256(b)
	if s := hex.EncodeToString(sumSha256[0:]); s != packerSHA256 {
		t.Errorf("sha256 mismatch: expected (%s) got (%s)", packerSHA256, s)
	}
}

func TestSha1(t *testing.T) {
	data, err := Bytes()
	if err != nil {
		t.Fatal(err)
	}
	a := sha1.Sum(data)
	b := Sha1()
	if !bytes.Equal(a[0:], b[0:]) {
		t.Errorf("sha1 mismatch: expected (%x) got (%x)", a, b)
	}
}

func TestSha256(t *testing.T) {
	data, err := Bytes()
	if err != nil {
		t.Fatal(err)
	}
	a := sha256.Sum256(data)
	b := Sha256()
	if !bytes.Equal(a[0:], b[0:]) {
		t.Errorf("sha256 mismatch: expected (%x) got (%x)", a, b)
	}
}

func TestWriteFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "packerbin-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filename := filepath.Join(dir, Name)
	if err := WriteFile(filename); err != nil {
		t.Fatal(err)
	}
	testExecutable(t, filename)
}

func TestReader(t *testing.T) {
	dir, err := ioutil.TempDir("", "packerbin-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filename := filepath.Join(dir, Name)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 755)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := NewReader()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if _, err := io.Copy(f, r); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	testExecutable(t, filename)
}

func testFileHash(t *testing.T, filename string) {
	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h1 := sha1.New()
	h256 := sha256.New()
	w := io.MultiWriter(h1, h256)
	if _, err := io.Copy(w, f); err != nil {
		t.Fatal(err)
	}

	sum1 := h1.Sum(nil)
	x1 := Sha1()
	if !bytes.Equal(sum1[0:], x1[0:]) {
		t.Errorf("sha1 mismatch: expected (%x) got (%x)", x1, sum1)
	}

	sum256 := h256.Sum(nil)
	x256 := Sha1()
	if !bytes.Equal(sum256[0:], x256[0:]) {
		t.Errorf("sha1 mismatch: expected (%x) got (%x)", x256, sum256)
	}
}

func testExecutable(t *testing.T, filename string) {
	path, err := exec.LookPath(filename)
	if err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %s\nOutput:\n%s\n", err, string(out))
	}
	if !bytes.Contains(out, []byte(Version)) {
		t.Fatalf("expected to find version (%s) in output of `packer version`:\n%s\n", Version, string(out))
	}
}

// benchmarks were written out of curiosity

func BenchmarkBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Bytes(); err != nil {
			b.Fatal(err)
		}
	}
}

type noopWriter struct{}

func (n *noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func BenchmarkReader(b *testing.B) {
	var w noopWriter
	buf := make([]byte, 32*1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := newReader()
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.CopyBuffer(&w, r, buf); err != nil {
			b.Fatal(err)
		}
	}
}
