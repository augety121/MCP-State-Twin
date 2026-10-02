package plugin

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/task"
)

type store struct {
	root *os.Root
	out  string
	ops  *storeOps
}

type durableFile interface {
	Write([]byte) (int, error)
	Sync() error
	Close() error
}
type storeOps struct {
	create func(string) (durableFile, error)
	link   func(string, string) error
	remove func(string) error
}

func claim(root, out string, plan Plan) (*store, error) {
	if task.PortablePath(out) != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("PLUGIN_OUTPUT_FAILED")
	}
	s := &store{root: r, out: out}
	ok := false
	defer func() {
		if !ok {
			r.Close()
		}
	}()
	parts := strings.Split(out, "/")
	for i := 1; i < len(parts); i++ {
		info, e := r.Lstat(strings.Join(parts[:i], "/"))
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("PLUGIN_OUTPUT_FAILED")
		}
	}
	if err = r.Mkdir(out, 0700); err != nil {
		return nil, errors.New("PLUGIN_OUTPUT_EXISTS")
	}
	if err = s.write("claim.json", plan, 64<<10); err != nil {
		return nil, err
	}
	ok = true
	return s, nil
}
func (s *store) write(name string, v any, limit int) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil || len(raw) > limit || logging.ContainsSensitive(string(raw)) {
		return errors.New("PLUGIN_OUTPUT_FAILED")
	}
	var f durableFile
	if s.ops != nil {
		f, err = s.ops.create(path.Join(s.out, name))
	} else {
		f, err = s.root.OpenFile(path.Join(s.out, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	}
	if err != nil {
		return errors.New("PLUGIN_OUTPUT_FAILED")
	}
	n, err := f.Write(raw)
	if err == nil && n != len(raw) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	closed := f.Close()
	if err != nil || closed != nil {
		return errors.New("PLUGIN_OUTPUT_FAILED")
	}
	return nil
}
func (s *store) publish(name string, v any, limit int) error {
	if err := s.write(name+".pending", v, limit); err != nil {
		return err
	}
	link, remove := s.root.Link, s.root.Remove
	if s.ops != nil {
		link, remove = s.ops.link, s.ops.remove
	}
	if err := link(path.Join(s.out, name+".pending"), path.Join(s.out, name)); err != nil {
		return errors.New("PLUGIN_OUTPUT_FAILED")
	}
	if err := remove(path.Join(s.out, name+".pending")); err != nil {
		return errors.New("PLUGIN_OUTPUT_RESIDUE")
	}
	return nil
}
