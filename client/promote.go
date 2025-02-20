package client

import (
	"fmt"

	"github.com/hozalex/knox"
)

var cmdPromote = &Command{
	Run:       runPromote,
	UsageLine: "promote <key_identifier> <key_version>",
	Short:     "promotes a key to primary state",
	Long: `
Promote will take an active key version and make it the primary key version. This also makes the current primary key active.

To use this command, you must have write permissions on the key.

For more about knox, see https://github.com/pinterest/knox.

See also: knox reactivate, knox deactivate
	`,
}

func runPromote(cmd *Command, args []string) {
	if len(args) != 2 {
		fatalf("promote takes exactly two argument. See 'knox help promote'")
	}
	keyID := args[0]
	versionID := args[1]

	err := cli.UpdateVersion(keyID, versionID, knox.Primary)
	if err != nil {
		fatalf("Error promoting version: %s", err.Error())
	}
	fmt.Printf("Promoted %s successfully.\n", versionID)
}
