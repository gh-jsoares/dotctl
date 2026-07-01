package context

import "os/exec"

func runTmuxSetEnv(key, value string) {
	exec.Command("tmux", "set-environment", key, value).Run()
}
