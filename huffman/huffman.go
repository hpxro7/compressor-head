// pakcage huffman exports a Reader/Writer for compressing and decompressing data streams using
// variable length codes via huffman encoding using a specified probability distribution
package huffman

import (
	"container/heap"
	"errors"
	"fmt"
	"io"
)

type code struct {
	path byte   // Huffman encoding of the path to the value node in a tree
	len  uint32 // The number of bits that this encoding takes up
}

func (c code) String() string {
	buf := make([]byte, 0)
	for n, left := c.path, c.len; left > 0; n, left = n/2, left-1 {
		buf = append(buf, '0'+(n%2))
	}
	return string(buf)
}

// Distribution stores a probability distribution over input bytes.
type Distribution struct {
	cnts  map[byte]uint64
	total uint64
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
func (d *Distribution) toHeap() *nodeHeap {
	nodes := make(nodeHeap, 0)
	for val, cnt := range d.cnts {
		nodes = append(nodes, &node{val: val, cnt: cnt})
	}
	heap.Init(&nodes)
	return &nodes
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

// node represents a node in a Huffman tree which is either a resolved leaf node representing a
// value or some node on the path to a child
type node struct {
	l   *node
	r   *node
	val byte   // Value that this node represents
	cnt uint64 // Frequency of val
}

type nodeHeap []*node

func (h nodeHeap) Len() int {
	return len(h)
}

func (h nodeHeap) Less(i, j int) bool {
	return h[i].cnt < h[j].cnt
}

func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]

}

func (h *nodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

func (h *nodeHeap) Push(val interface{}) {
	item := val.(*node)
	*h = append(*h, item)
}

// buildTree constructs a Huffman tree from the min heap of byte value nodes.
func (h *nodeHeap) buildTree() (root *node) {
	if len(*h) >= 1 {
		for len(*h) >= 2 {
			l, r := heap.Pop(h).(*node), heap.Pop(h).(*node)
			next := &node{
				cnt: r.cnt + l.cnt,
			}
			next.l, next.r = l, r
			heap.Push(h, next)
		}
		last := heap.Pop(h).(*node)
		root = last
	}
	return
}

func getMapping(root *node) map[byte]code {
	mapping := make(map[byte]code)
	expandPaths(0, 0, root, mapping)
	return mapping
}

func decode(curr *node, next func() (byte, error)) (byte, error) {
	if curr.l == nil && curr.r == nil {
		return curr.val, nil
	} else {
		dir, err := next()
		if err != nil {
			return 0, err
		}
		if dir == 0 {
			return decode(curr.l, next)
		} else {
			return decode(curr.r, next)
		}
	}
}

func expandPaths(path byte, depth uint32, curr *node, mapping map[byte]code) {
	if curr != nil {
		// Node is a child node, add its value to the mapping
		if curr.l == nil && curr.r == nil {
			mapping[curr.val] = code{path: path, len: depth}
		} else {
			leftPath := path
			var rightPath byte
			if depth == 0 {
				rightPath = path + 1
			} else {
				rightPath = path + (2 << (depth - 1))
			}
			expandPaths(leftPath, depth+1, curr.l, mapping)
			expandPaths(rightPath, depth+1, curr.r, mapping)
		}
	}
}

type Writer struct {
	io.Writer
	root        *node         // Root of the Huffman tree
	mapping     map[byte]code // Mapping from byte values to their Huffman binary representation
	wroteHeader bool
}

// NewWriter returns a new Writer.
// Writes to the returned writer are compressed in accordance to the provided
// distribution of bytes.
//
// It is the caller's responsibility to call Close on the WriterCloser when done.
// Writes may be buffered and not flushed until Close.
//
// The distribution must be completely representative of the data to be written. Writing data
// whose probabilities have not been specified will result in an error.
func NewWriter(w io.Writer, d *Distribution) *Writer {
	h := new(Writer)
	h.Writer = w
	h.root = d.toHeap().buildTree()
	h.mapping = getMapping(h.root)
	return h
}

func (w Writer) Write(p []byte) (n int, err error) {
	encoded := make([]byte, 0)
	curr := byte(0)
	left := 8
	for i, val := range p {
		wrote := false
		c, ok := w.mapping[val]
		if !ok {
			return 0, errors.New(fmt.Sprintf("probability of '%c'(%d) was not in distribution", val, val))
		}
		for read, p := uint32(0), c.path; read < c.len; p, read = p/2, read+1 {
			if left == 8 {
				curr += p % 2
			} else {
				curr += (p % 2) * (2 << byte(8-left-1))
			}

			if left == 1 {
				left = 8
				wrote = true
				encoded = append(encoded, curr)
				curr = 0
			} else {
				left--
			}
		}

		if i == len(p)-1 && !wrote {
			encoded = append(encoded, curr)
		}
	}
	return w.Writer.Write(encoded)
}

func (w Writer) Huffman() *node {
	return w.root
}

type Reader struct {
	io.Reader
	root *node // Root of the Huffman tree
}

func NewReader(r io.Reader, root *node) *Reader {
	return &Reader{r, root}
}

//TODO(zac): fix reading extraneous bits at end of stream
func toBitStream(input []byte) func() (byte, error) {
	buf := make([]byte, len(input))
	copy(buf, input)
	var left byte = 8
	return func() (next byte, err error) {
		if len(buf) == 0 {
			return 0, io.EOF
		}
		left--
		next = buf[0] % 2
		buf[0] /= 2

		if left == 0 {
			left = 8
			buf = buf[1:len(buf)]
		}
		return
	}
}

func (r Reader) Read(p []byte) (n int, err error) {
	buf := make([]byte, len(p))
	n, err = r.Reader.Read(buf)
	if err != nil {
		return
	}
	n, err = 0, nil
	stream := toBitStream(buf)
	for err == nil && n < len(p) {
		p[n], err = decode(r.root, stream)
		n++
	}

	return n, err
}
