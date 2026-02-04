package internal

import (
	"crypto/rand"
	"io"
)

func RandomID(n int) string {
	const characters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err) // unlikely to fail
	}
	for i := 0; i < n; i++ {
		b[i] = characters[int(b[i])%len(characters)]
	}
	return string(b)
}
