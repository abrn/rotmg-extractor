// Package paths computes the on-disk layout used by the extractor, derived from
// a configurable output root. It mirrors the directory structure of the
// original program:
//
//	<root>/temp/files/<env>/<build>     downloaded build files
//	<root>/temp/work/<env>/<build>      work-in-progress output
//	<root>/publish/<env>/<build>        published output (served on the web)
package paths

import (
	"path/filepath"
	"strings"
)

// Layout resolves output directories relative to a root.
type Layout struct {
	Root string
}

// New returns a Layout rooted at the given output directory.
func New(root string) Layout { return Layout{Root: root} }

// Temp is <root>/temp, cleared on each run.
func (l Layout) Temp() string { return filepath.Join(l.Root, "temp") }

// Publish is <root>/publish.
func (l Layout) Publish() string { return filepath.Join(l.Root, "publish") }

// FilesDir is the download location for a given env/build.
func (l Layout) FilesDir(env, build string) string {
	return filepath.Join(l.Temp(), "files", norm(env), norm(build))
}

// WorkDir is the work-in-progress location for a given env/build.
func (l Layout) WorkDir(env, build string) string {
	return filepath.Join(l.Temp(), "work", norm(env), norm(build))
}

// PublishDir is the published location for a given env/build.
func (l Layout) PublishDir(env, build string) string {
	return filepath.Join(l.Publish(), norm(env), norm(build))
}

func norm(s string) string { return strings.ToLower(s) }
