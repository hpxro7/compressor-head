// package bits provides simple wrappers for reading and writing streams of bytes and bits.
package bits

import "io"

//TODO(zac): fix reading extraneous bits at end of stream
// ToStream returns a function which is a stream over the input bytes. Calls to it returns
// the next bit in the stream.
func ToStream(input []byte) func() (byte, error) {
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
