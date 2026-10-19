package all

import (
	"fmt"
	"uuid"

	"github.com/xtls/xray-core/main/commands/base"
)

var cmdUUID = &base.Command{
	UsageLine: `{{.Exec}} uuid`,
	Short:     `Generate a random UUIDv4`,
	Long: `
Generate a random UUIDv4: {{.Exec}} uuid
`,
}

func init() {
	cmdUUID.Run = executeUUID // break init loop
}

func executeUUID(cmd *base.Command, args []string) {
	fmt.Println(uuid.NewV4())
}
