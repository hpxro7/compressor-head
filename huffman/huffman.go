package huffman

import (
	"fmt"
	"io"
)

// Distribution stores a probability distribution over input bytes.
type Distribution struct {
	cnts  map[byte]uint64
	total uint64
}

type Writer struct {
	w           io.Writer
	d           Distribution
	wroteHeader bool
}

func NewDistribution(bs []byte) *Distribution {
	d := new(Distribution)
	d.cnts = make(map[byte]uint64)

	for _, val := range bs {
		d.cnts[val]++
		d.total++
	}

	return d
}

// Of returns the probability of b in this distribution.
func (d *Distribution) Of(b byte) float64 {
	return float64(d.cnts[b]) / float64(d.total)
}

func (d *Distribution) String() string {
	buf := make([]byte, 0)
	for val, cnt := range d.cnts {
		prob := float64(cnt) / float64(d.total)
		v := fmt.Sprintf("%d(%c):%.3f, ", val, val, prob)
		buf = append(buf, v...)
	}
	return "dist[" + string(buf[:len(buf)-2]) + "]"
}

// NewWriter returns a new Writer.
// Writes to the returned writer are compressed in accordance to the provided
// distribution of bytes.
//
// It is the caller's responsibility to call Close on the WriterCloser when done.
// Writes may be buffered and not flushed until Close.
func NewWriter(w io.Writer, d *Distribution) *Writer {
	h := new(Writer)
	h.d = *d
	return h
}
