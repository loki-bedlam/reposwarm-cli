//go:build darwin

package commands

import (
	"os"
	"golang.org/x/sys/unix"
)

func makeRaw(fd int) (*unix.Termios, error) {
	oldState, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return nil, err
	}
	newState := *oldState
	newState.Lflag &^= unix.ECHO | unix.ICANON
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &newState); err != nil {
		return nil, err
	}
	return oldState, nil
}

func restoreTerminal(fd int, state *unix.Termios) {
	unix.IoctlSetTermios(fd, unix.TIOCSETA, state)
}

func stdinFd() int {
	return int(os.Stdin.Fd())
}
