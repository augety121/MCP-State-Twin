package agenteval

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func pluginPreparationInputs(t testing.TB, root, id string) (*task.Task, []byte, *Witness) {
	t.Helper()
	read := func(name string) []byte {
		raw, err := os.ReadFile(filepath.Join(root, "issue-tracker", name))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	ta, err := task.Decode(read("agent-tasks/" + id + ".json"))
	if err != nil {
		t.Fatal(err)
	}
	witness, err := DecodeWitness(read("agent-witnesses/" + id + ".json"))
	if err != nil {
		t.Fatal(err)
	}
	return ta, read(ta.Bundle), witness
}

func TestPluginPreparedWorldDetachesCallerInputs(t *testing.T) {
	root := testfixture.Baseline(t)
	ta, raw, witness := pluginPreparationInputs(t, root, "close-issue")
	before, err := json.Marshal(ta)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := task.Decode(before)
	if err != nil {
		t.Fatal(err)
	}
	originalBundle := bytes.Clone(raw)
	ctx := context.Background()
	w, err := NewPluginWorld(ctx, ta, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Finish("cancelled", nil)
	ta.Oracle[0].Expr = "false"
	ta.ExpectedOutcome = "task_failed"
	ta.Authority[0].Equals["owner"] = "changed"
	ta.Tools[0] = "changed"
	raw[0] ^= 0xff
	for _, call := range witness.Calls {
		e, err := w.Call(ctx, call)
		if err != nil || !e.Authorized || !e.EffectCommitted {
			t.Fatal("caller mutated active authority or world", e, err)
		}
	}
	terminal, err := w.Finish("completed", witness.Answer)
	if err != nil || terminal.Evaluation.Outcome != "success" || !same(terminal.Task, expected) {
		t.Fatal("caller rewrote the admitted oracle or terminal", terminal, err)
	}
	if err = VerifyPluginTerminal(ctx, terminal, expected, originalBundle); err != nil {
		t.Fatal(err)
	}
}

func TestPluginPreparedWorldsKeepStateAndFaultsIndependent(t *testing.T) {
	root := testfixture.Baseline(t)
	ta, raw, witness := pluginPreparationInputs(t, root, "after-commit-confirm")
	ctx := context.Background()
	first, err := NewPluginWorld(ctx, ta, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Finish("cancelled", nil)
	second, err := NewPluginWorld(ctx, ta, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Finish("cancelled", nil)
	run := func(w *PluginWorld) *PluginTerminal {
		for i, call := range witness.Calls {
			e, err := w.Call(ctx, call)
			if err != nil || (i == 0 && (e.ErrorClass != "TIMEOUT_AFTER_EFFECT" || !e.EffectCommitted)) {
				t.Fatal("world lost its independent state or fault plan", e, err)
			}
		}
		terminal, err := w.Finish("completed", witness.Answer)
		if err != nil || terminal.Evaluation.Outcome != "success" {
			t.Fatal(terminal, err)
		}
		return terminal
	}
	a := run(first)
	// Runtime.Spec exposes a mutable pointer. A completed world must not retain
	// aliases into another world's Task or TwinSpec when preparation is reused.
	first.env.runtime.Spec().Metadata.Name = "changed"
	first.terminal.Task.Authority[0].Equals["owner"] = "changed"
	if second.env.runtime.Spec().Metadata.Name != ta.Domain {
		t.Fatal("TwinSpec was shared across worlds")
	}
	b := run(second)
	if !same(a, b) {
		t.Fatal("independent worlds produced different evidence")
	}
	if err = VerifyPluginTerminal(ctx, b, ta, raw); err != nil {
		t.Fatal("replay did not create another fresh world", err)
	}
}

func TestPluginPreparationRetainsStrictAdmissionAndArchiveEquivalence(t *testing.T) {
	root := testfixture.Baseline(t)
	ta, raw, witness := pluginPreparationInputs(t, root, "close-issue")
	b, err := bundle.OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := task.Decode(mustTaskJSON(t, ta))
	if err != nil {
		t.Fatal(err)
	}
	invalid.Oracle[0].Expr = "missing_function()"
	if Admit(invalid, b) == nil {
		t.Fatal("static admission accepted an invalid oracle")
	}
	ctx := context.Background()
	if _, err = RunWitness(ctx, invalid, b, witness); err == nil {
		t.Fatal("witness accepted an invalid oracle")
	}
	if _, err = NewPluginWorld(ctx, invalid, raw); err == nil {
		t.Fatal("world accepted an invalid oracle")
	}
	w, err := NewPluginWorld(ctx, ta, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Finish("cancelled", nil)
	for _, call := range witness.Calls {
		if _, err = w.Call(ctx, call); err != nil {
			t.Fatal(err)
		}
	}
	terminal, err := w.Finish("completed", witness.Answer)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	var repacked bytes.Buffer
	writer := zip.NewWriter(&repacked)
	if err = writer.SetComment("same admitted members, different archive container"); err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if err = writer.Copy(file); err != nil {
			t.Fatal(err)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	terminal.Bundle = base64.StdEncoding.EncodeToString(repacked.Bytes())
	if bytes.Equal(raw, repacked.Bytes()) || VerifyPluginTerminal(ctx, terminal, ta, raw) != nil {
		t.Fatal("equivalent archive container was rejected")
	}
	terminal.Task.Oracle[0].Expr = "true"
	if VerifyPluginTerminal(ctx, terminal, ta, raw) == nil {
		t.Fatal("terminal oracle rewrite was accepted")
	}
}

func TestPreparedWitnessPreservesNumericAuthorityTypes(t *testing.T) {
	root := testfixture.Baseline(t)
	ta, raw, witness := pluginPreparationInputs(t, root, "close-issue")
	b, err := bundle.OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	const large = 9007199254740992
	for i := range ta.Authority {
		if ta.Authority[i].Tool == "close_issue" {
			ta.Authority[i].Equals["number"] = float64(large)
		}
	}
	witness.Calls[0].Input["number"] = int64(large)
	if ta.Authorized(witness.Calls[0].Tool, witness.Calls[0].Input) {
		t.Fatal("fixture must distinguish a large float from an integer")
	}
	r, err := RunWitness(context.Background(), ta, b, witness)
	if err != nil || len(r.View.Events) != 1 || r.View.Events[0].Authorized || r.View.Events[0].Dispatched || r.View.Events[0].ErrorClass != "AUTHORITY_DENIED" || r.Evaluation.Outcome != "policy_violation" {
		t.Fatal("detachment changed permission semantics", r, err)
	}
}

func mustTaskJSON(t testing.TB, ta *task.Task) []byte {
	t.Helper()
	raw, err := json.Marshal(ta)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
