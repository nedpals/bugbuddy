package release

import (
	"fmt"

	"github.com/carlmjohnson/versioninfo"
)

func Version() string {
	return fmt.Sprintf("%s %s", versioninfo.Short(), versioninfo.Version)
}
