package vmess_test

import (
	"testing"
	"uuid"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/protocol"
	. "github.com/nickshiro/veil/proxy/vmess"
)

func toAccount(a *Account) protocol.Account {
	account, err := a.AsAccount()
	common.Must(err)
	return account
}

func BenchmarkUserValidator(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := NewTimedUserValidator()

		for range 1500 {
			id := uuid.NewV4()
			v.Add(&protocol.MemoryUser{
				Email: "test",
				Account: toAccount(&Account{
					Id: id.String(),
				}),
			})
		}

		common.Close(v)
	}
}
