package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/hpxro7/compressor-head/huffman"
)

func main() {
	dist := huffman.NewDistribution([]byte("the quick brown fox jumped over the lazy dog"))

	tmpName := "/tmp/1234"

	f, err := os.Create(tmpName)
	if err != nil {
		log.Fatal(err)
	}

	w := huffman.NewWriter(f, dist)
	input := []byte("the quick brown fox jumped over the lazy dog")
	n, err := w.Write(input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Wrote:", n, "Originally:", len(input))
	fmt.Printf("Compression rate: %.2f%%\n", 100*float32(len(input)-n)/float32(len(input)))

	f, err = os.Open(tmpName)
	if err != nil {
		log.Fatal(err)
	}

	r := huffman.NewReader(f, w.Huffman())
	rBuf := make([]byte, len(input))
	n, err = r.Read(rBuf)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	fmt.Println("Decompressed into:", string(rBuf))
	f.Close()

	err = os.Remove(tmpName)
	if err != nil {
		log.Fatal(err)
	}

}
