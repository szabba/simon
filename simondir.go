//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"flag"
	"os"
	"path/filepath"
	"time"
)

type Simondir string

var _ flag.Value = new(Simondir)

func (dir *Simondir) Set(s string) error {
	abs, err := filepath.Abs(s)
	if err == nil {
		*dir = Simondir(abs)
	}
	return err
}

func (dir Simondir) String() string { return string(dir) }

func (dir Simondir) InJob(job LocatedJobSpec, parts ...string) string {
	prefix := dir.NormalizePath(job.Dir)
	allParts := append([]string{prefix}, parts...)
	return filepath.Join(allParts...)
}

func (dir Simondir) FreshJobPath() string {
	now := time.Now().UTC()
	day := now.Format("2006-01-02")
	hour := now.Format("15:04:05.000000000")
	return filepath.Join(dir.String(), day, hour)
}

func (dir Simondir) NormalizeJobPath(job LocatedJobSpec) string {
	abs, _ := filepath.Abs(job.Dir)
	suffix, err := filepath.Rel(dir.String(), abs)
	if err == nil {
		return suffix
	}
	cwd, _ := os.Getwd()
	suffix, _ = filepath.Rel(cwd, abs)
	return suffix
}

func (dir Simondir) NormalizePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir.String(), path)
}
