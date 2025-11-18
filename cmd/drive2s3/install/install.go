// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package install

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pluto-org-co/fsio/cmd/drive2s3/config"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

const (
	configDirectory = "/etc/drive2s3"
	configFile      = configDirectory + "/config.yaml"
)

const serviceFile = "/etc/systemd/system/drive2s3.service"
const binaryPath = "/usr/bin/drive2s3"
const serviceName = "drive2s3"
const username = "drive2s3"

var InstallCommand = &cli.Command{
	Name:        "install",
	Description: "install drive2s3 as a service in a debian based system",
	Action: func(ctx context.Context, c *cli.Command) (err error) {
		var systemd Systemd

		log.Println("Stopping live service")
		systemd.Stop(serviceName)

		// Create user
		exists, err := UserExists(username)
		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if !exists {
			log.Printf("User %s doesn't exists", username)
			err = CreateUserWithHome(username)
			if err != nil {
				return fmt.Errorf("failed to create username: %w", err)
			}
		} else {
			log.Printf("User %s already exists", username)
		}

		// Prepare executable
		log.Println("Preparing Binary")
		execContents, err := os.ReadFile(os.Args[0])
		if err != nil {
			return fmt.Errorf("failed to read executable contents: %w", err)
		}

		err = os.WriteFile(binaryPath, execContents, 0o755)
		if err != nil {
			return fmt.Errorf("failed to write executable: %w", err)
		}

		// Prepare configuration location
		log.Println("Preparing configuration directory")
		os.MkdirAll(configDirectory, 0o755)
		config, err := yaml.Marshal(config.Example)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}

		_, err = os.Stat(configFile)
		if err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("failed to get file stat: %w", err)
			}
			log.Println("Writing new configuration")
			err = os.WriteFile(configFile, config, 0o755)
			if err != nil {
				return fmt.Errorf("failed to write example configuration: %w", err)
			}
		} else {
			log.Println("Skipping configuration. Already exists")
		}

		// Enable
		log.Println("Updating configuration directory permissions")
		err = Chown(configDirectory, true, "root", username)
		if err != nil {
			return fmt.Errorf("failed to change ownership of configuration: %w", err)
		}

		// Write service file
		log.Println("Writing service file")
		err = os.WriteFile(serviceFile, serviceFilContents, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write relayer service: %w", err)
		}

		log.Println("Reload service daemon")
		err = systemd.DaemonReload()
		if err != nil {
			return fmt.Errorf("failed to reload daemon: %w", err)
		}

		log.Println("Enable service")
		err = systemd.Enable(serviceName)
		if err != nil {
			return fmt.Errorf("failed to enable servce: %w", err)
		}

		log.Println("Restart service")
		err = systemd.Restart(serviceName)
		if err != nil {
			return fmt.Errorf("failed to restart servce: %w", err)
		}
		return nil
	},
}
