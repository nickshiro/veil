package conf

import (
	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/common/task"
	"github.com/nickshiro/veil/proxy/hysteria"
	"github.com/nickshiro/veil/proxy/hysteria/account"
	"google.golang.org/protobuf/proto"
)

type HysteriaClientConfig struct {
	Version int32    `json:"version"`
	Address *Address `json:"address"`
	Port    uint16   `json:"port"`
}

func (c *HysteriaClientConfig) Build() (proto.Message, error) {
	if c.Version != 2 {
		return nil, errors.New("version != 2")
	}

	config := &hysteria.ClientConfig{}
	config.Server = &protocol.ServerEndpoint{
		Address: c.Address.Build(),
		Port:    uint32(c.Port),
	}

	return config, nil
}

type HysteriaUserConfig struct {
	Auth  string `json:"auth"`
	Level uint32 `json:"level"`
	Email string `json:"email"`
}

type HysteriaServerConfig struct {
	Version int32                 `json:"version"`
	Users   []*HysteriaUserConfig `json:"users"`
	Clients []*HysteriaUserConfig `json:"clients"`
}

func (c *HysteriaServerConfig) Build() (proto.Message, error) {
	if c.Version != 2 {
		return nil, errors.New("version != 2")
	}

	config := new(hysteria.ServerConfig)

	if c.Clients != nil {
		c.Users = c.Clients
	}
	if len(c.Users) > 0 {
		config.Users = make([]*protocol.User, len(c.Users))
		processUser := func(idx int) error {
			user := c.Users[idx]
			acc := &account.Account{
				Auth: user.Auth,
			}
			config.Users[idx] = &protocol.User{
				Email:   user.Email,
				Level:   user.Level,
				Account: serial.ToTypedMessage(acc),
			}
			return nil
		}
		if err := task.ParallelForN(len(c.Users), processUser); err != nil {
			return nil, err
		}
	}

	return config, nil
}
