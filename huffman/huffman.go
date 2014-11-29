// pakcage huffman exports a Reader/Writer for compressing and decompressing data streams using
// variable length codes via huffman encoding using a specified probability distribution
package huffman

import (
	"container/heap"
	"fmt"
	"io"
)

// Distribution stores a probability distribution over input bytes.
type Distribution struct {
	cnts  map[byte]uint64
	total uint64
}

// node represents a node in a Huffman tree which is either a resolved leaf node representing a
// value or some node on the path to a child
type node struct {
	l     *node
	r     *node
	val   byte   // Value that this node represents
	cnt   uint64 // Frequency of val
	index int
}

type nodeHeap []*node

type Writer struct {
	w           io.Writer
	root        *node          // Root of the Huffman tree
	mapping     *map[byte]byte // Mapping from byte values to their Huffman binary representation
	wroteHeader bool
}

// NewDistribution returns a new probability distribution over a sample of a stream of bytes.
//
// The probablity distribution of the stream is approximated by using the frequency of byte values
// in the sample.
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

// toHeap returns a min heap minimizing over the node's count of values.
func (d *Distribution) toHeap() nodeHeap {
	nodes := make(nodeHeap, 0)
	for val, cnt := range d.cnts {
		nodes = append(nodes, &node{val: val, cnt: cnt})
	}
	heap.Init(&nodes)
	return nodes
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

func (h nodeHeap) Len() int {
	return len(h)
}

func (h nodeHeap) Less(i, j int) bool {
	return h[i].cnt < h[j].cnt
}

func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = h[j].index, h[i].index
}

func (h *nodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1
	*h = old[0 : n-1]
	return item
}

func (h *nodeHeap) Push(val interface{}) {
	n := len(*h)
	item := val.(*node)
	item.index = n
	*h = append(*h, item)
}

// buildTree constructs a Huffman tree from the min heap of byte value nodes.
func (h nodeHeap) buildTree() (root *node) {
	if len(h) >= 1 {
		for len(h) >= 2 {
			r, l := h.Pop().(node), h.Pop().(node)
			next := &node{
				cnt: r.cnt + l.cnt,
			}
			next.l, next.r = &l, &r
			h.Push(next)
		}
		last := h.Pop().(node)
		root = &last
	}
	return
}

// NewWriter returns a new Writer.
// Writes to the returned writer are compressed in accordance to the provided
// distribution of bytes.
//
// It is the caller's responsibility to call Close on the WriterCloser when done.
// Writes may be buffered and not flushed until Close.
//
// The distribution must be completely representative of the data to be written. Writing data whose
// probabilities have not been specified will result in an error.
func NewWriter(w io.Writer, d *Distribution) *Writer {
	h := new(Writer)
	h.root = d.toHeap().buildTree()
	return h
}
