package main

import "os"

func main() {
	os.Exit(run())
}

func run() int {
	cmd := newRootCmd(defaultCommandDeps())
	if err := cmd.Execute(); err != nil {
		_, _ = cmd.ErrOrStderr().Write([]byte("Error: " + err.Error() + "\n"))
		return 1
	}

	return 0
}
