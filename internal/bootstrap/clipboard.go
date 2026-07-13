package bootstrap

import (
	"fmt"
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch {
	case execExists("pbcopy"):
		cmd = exec.Command("pbcopy")
	case execExists("wl-copy"):
		cmd = exec.Command("wl-copy")
	case execExists("xclip"):
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case execExists("xsel"):
		cmd = exec.Command("xsel", "--clipboard", "--input")
	default:
		return fmt.Errorf("no clipboard tool found")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func execExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
