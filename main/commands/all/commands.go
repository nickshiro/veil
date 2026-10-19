package all

import (
	"github.com/nickshiro/veil/main/commands/all/api"
	"github.com/nickshiro/veil/main/commands/all/convert"
	"github.com/nickshiro/veil/main/commands/all/tls"
	"github.com/nickshiro/veil/main/commands/base"
)

func init() {
	base.RootCommand.Commands = append(
		base.RootCommand.Commands,
		api.CmdAPI,
		convert.CmdConvert,
		tls.CmdTLS,
	)
}
