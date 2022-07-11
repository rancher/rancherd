package version

import (
	"fmt"
)

var (
	Version   = "dev"
	GitCommit = "Head"
)

func FriendlyVersion() string {
	return fmt.Sprintf("%s (%s)", Version, GitCommit)
}
