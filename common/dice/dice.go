// Package dice provides a reproducible random number generator.
package dice // import "github.com/nickshiro/veil/common/dice"

import (
	"math/rand"
)

func NewDeterministicDice(seed int64) *DeterministicDice {
	return &DeterministicDice{rand.New(rand.NewSource(seed))}
}

type DeterministicDice struct {
	*rand.Rand
}

func (dd *DeterministicDice) Roll(n int) int {
	if n == 1 {
		return 0
	}
	return dd.Intn(n)
}
