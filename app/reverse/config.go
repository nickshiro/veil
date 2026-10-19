package reverse

import (
	"crypto/rand"
	"io"
	randv2 "math/rand/v2"
)

func (c *Control) FillInRandom() {
	randomLength := randv2.IntN(64)
	randomLength++
	c.Random = make([]byte, randomLength)
	io.ReadFull(rand.Reader, c.Random)
}
