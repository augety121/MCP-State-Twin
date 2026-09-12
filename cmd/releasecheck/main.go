// releasecheck is a maintainer-only, read-only release declaration validator.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/augety121/mcp-state-twin/internal/releasepolicy"
)

func run(args []string, out io.Writer) error {
	f := flag.NewFlagSet("releasecheck", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", ".", "trusted repository root")
	tag := f.String("tag", "", "reviewed release tag")
	format := f.String("format", "json", "json or github")
	if f.Parse(args) != nil || f.NArg() != 0 || (*format != "json" && *format != "github") {
		return errors.New("RELEASE_ARGUMENT_INVALID")
	}
	a, err := releasepolicy.Check(*root, *tag)
	if err != nil {
		return err
	}
	if *format == "github" {
		_, err = fmt.Fprintf(out, "tag=%s\nversion=%s\nprerelease=%t\nnotes=%s\n", a.Tag, a.Version, a.Prerelease, a.Notes)
		return err
	}
	return json.NewEncoder(out).Encode(a)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, diagnostic(err))
		os.Exit(1)
	}
}

func diagnostic(err error) string {
	switch err.Error() {
	case "RELEASE_ARGUMENT_INVALID", "RELEASE_TAG_INVALID", "RELEASE_PLAN_INVALID", "RELEASE_PLAN_MISMATCH_OR_UNSUPPORTED", "RELEASE_REVIEW_REQUIRED", "RELEASE_ROOT_UNAVAILABLE", "RELEASE_DIRECTORY_INVALID", "RELEASE_FILE_INVALID", "RELEASE_NOTES_INVALID":
		return err.Error()
	default:
		return "RELEASE_OUTPUT_FAILED"
	}
}
