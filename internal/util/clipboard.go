package util

import (
	"os/exec"
	"runtime"
)

// CopyToClipboard writes text to the system clipboard.
// Returns an error if no clipboard tool is available.
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return nil
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := pipe.Write([]byte(text)); err != nil {
		return err
	}
	pipe.Close()
	return cmd.Wait()
}
