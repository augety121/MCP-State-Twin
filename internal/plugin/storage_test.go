package plugin

import (
	"errors"
	"os"
	"testing"
)

type faultFile struct {
	*os.File
	fault string
}

func (f faultFile) Write(p []byte) (int, error) {
	if f.fault == "write" {
		return 0, errors.New("injected")
	}
	if f.fault == "short" {
		return f.File.Write(p[:len(p)/2])
	}
	return f.File.Write(p)
}
func (f faultFile) Sync() error {
	if f.fault == "sync" {
		return errors.New("injected")
	}
	return f.File.Sync()
}
func (f faultFile) Close() error {
	e := f.File.Close()
	if f.fault == "close" {
		return errors.New("injected")
	}
	return e
}
func TestPluginPublicationFaults(t *testing.T) {
	for _, fault := range []string{"write", "short", "sync", "close", "link", "remove"} {
		t.Run(fault, func(t *testing.T) {
			fs, e := os.OpenRoot(t.TempDir())
			if e != nil {
				t.Fatal(e)
			}
			defer fs.Close()
			fs.Mkdir("output", 0700)
			ops := &storeOps{create: func(n string) (durableFile, error) {
				f, e := fs.OpenFile(n, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				return faultFile{f, fault}, e
			}, link: fs.Link, remove: fs.Remove}
			if fault == "link" {
				ops.link = func(string, string) error { return errors.New("injected") }
			}
			if fault == "remove" {
				ops.remove = func(string) error { return errors.New("injected") }
			}
			s := &store{root: fs, out: "output", ops: ops}
			e = s.publish("report.json", map[string]string{"state": "complete"}, 4096)
			if e == nil {
				t.Fatal("failure hidden")
			}
			_, exists := fs.Stat("output/report.json")
			if fault == "remove" {
				if exists != nil || e.Error() != "PLUGIN_OUTPUT_RESIDUE" {
					t.Fatal(e, exists)
				}
			} else if !os.IsNotExist(exists) {
				t.Fatal("false publication", exists)
			}
			if _, e = fs.Stat("output/report.json.pending"); e != nil {
				t.Fatal("lost partial", e)
			}
		})
	}
}
