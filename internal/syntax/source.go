package syntax

import "bytes"

// Filter prepares raw source the way ReadSortedSourcecode does before parsing:
// every byte outside [A-Za-z.,] becomes a space.
//
// It also reproduces how the original reads the file, with fopen(..., "rt")
// and fgets(line, 1024, ...) under the Win32 C runtime: text mode turns CRLF
// into LF, a Ctrl-Z (0x1A) ends the file, and a NUL byte ends the line buffer,
// dropping the rest of that fgets chunk (up to and including its newline, or
// 1023 bytes in all).
func Filter(raw []byte) string {
	const chunk = 1023 // fgets(line, 1024, fp) reads at most 1023 bytes
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	out := make([]byte, 0, len(raw))
	for len(raw) > 0 {
		// One fgets call: up to and including '\n', at most chunk bytes,
		// and nothing at or after a Ctrl-Z.
		n := 0
		for n < len(raw) && n < chunk {
			if raw[n] == 0x1A {
				break
			}
			n++
			if raw[n-1] == '\n' {
				break
			}
		}
		line := raw[:n]
		eof := n < len(raw) && raw[n] == 0x1A
		for _, b := range line {
			if b == 0 {
				break
			}
			if isValidChar(b) {
				out = append(out, b)
			} else {
				out = append(out, ' ')
			}
		}
		if eof || n == 0 {
			break
		}
		raw = raw[n:]
	}
	return string(out)
}

func isChar(b byte) bool { return 'A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' }

func isValidChar(b byte) bool { return isChar(b) || b == '.' || b == ',' }
