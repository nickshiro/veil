package dice_test

import (
	"math/rand"
	"testing"

	"github.com/nickshiro/veil/common/dice"
)

// The seeded generator must retain its sequence, including not advancing for n=1.
func TestDeterministicSequence(t *testing.T) {
	for _, seed := range []int64{0, 1, -1, 123456789} {
		got := dice.NewDeterministicDice(seed)
		want := rand.New(rand.NewSource(seed))
		for range 100 {
			for _, n := range []int{1, 2, 1, 16, 256, 1000} {
				expected := 0
				if n != 1 {
					expected = want.Intn(n)
				}
				if actual := got.Roll(n); actual != expected {
					t.Fatalf("seed=%d n=%d: got %d, want %d", seed, n, actual, expected)
				}
			}
		}
	}
}
