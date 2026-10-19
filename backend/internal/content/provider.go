package content

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Provider serves the current course and transparently reloads it when any file
// under the content directory changes (checked by path/mtime/size on each call).
type Provider struct {
	dir string
	log *slog.Logger

	mu     sync.Mutex
	sig    string
	course *Course
	err    error
}

func NewProvider(dir string, log *slog.Logger) *Provider {
	return &Provider{dir: dir, log: log}
}

// Current returns the latest valid course, or an ErrorList if the content on disk is invalid.
func (p *Provider) Current() (*Course, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	sig, err := signature(p.dir)
	if err != nil {
		return nil, fmt.Errorf("read content directory: %w", err)
	}
	if sig == p.sig && (p.course != nil || p.err != nil) {
		return p.course, p.err
	}

	reload := p.sig != ""
	p.sig = sig
	p.course, p.err = Load(os.DirFS(p.dir))
	if reload {
		if p.err != nil {
			p.log.Warn("content reloaded with errors", "count", len(p.err.(ErrorList)))
			LogErrors(p.log, p.err)
		} else {
			p.log.Info("content reloaded")
		}
	}
	return p.course, p.err
}

// LogErrors logs each content error as its own record.
func LogErrors(log *slog.Logger, err error) {
	if list, ok := err.(ErrorList); ok {
		for _, e := range list {
			log.Error("content error", "file", e.File, "problem", e.Message)
		}
		return
	}
	log.Error("content error", "err", err)
}

// signature fingerprints the content tree by file path, size and mtime.
func signature(dir string) (string, error) {
	h := fnv.New64a()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && path != dir {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s|%d|%d|%t\n", path, info.Size(), info.ModTime().UnixNano(), d.IsDir())
		return nil
	})
	return fmt.Sprintf("%x", h.Sum64()), err
}
