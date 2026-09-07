package main

import (
	"context"
	"errors"
	"flag"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func runTask(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("task requires validate or witness")
	}
	command := args[0]
	if command != "validate" && command != "witness" {
		return errors.New("unsupported task command")
	}
	flags := flag.NewFlagSet("task "+command, flag.ContinueOnError)
	root := flags.String("root", ".", "trusted artifact root")
	name := flags.String("task", "", "relative Task path")
	witnessName := flags.String("witness", "", "relative synthetic witness path")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected task arguments")
	}
	if command == "validate" && *witnessName != "" {
		return errors.New("validate does not accept a witness")
	}
	t, b, err := agenteval.Load(*root, *name)
	if err != nil {
		return err
	}
	if command == "validate" {
		return printJSON(map[string]any{"kind": "AgentTaskAdmission", "taskId": t.ID, "status": "structurally-valid", "solvability": "not-verified", "live": false})
	}
	data, err := task.ReadFile(*root, *witnessName, task.MaxBytes)
	if err != nil {
		return err
	}
	w, err := agenteval.DecodeWitness(data)
	if err != nil {
		return err
	}
	r, err := agenteval.RunWitness(ctx, t, b, w)
	if err != nil {
		return err
	}
	if err = printJSON(r); err != nil {
		return err
	}
	if r.Evaluation.Outcome != "success" && r.Evaluation.Outcome != "expected_abstention" {
		return errors.New("task witness did not pass independent evaluation")
	}
	return nil
}
