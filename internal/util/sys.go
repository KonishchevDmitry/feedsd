package util

import "os"

func IsContainer() bool {
	return os.Getpid() == 1
}
