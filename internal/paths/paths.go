// Package paths computes the on-disk layout used by the extractor, derived from
// a configurable output root. It mirrors the directory structure of the
// original program:
//
//	<root>/temp/files/<env>/<build>       transient downloaded files (local snapshot)
//	<root>/temp/work/<env>/<build>        work-in-progress output
//	<root>/buildfiles/<env>/<build>       persistent build files (remote incremental download)
//	<root>/publish/<env>/<build>          published output (served on the web)
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

// BuildFilesDir is the persistent location remote downloads accumulate in, so
// unchanged files can be skipped on the next build. Not cleared with temp.
func (l Layout) BuildFilesDir(env, build string) string {
	return filepath.Join(l.Root, "buildfiles", norm(env), norm(build))
}

// PublishDir is the published location for a given env/build.
func (l Layout) PublishDir(env, build string) string {
	return filepath.Join(l.Publish(), norm(env), norm(build))
}

func norm(s string) string { return strings.ToLower(s) }
