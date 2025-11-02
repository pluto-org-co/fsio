package install

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

type Systemd struct {
}

// Stops service
func (s *Systemd) Stop(name string) (err error) {
	return exec.Command("/usr/bin/systemctl", "stop", name).Run()
}

// Starts the service
func (s *Systemd) Restart(name string) (err error) {
	return exec.Command("/usr/bin/systemctl", "restart", name).Run()
}

// Enables the service
func (s *Systemd) Enable(name string) (err error) {
	return exec.Command("/usr/bin/systemctl", "enable", name).Run()
}

// Disables the service
func (s *Systemd) Disable(name string) (err error) {
	return exec.Command("/usr/bin/systemctl", "disable", name).Run()
}

// Starts the service
func (s *Systemd) DaemonReload() (err error) {
	return exec.Command("/usr/bin/systemctl", "daemon-reload").Run()
}

func Chown(path string, recursive bool, username, group string) (err error) {
	args := []string{
		username + ":" + group,
		path,
	}
	if recursive {
		args = append(args, "-R")
	}
	return exec.Command("chown", args...).Run()
}

func CreateUserWithHome(username string) (err error) {
	return exec.Command("/sbin/useradd", "-m", username).Run()
}

// Check if user exists
func UserExists(username string) (bool, error) {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) > 0 && parts[0] == username {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}
