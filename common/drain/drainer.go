package drain

import (
	"io"
	"math/rand"
	randv2 "math/rand/v2"

	"github.com/nickshiro/veil/common/errors"
)

type BehaviorSeedLimitedDrainer struct {
	DrainSize int
}

func NewBehaviorSeedLimitedDrainer(behaviorSeed int64, drainFoundation, maxBaseDrainSize, maxRandDrain int) (Drainer, error) {
	behaviorRand := rand.New(rand.NewSource(behaviorSeed))
	BaseDrainSize := roll(behaviorRand, maxBaseDrainSize)
	RandDrainMax := roll(behaviorRand, maxRandDrain) + 1
	RandDrainRolled := randv2.IntN(RandDrainMax)
	DrainSize := drainFoundation + BaseDrainSize + RandDrainRolled
	return &BehaviorSeedLimitedDrainer{DrainSize: DrainSize}, nil
}

// roll preserves the seeded sequence by not advancing it when n is 1.
func roll(r *rand.Rand, n int) int {
	if n == 1 {
		return 0
	}
	return r.Intn(n)
}

func (d *BehaviorSeedLimitedDrainer) AcknowledgeReceive(size int) {
	d.DrainSize -= size
}

func (d *BehaviorSeedLimitedDrainer) Drain(reader io.Reader) error {
	if d.DrainSize > 0 {
		err := drainReadN(reader, d.DrainSize)
		if err == nil {
			return errors.New("drained connection")
		}
		return errors.New("unable to drain connection").Base(err)
	}
	return nil
}

func drainReadN(reader io.Reader, n int) error {
	_, err := io.CopyN(io.Discard, reader, int64(n))
	return err
}

func WithError(drainer Drainer, reader io.Reader, err error) error {
	drainErr := drainer.Drain(reader)
	if drainErr == nil {
		return err
	}
	return errors.New(drainErr).Base(err)
}

type NopDrainer struct{}

func (n NopDrainer) AcknowledgeReceive(size int) {
}

func (n NopDrainer) Drain(reader io.Reader) error {
	return nil
}

func NewNopDrainer() Drainer {
	return &NopDrainer{}
}
