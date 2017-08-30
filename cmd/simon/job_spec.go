//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

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

func (spec *JobSpec) PopulateVersion() error {
	var err error

	spec.Revision, err = shellOut("git rev-parse HEAD")
	if err != nil {
		return errors.Wrap(err, "can't obtain git revision")
	}

	patch, err := shellOut("git diff -p")
	if err != nil {
		return errors.Wrap(err, "can't obtain git diff")
	}

	spec.Patch = strings.Split(patch, "\n")
	return nil
}

func (spec JobSpec) Locate(path string) LocatedJobSpec {
	return LocatedJobSpec{Dir: path, JobSpec: spec}
}

func (job *LocatedJobSpec) Load(simondir Simondir, name string) error {
	job.Dir = name

	specPath := simondir.InJob(*job, specFileName)
	specFile, err := os.Open(specPath)
	err = errors.Wrap(err, "can't open job spec")

	if err == nil {
		defer specFile.Close()
		dec := json.NewDecoder(specFile)
		err = dec.Decode(&job.JobSpec)
		err = errors.Wrapf(err, "can't decode spec file %q", specPath)
	}

	return errors.Wrapf(err, "can't load job %q", name)
}

func (spec LocatedJobSpec) EnsureLocationExists() error {
	return errors.Wrapf(os.MkdirAll(spec.Dir, 0755), "can't create directory %q: %s", spec.Dir)
}

func (spec LocatedJobSpec) StoreDefinition(dir Simondir) error {
	f, err := os.Create(dir.InJob(spec, specFileName))
	if err != nil {
		return errors.Wrapf(err, "can't create job spec file")
	}
	defer f.Close()

	dec := json.NewEncoder(f)
	dec.SetIndent("", "    ")

	return errors.Wrap(dec.Encode(spec.JobSpec), "can't encode job spec")
}

func (spec LocatedJobSpec) Build(ctx context.Context, dir Simondir) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", spec.BuildCmd)

	var err error

	cmd.Stdout, err = os.Create(dir.InJob(spec, "bld.out"))
	err = errors.Wrap(err, "can't create file")

	if err == nil {
		defer Close(cmd.Stdout)

		cmd.Stderr, err = os.Create(dir.InJob(spec, "bld.err"))
		err = errors.Wrap(err, "can't create file")
	}

	if err == nil {
		defer Close(cmd.Stderr)

		log.Printf("job %q: running build command: %s", spec.Dir, spec.BuildCmd)
		err = cmd.Run()
	}

	return errors.Wrapf(err, "job %q: build failed", spec.Dir)
}

func (spec LocatedJobSpec) Init(ctx context.Context, dir Simondir) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", spec.InitCmd)

	var err error

	cmd.Stdout, err = os.Create(dir.InJob(spec, "ini.out"))
	err = errors.Wrap(err, "can't create file")

	if err == nil {
		defer Close(cmd.Stdout)

		cmd.Stderr, err = os.Create(dir.InJob(spec, "ini.err"))
		err = errors.Wrap(err, "can't create file")
	}

	if err == nil {
		defer Close(cmd.Stderr)

		log.Printf("job %q: running init command: %s", spec.Dir, spec.InitCmd)
		err = cmd.Run()
	}

	return errors.Wrapf(err, "job %q: init failed", spec.Dir)
}

func (spec LocatedJobSpec) Run(ctx context.Context, dir Simondir) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", spec.RunCmd)

	var err error

	cmd.Stdin, err = os.Open(dir.InJob(spec, "ini.out"))
	err = errors.Wrap(err, "can't access initial conditions")

	if err == nil {
		defer Close(cmd.Stdin)

		cmd.Stdout, err = os.Create(dir.InJob(spec, "run.out"))
		err = errors.Wrap(err, "can't create file")
	}

	if err == nil {
		defer Close(cmd.Stdout)

		cmd.Stderr, err = os.Create(dir.InJob(spec, "run.err"))
		err = errors.Wrap(err, "can't create file")
	}

	if err == nil {
		defer Close(cmd.Stderr)

		log.Printf("job %q: running run command: %s", spec.Dir, spec.RunCmd)
		err = cmd.Run()
	}

	return errors.Wrapf(err, "job %q: run failed", spec.Dir)
}

func Close(closer interface{}) {
	switch closer := closer.(type) {
	case io.Closer:
		closer.Close()
	}
}

func shellOut(cmd string) (string, error) {
	bs, err := exec.Command("sh", "-c", cmd).Output()
	return string(bs), err
}

const specFileName = "job_spec.json"
