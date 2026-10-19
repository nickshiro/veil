package all

import (
	"github.com/nickshiro/veil/main/commands/all/api"
	"github.com/nickshiro/veil/main/commands/all/base64"
	"github.com/nickshiro/veil/main/commands/all/convert"
	"github.com/nickshiro/veil/main/commands/all/tls"
	"github.com/nickshiro/veil/main/commands/base"
)

func init() {
	base.RootCommand.Commands = append(
		base.RootCommand.Commands,
		api.CmdAPI,
		base64.CmdBase64,
		convert.CmdConvert,
		tls.CmdTLS,
		cmdX25519,
		// cmdEd25519,
		cmdWG,
		cmdMLDSA65,
		cmdMLKEM768,
		cmdVLESSEnc,
	)
}
