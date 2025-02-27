// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml"
)

type GCPHostConfig struct {
	MachineType    string
	MinCPUPlatform string
}

type HostConfig struct {
	GCP GCPHostConfig
}

type Config struct {
	ServiceURL           string
	Zone                 string
	HTTPProxy            string
	ConnectionControlDir string
	KeepLogFilesDays     int
	Host                 HostConfig
}

func (c *Config) ConnectionControlDirExpanded() string {
	return expandPath(c.ConnectionControlDir)
}

func (c *Config) LogFilesDeleteThreshold() time.Duration {
	return time.Duration(c.KeepLogFilesDays*24) * time.Hour
}

func DefaultConfig() Config {
	return Config{
		ConnectionControlDir: "~/.cvdr/connections",
		KeepLogFilesDays:     30, // A default is needed to not keep forever
	}
}

func ParseConfigFile(config *Config, confFile io.Reader) error {
	decoder := toml.NewDecoder(confFile)
	// Fail if there is some unknown configuration. This is better than silently
	// ignoring a (perhaps mispelled) config entry.
	decoder.Strict(true)
	err := decoder.Decode(config)
	if err != nil {
		return err
	}
	return nil
}

func expandPath(path string) string {
	if !strings.Contains(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("Unable to expand path %q: %v", path, err))
	}
	return strings.ReplaceAll(path, "~", home)
}
