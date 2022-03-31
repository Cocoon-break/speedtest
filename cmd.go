package speedtest

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
)

func ExecCmd(_cmd string, timeout int, args ...string) (stdOut, stdErr string, err error) {
	findcmd := cmd.NewCmd(_cmd, args...)
	statusChan := findcmd.Start()
	if timeout == 0 {
		timeout = 15
	}
	ticker := time.NewTicker(time.Duration(timeout) * time.Second)
	go func() {
		select {
		case <-findcmd.Done():
		case <-ticker.C:
			findcmd.Stop()
		}
	}()
	finalStatus := <-statusChan
	stdErr = strings.Join(finalStatus.Stderr, "\n")
	stdOut = strings.Join(finalStatus.Stdout, "\n")
	if finalStatus.Exit != 0 {
		errstr := fmt.Sprintf("exit code:%d", finalStatus.Exit)
		if finalStatus.Error != nil {
			errstr += fmt.Sprintf(" errmsg:%s", finalStatus.Error.Error())
		}
		return stdOut, stdErr, fmt.Errorf(errstr)
	}
	return stdOut, stdErr, nil

}
