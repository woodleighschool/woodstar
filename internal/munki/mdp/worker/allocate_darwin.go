package worker

import (
	"os"

	"golang.org/x/sys/unix"
)

func allocateFile(file *os.File, size int64) error {
	if size == 0 {
		return nil
	}
	return unix.FcntlFstore(file.Fd(), unix.F_PREALLOCATE, &unix.Fstore_t{
		Flags:   unix.F_ALLOCATEALL,
		Posmode: unix.F_PEOFPOSMODE,
		Length:  size,
	})
}
