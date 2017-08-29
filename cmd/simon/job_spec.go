//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/szabba/msc-thesis/assert"
)

type JobSpec struct {
	Revision string   `json:"revision"`
	Patch    []string `json:"patch"`

	BuildCmd string `json:"build"`
	InitCmd  string `json:"init"`
	RunCmd   string `json:"run"`
}

type LocatedJobSpec struct {
	Dir string

	JobSpec
}

func (spec *JobSpec) ReadCommands(r io.Reader) {
	scn := bufio.NewScanner(r)

	assert.That(scn.Scan(), log.Fatalf, "unexpected error reading the build command: %s", scn.Err())
	spec.BuildCmd = scn.Text()

	assert.That(scn.Scan(), log.Fatalf, "unexpected error reading the init command: %s", scn.Err())
	spec.InitCmd = scn.Text()

	assert.That(scn.Scan() || scn.Err() == io.EOF, log.Fatalf, "unexpected error reading run command: %s", scn.Err())
	spec.RunCmd = scn.Text()
}

func (spec *JobSpec) PopulateVersion() {
	var err error

	spec.Revision, err = shellOut("git rev-parse HEAD")
	assert.That(err == nil, log.Fatalf, "can't retrieve git revision: %s", err)

	patch, err := shellOut("git diff -p")
	spec.Patch = strings.Split(patch, "\n")
	assert.That(err == nil, log.Fatalf, "can't generate git patch: %s", err)
}

func (spec JobSpec) Locate(path string) LocatedJobSpec {
	return LocatedJobSpec{Dir: path, JobSpec: spec}
}

func (spec LocatedJobSpec) EnsureLocationExists() {
	err := os.MkdirAll(spec.Dir, 0755)
	assert.That(err == nil, log.Fatalf, "can't create directory %q: %s", spec.Dir, err)
}

func (spec LocatedJobSpec) StoreDefinition() {
	f := spec.create(specFileName)
	defer f.Close()

	dec := json.NewEncoder(f)
	dec.SetIndent("", "    ")

	err := dec.Encode(spec.JobSpec)
	assert.That(err == nil, log.Fatalf, "can't format run info: %s", err)
}

func (spec LocatedJobSpec) Build() {
	cmd := exec.Command("sh", "-c", spec.BuildCmd)

	stdout := spec.create("bld.out")
	defer stdout.Close()
	cmd.Stdout = stdout

	stderr := spec.create("bld.err")
	defer stderr.Close()
	cmd.Stderr = stderr

	err := cmd.Run()
	assert.That(err == nil, log.Fatalf, "build failed: %q", err)
}

func (spec LocatedJobSpec) Init() {
	cmd := exec.Command("sh", "-c", spec.InitCmd)

	stdout := spec.create("ini.out")
	defer stdout.Close()
	cmd.Stdout = stdout

	stder := spec.create("ini.err")
	defer stder.Close()
	cmd.Stderr = stder

	err := cmd.Run()
	assert.That(err == nil, log.Fatalf, "init failed: %s", err)
}

func (spec LocatedJobSpec) Run(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "sh", "-c", spec.RunCmd)

	stdin := spec.open("ini.out")
	defer stdin.Close()
	cmd.Stdin = stdin

	stdout := spec.create("run.out")
	defer stdout.Close()
	cmd.Stdout = stdout

	stderr := spec.create("run.err")
	defer stderr.Close()
	cmd.Stderr = stderr

	err := cmd.Run()
	assert.That(err == nil, log.Fatalf, "init failed: %s", err)
}

func (spec LocatedJobSpec) create(path string) *os.File {
	prefixedPath := spec.prefixPath(path)
	f, err := os.Create(prefixedPath)
	assert.That(err == nil, log.Fatalf, "can't create %q: %s", prefixedPath, err)
	return f
}

func (spec LocatedJobSpec) open(path string) *os.File {
	prefixedPath := spec.prefixPath(path)
	f, err := os.Open(prefixedPath)
	assert.That(err == nil, log.Fatalf, "can't open %q: %s", prefixedPath, err)
	return f
}

func (spec LocatedJobSpec) prefixPath(path string) string {
	return filepath.Join(spec.Dir, path)
}

func shellOut(cmd string) (string, error) {
	proc := exec.Command("sh", "-c", cmd)
	var out bytes.Buffer
	proc.Stdout = &out
	proc.Stderr = os.Stderr
	err := proc.Run()

	if err == nil && !proc.ProcessState.Success() {
		return "", ErrFailedSubcommand
	}
	return out.String(), nil
}

var ErrFailedSubcommand = errors.New("failed subcommand")

const specFileName = "job_spec.json"
