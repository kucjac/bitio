/*

Reader interface definition and implementation.

*/

package bitio

import (
	"io"
)

// An io.Reader and io.ByteReader at the same time.
type readerAndByteReader interface {
	io.Reader
	io.ByteReader
}

// reader is the bit reader implementation.
type Reader struct {
	in           []byte
	cache        byte // unread bits are stored here
	bits         byte // number of unread bits in cache
	r, w         int  // buf read and write positions
	err          error
	lastByte     int
	lastRuneSize int
}

// NewReader returns a new Reader using the specified byte slice as the input (source).
func NewReader(p []byte) *Reader {
	return &Reader{in: p}
}

func (r *Reader) readBufferByte() (b byte, err error) {
	if r.r >= len(r.in) {
		return 0, io.EOF
	}
	r.lastRuneSize = -1
	c := r.in[r.r]
	r.r++
	r.lastByte = int(c)
	return c, nil
}

func (r *Reader) ReadBits(n byte) (u uint64, err error) {
	// Some optimization, frequent cases
	if n < r.bits {
		// cache has all needed bits, and there are some extra which will be left in cache
		shift := r.bits - n
		u = uint64(r.cache >> shift)
		r.cache &= 1<<shift - 1
		r.bits = shift
		return
	}

	if n > r.bits {
		// all cache bits needed, and it's not even enough so more will be read
		if r.bits > 0 {
			u = uint64(r.cache)
			n -= r.bits
		}
		// Read whole bytes
		for n >= 8 {
			b, err2 := r.readBufferByte()
			if err2 != nil {
				return 0, err2
			}
			u = u<<8 + uint64(b)
			n -= 8
		}
		// Read last fraction, if any
		if n > 0 {
			if r.cache, err = r.readBufferByte(); err != nil {
				return 0, err
			}
			shift := 8 - n
			u = u<<n + uint64(r.cache>>shift)
			r.cache &= 1<<shift - 1
			r.bits = shift
		} else {
			r.bits = 0
		}
		return u, nil
	}

	// cache has exactly as many as needed
	r.bits = 0 // no need to clear cache, will be overridden on next read
	return uint64(r.cache), nil
}

// ReadByte implements io.ByteReader.
func (r *Reader) ReadByte() (b byte, err error) {
	// r.bits will be the same after reading 8 bits, so we don't need to update that.
	if r.bits == 0 {
		return r.readBufferByte()
	}
	return r.readUnalignedByte()
}

// readUnalignedByte reads the next 8 bits which are (may be) unaligned and returns them as a byte.
func (r *Reader) readUnalignedByte() (b byte, err error) {
	// r.bits will be the same after reading 8 bits, so we don't need to update that.
	bits := r.bits
	b = r.cache << (8 - bits)
	r.cache, err = r.readBufferByte()
	if err != nil {
		return 0, err
	}
	b |= r.cache >> bits
	r.cache &= 1<<bits - 1
	return
}

func (r *Reader) ReadBool() (b bool, err error) {
	if r.bits == 0 {
		r.cache, err = r.readBufferByte()
		if err != nil {
			return
		}
		b = (r.cache & 0x80) != 0
		r.cache, r.bits = r.cache&0x7f, 7
		return
	}

	r.bits--
	b = (r.cache & (1 << r.bits)) != 0
	r.cache &= 1<<r.bits - 1
	return
}

func (r *Reader) Align() (skipped byte) {
	skipped = r.bits
	r.bits = 0 // no need to clear cache, will be overwritten on next read
	return
}
