package protocol_test

import (
	"testing"
	"uuid"

	. "github.com/xtls/xray-core/common/protocol"
)

func TestIdEquals(t *testing.T) {
	id1 := NewID(uuid.NewV4())
	id2 := NewID(id1.UUID())

	if !id1.Equals(id2) {
		t.Error("expected id1 to equal id2, but actually not")
	}

	if id1.String() != id2.String() {
		t.Error(id1.String(), " != ", id2.String())
	}
}
