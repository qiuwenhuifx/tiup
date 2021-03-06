// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package instance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/c4pt0r/tiup/pkg/localdata"
	"github.com/c4pt0r/tiup/pkg/meta"
	"github.com/c4pt0r/tiup/pkg/utils"
	"github.com/pingcap/errors"
)

// TiKVInstance represent a running tikv-server
type TiKVInstance struct {
	id     int
	dir    string
	host   string
	port   int
	status int
	pds    []*PDInstance
	cmd    *exec.Cmd
}

// NewTiKVInstance return a TiKVInstance
func NewTiKVInstance(dir, host string, id int, pds []*PDInstance) *TiKVInstance {
	return &TiKVInstance{
		id:     id,
		dir:    dir,
		host:   host,
		port:   utils.MustGetFreePort(host, 20160),
		status: utils.MustGetFreePort(host, 20180),
		pds:    pds,
	}
}

// Start calls set inst.cmd and Start
func (inst *TiKVInstance) Start(ctx context.Context, version meta.Version) error {
	if err := os.MkdirAll(inst.dir, 0755); err != nil {
		return err
	}
	configPath := path.Join(inst.dir, "tikv.toml")
	cf, err := os.Create(configPath)
	if err != nil {
		return errors.Trace(err)
	}
	defer cf.Close()
	if err := writeConfig(cf); err != nil {
		return errors.Trace(err)
	}

	endpoints := make([]string, 0, len(inst.pds))
	for _, pd := range inst.pds {
		endpoints = append(endpoints, fmt.Sprintf("http://%s:%d", inst.host, pd.clientPort))
	}
	inst.cmd = exec.CommandContext(ctx,
		"tiup", "run", compVersion("tikv", version), "--",
		fmt.Sprintf("--addr=%s:%d", inst.host, inst.port),
		fmt.Sprintf("--status-addr=%s:%d", inst.host, inst.status),
		fmt.Sprintf("--pd=%s", strings.Join(endpoints, ",")),
		fmt.Sprintf("--config=%s", configPath),
		fmt.Sprintf("--data-dir=%s", filepath.Join(inst.dir, "data")),
		fmt.Sprintf("--log-file=%s", filepath.Join(inst.dir, "tikv.log")),
	)
	inst.cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("%s=%s", localdata.EnvNameInstanceDataDir, inst.dir),
	)
	inst.cmd.Stderr = os.Stderr
	inst.cmd.Stdout = os.Stdout
	return inst.cmd.Start()
}

// Wait calls inst.cmd.Wait
func (inst *TiKVInstance) Wait() error {
	return inst.cmd.Wait()
}

// Pid return the PID of the instance
func (inst *TiKVInstance) Pid() int {
	return inst.cmd.Process.Pid
}
