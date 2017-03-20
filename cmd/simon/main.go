//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/szabba/msc-thesis/assert"
)

var (
	prefix string
)

func main() {
	flag.StringVar(&prefix, "path", ".", "path prefix for the run info")
	flag.Parse()

	var ri RunInfo

	ri.ReadCommands(os.Stdin)
	ri.PopulateVersion()

	ri.RunAll(path(prefix))
}

type RunInfo struct {
	Revision string   `json:"revision"`
	Patch    []string `json:"patch"`

	Build string `json:"build"`
	Init  string `json:"init"`
	Run   string `json:"run"`
}

func (ri *RunInfo) ReadCommands(r io.Reader) {
	scn := bufio.NewScanner(r)

	assert.That(scn.Scan(), log.Fatalf, "unexpected error reading the build command: %s", scn.Err())
	ri.Build = scn.Text()

	assert.That(scn.Scan(), log.Fatalf, "unexpected error reading the init command: %s", scn.Err())
	ri.Init = scn.Text()

	assert.That(scn.Scan() || scn.Err() == io.EOF, log.Fatalf, "unexpected error reading run command: %s", scn.Err())
	ri.Run = scn.Text()
}

func (ri *RunInfo) PopulateVersion() {
	var err error

	ri.Revision, err = shellOut("git rev-parse HEAD")
	assert.That(err == nil, log.Fatalf, "can't retrieve git revision: %s", err)

	patch, err := shellOut("git diff -p")
	ri.Patch = strings.Split(patch, "\n")
	assert.That(err == nil, log.Fatalf, "can't generate git patch: %s", err)
}

func (ri *RunInfo) RunAll(dir string) {
	err := os.MkdirAll(dir, 0755)
	assert.That(err == nil, log.Fatalf, "can't create directory %q: %s", dir, err)

	ri.RunUptoRun(dir)
	ri.RunRun(dir)
}

func (ri *RunInfo) RunUptoRun(dir string) {
	ri.DumpSelf(dir)
	ri.RunBuild(dir)
	ri.RunInit(dir)
}

func (ri *RunInfo) DumpSelf(dir string) {
	path := filepath.Join(dir, "simon.json")
	f, err := os.Create(path)
	assert.That(err == nil, log.Fatalf, "can't create %q: %s", path, err)
	defer f.Close()

	dec := json.NewEncoder(f)
	dec.SetIndent("", "    ")
	err = dec.Encode(ri)
	assert.That(err == nil, log.Fatalf, "can't format run info: %s", err)
}

func (ri *RunInfo) RunBuild(dir string) {
	bld := exec.Command("sh", "-c", ri.Build)

	bldOutPath := filepath.Join(dir, "bld.out")
	bldOut, err := os.Create(bldOutPath)
	assert.That(err == nil, log.Fatalf, "can't create %q: %s", bldOutPath, err)
	defer bldOut.Close()
	bld.Stdout = bldOut

	bldErrPath := filepath.Join(dir, "bld.err")
	bldErr, err := os.Create(bldErrPath)
	assert.That(err == nil, log.Fatalf, "cant' create %q: %s", bldErrPath, err)
	defer bldErr.Close()
	bld.Stderr = bldErr

	err = bld.Run()
	assert.That(err == nil, log.Fatalf, "build failed: %q", err)
}

func (ri *RunInfo) RunInit(dir string) {
	ini := exec.Command("sh", "-c", ri.Init)

	iniOutPath := filepath.Join(dir, "ini.out")
	iniOut, err := os.Create(iniOutPath)
	assert.That(err == nil, log.Fatalf, "can't create %q: %s", iniOutPath, err)
	defer iniOut.Close()
	ini.Stdout = iniOut

	iniErrPath := filepath.Join(dir, "ini.err")
	iniErr, err := os.Create(iniErrPath)
	assert.That(err == nil, log.Fatalf, "can't create %q: %s", iniErrPath, err)
	defer iniErr.Close()
	ini.Stderr = iniErr

	err = ini.Run()
	assert.That(err == nil, log.Fatalf, "init failed: %s", err)
}

func (ri *RunInfo) RunRun(dir string) {
	cmd := exec.Command("sh", "-c", ri.Run)

	inPath := filepath.Join(dir, "ini.out")
	stdin, err := os.Open(inPath)
	assert.That(err == nil, log.Fatalf, "can't open %q: %s", inPath, err)
	defer stdin.Close()
	cmd.Stdin = stdin

	outPath := filepath.Join(dir, "run.out")
	stdout, err := os.Create(outPath)
	assert.That(err == nil, log.Fatalf, "can't create %q: %s", outPath, err)
	defer stdout.Close()
	cmd.Stdout = stdout

	errPath := filepath.Join(dir, "run.err")
	stderr, err := os.Create(errPath)
	assert.That(err == nil, log.Fatalf, "can't create %q: %s", errPath, err)
	defer stderr.Close()
	cmd.Stderr = stderr

	err = cmd.Run()
	assert.That(err == nil, log.Fatalf, "run failed: %s", err)
}

func path(prefix string) string {
	now := time.Now().UTC()
	day := now.Format("2006-01-02")
	hour := now.Format("15:04:05.000")
	return filepath.Join(prefix, day, hour)
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
