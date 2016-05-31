package bitio

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	data := []byte{3, 255, 0xcc, 0x1a, 0xbc, 0xde, 0x80, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	br := NewReader(bytes.NewBuffer(data))

	if b, err := br.ReadByte(); b != 3 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", b, 3, err)
	}
	if i, err := br.ReadBits(8); i != 255 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", i, 255, err)
	}

	if i, err := br.ReadBits(4); i != 0xc || err != nil {
		t.Errorf("Got %x, want %x, error: %v", i, 0xc, err)
	}

	if i, err := br.ReadBits(8); i != 0xc1 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", i, 0xc1, err)
	}

	if i, err := br.ReadBits(20); i != 0xabcde || err != nil {
		t.Errorf("Got %x, want %x, error: %v", i, 0xabcde, err)
	}

	if b, err := br.ReadBool(); !b || err != nil {
		t.Errorf("Got %v, want %v, error: %v", b, false, err)
	}
	if b, err := br.ReadBool(); b || err != nil {
		t.Errorf("Got %v, want %v, error: %v", b, true, err)
	}

	if n := br.Align(); n != 6 {
		t.Errorf("Got %v, want %v", n, 6)
	}

	s := make([]byte, 2)
	if n, err := br.Read(s); n != 2 || err != nil || !bytes.Equal(s, []byte{0x01, 0x02}) {
		t.Errorf("Got %v, want %v, error: %v", s, []byte{0x01, 0x02}, err)
	}

	if i, err := br.ReadBits(4); i != 0xf || err != nil {
		t.Errorf("Got %x, want %x, error: %v", i, 0xf, err)
	}

	if n, err := br.Read(s); n != 2 || err != nil || !bytes.Equal(s, []byte{0x80, 0x8f}) {
		t.Errorf("Got %v, want %v, error: %v", s, []byte{0x80, 0x8f}, err)
	}
}

func TestWriter(t *testing.T) {
	b := &bytes.Buffer{}

	bw := NewWriter(b)

	expected := []byte{0xc1, 0x7f, 0xac, 0x89, 0x24, 0x78, 0x01, 0x02, 0xf8, 0x08, 0xf0}

	errs := []error{}
	errs = append(errs, bw.WriteByte(0xc1))
	errs = append(errs, bw.WriteBool(false))
	errs = append(errs, bw.WriteBits(0x3f, 6))
	errs = append(errs, bw.WriteBool(true))
	errs = append(errs, bw.WriteByte(0xac))
	errs = append(errs, bw.WriteBits(0x01, 1))
	errs = append(errs, bw.WriteBits(0x1248f, 20))

	if n, err := bw.Align(); n != 3 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", n, 3, err)
	}

	if n, err := bw.Write([]byte{0x01, 0x02}); n != 2 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", n, 2, err)
	}

	errs = append(errs, bw.WriteBits(0x0f, 4))

	if n, err := bw.Write([]byte{0x80, 0x8f}); n != 2 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", n, 2, err)
	}

	if n, err := bw.Align(); n != 4 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", n, 4, err)
	}
	if n, err := bw.Align(); n != 0 || err != nil {
		t.Errorf("Got %x, want %x, error: %v", n, 0, err)
	}

	errs = append(errs, bw.Close())

	for _, v := range errs {
		if v != nil {
			t.Error("Got error:", v)
		}
	}

	if !bytes.Equal(b.Bytes(), expected) {
		t.Errorf("Got: %x, want: %x", b.Bytes(), expected)
	}
}

func TestChain(t *testing.T) {
	b := &bytes.Buffer{}
	bw := NewWriter(b)

	rand.Seed(time.Now().UnixNano())

	expected := make([]uint64, 100000)
	bits := make([]byte, len(expected))

	// Writing (generating)
	for i := range expected {
		expected[i] = uint64(rand.Int63())
		bits[i] = byte(1 + rand.Int31n(60))
		expected[i] &= uint64(1)<<bits[i] - 1
		bw.WriteBits(expected[i], bits[i])
	}
	if err := bw.Close(); err != nil {
		t.Error("Got error:", err)
	}

	br := NewReader(bytes.NewBuffer(b.Bytes()))

	// Reading (verifying)
	for i, v := range expected {
		if r, err := br.ReadBits(bits[i]); r != v || err != nil {
			t.Errorf("Idx: %d, Got: %x, want: %x, bits: %d, error: %v", i, r, v, bits[i], err)
		}
	}
}