package agenteval

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const ProjectFormat = "statetwin.dev/evaluation-project/v1alpha1"
const ProjectProfile = "offline-reviewed-project-v1"
const projectInputLimit = 128 << 20
const projectReportLimit = 1 << 20

type ProjectManifest struct {
	Format       string `json:"format"`
	Profile      string `json:"profile"`
	ID           string `json:"id"`
	Suite        string `json:"suite"`
	Expectation  string `json:"expectation"`
	TaskCatalog  string `json:"taskCatalog"`
	WorldCatalog string `json:"worldCatalog"`
	Cases        string `json:"cases"`
	Policy       string `json:"policy"`
}
type ProjectStage struct {
	Stage      string `json:"stage"`
	Status     string `json:"status"`
	ReasonCode string `json:"reasonCode"`
}
type ProjectCheck struct {
	Format             string         `json:"format"`
	ProjectID          string         `json:"projectId"`
	Status             string         `json:"status"`
	Stages             []ProjectStage `json:"stages"`
	PlannedCases       int            `json:"plannedCases"`
	PlannedPairs       int            `json:"plannedPairs"`
	PlannedTrials      int            `json:"plannedTrials"`
	ExecutionPerformed bool           `json:"executionPerformed"`
	Provenance         string         `json:"provenance"`
}
type frozenInput struct {
	raw  []byte
	info os.FileInfo
}

// Private, operation-local caches are keyed by rooted path and backing slice,
// not by content digests. Nothing mutable is returned to public callers.
type projectSource struct {
	failure                      error
	ctx                          context.Context
	root                         string
	out                          string
	files                        map[string]frozenInput
	bundles                      map[*byte]*bundle.Artifact
	rawLeft, extractedLeft       int
	reads, decodes               int
	noCache                      bool
	projectRaw, projectExtracted int
	projectFiles                 map[string]bool
	projectBundles               map[*byte]bool
}

func newProjectSource(ctx context.Context, root, out string, limit int) *projectSource {
	return &projectSource{ctx: ctx, root: root, out: out, files: map[string]frozenInput{}, bundles: map[*byte]*bundle.Artifact{}, rawLeft: limit, extractedLeft: limit}
}
func (s *projectSource) read(prefix, name string, limit int) (data []byte, failure error) {
	defer func() {
		if failure != nil {
			s.failure = failure
		}
	}()
	if s.ctx.Err() != nil {
		return nil, s.ctx.Err()
	}
	if task.PortablePath(name) != nil {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	n := path.Join(prefix, name)
	if s.out != "" && beneath(n, s.out) {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	if v, ok := s.files[n]; ok && !s.noCache {
		if len(v.raw) > limit {
			return nil, errors.New("PROJECT_RESOURCE_LIMIT")
		}
		if err := s.chargeProjectFile(n, len(v.raw)); err != nil {
			return nil, err
		}
		return v.raw, nil
	}
	for other := range s.files {
		if strings.EqualFold(other, n) && other != n {
			return nil, errors.New("PROJECT_INPUT_INVALID")
		}
	}
	fs, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, errors.New("PROJECT_INPUT_UNAVAILABLE")
	}
	defer fs.Close()
	info, err := fs.Lstat(n)
	if err != nil {
		return nil, errors.New("PROJECT_INPUT_UNAVAILABLE")
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	if info.Size() > int64(min(limit, s.rawLeft)) {
		return nil, errors.New("PROJECT_RESOURCE_LIMIT")
	}
	b, err := task.ReadFile(s.root, n, min(limit, s.rawLeft))
	if err != nil {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	if logging.ContainsSensitive(string(b)) {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	if err := s.chargeProjectFile(n, len(b)); err != nil {
		return nil, err
	}
	s.rawLeft -= len(b)
	s.reads++
	s.files[n] = frozenInput{b, info}
	return b, nil
}
func (s *projectSource) openBundle(raw []byte) (artifact *bundle.Artifact, failure error) {
	defer func() {
		if failure != nil {
			s.failure = failure
		}
	}()
	if s.ctx.Err() != nil {
		return nil, s.ctx.Err()
	}
	if len(raw) == 0 {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	if b := s.bundles[&raw[0]]; b != nil && !s.noCache {
		if err := s.chargeProjectBundle(raw, b); err != nil {
			return nil, err
		}
		return b, nil
	}
	b, err := bundle.OpenBytes(raw)
	if err != nil {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	for _, v := range b.Files {
		if len(v) > s.extractedLeft {
			return nil, errors.New("PROJECT_RESOURCE_LIMIT")
		}
		s.extractedLeft -= len(v)
		if logging.ContainsSensitive(string(v)) {
			return nil, errors.New("PROJECT_INPUT_INVALID")
		}
	}
	s.decodes++
	if err := s.chargeProjectBundle(raw, b); err != nil {
		return nil, err
	}
	s.bundles[&raw[0]] = b
	return b, nil
}

func (s *projectSource) chargeProjectFile(n string, size int) error {
	if s.projectFiles == nil || s.projectFiles[n] {
		return nil
	}
	if size > s.projectRaw {
		return errors.New("PROJECT_RESOURCE_LIMIT")
	}
	s.projectRaw -= size
	s.projectFiles[n] = true
	return nil
}
func (s *projectSource) chargeProjectBundle(raw []byte, b *bundle.Artifact) error {
	if s.projectBundles == nil || s.projectBundles[&raw[0]] {
		return nil
	}
	for _, v := range b.Files {
		if len(v) > s.projectExtracted {
			return errors.New("PROJECT_RESOURCE_LIMIT")
		}
		s.projectExtracted -= len(v)
	}
	s.projectBundles[&raw[0]] = true
	return nil
}
func (s *projectSource) separate(prefix, a, b string) bool {
	a, b = path.Join(prefix, a), path.Join(prefix, b)
	if strings.EqualFold(a, b) {
		return false
	}
	x, xok := s.files[a]
	y, yok := s.files[b]
	return xok && yok && !os.SameFile(x.info, y.info)
}

type preparedProject struct {
	actualTasks   []string
	actualBundles []string
	manifest      ProjectManifest
	suite         *PreparedSuite
	cases         *preparedCases
	expect        *SuiteExpectation
	catalog       *frozenCatalog
	worlds        *frozenWorlds
	check         *ProjectCheck
}

func checkSkeleton() *ProjectCheck {
	r := &ProjectCheck{Format: "statetwin.dev/project-check/v1alpha1", Status: "invalid", Stages: []ProjectStage{}, Provenance: "not-proven"}
	for _, n := range []string{"manifest", "references", "task_coverage", "world_coverage", "static_admission"} {
		r.Stages = append(r.Stages, ProjectStage{Stage: n, Status: "not_checked"})
	}
	return r
}
func prepareProject(s *projectSource, prefix, name string) (*preparedProject, *ProjectCheck, error) {
	s.projectRaw, s.projectExtracted = projectInputLimit, projectInputLimit
	s.projectFiles = map[string]bool{}
	s.projectBundles = map[*byte]bool{}
	r := checkSkeleton()
	stage := 0
	fail := func(err error) (*preparedProject, *ProjectCheck, error) {
		if s.failure != nil {
			err = s.failure
		}
		if s.ctx.Err() != nil {
			err = s.ctx.Err()
		}
		code := projectError(err)
		r.Stages[stage].Status = "failed"
		r.Stages[stage].ReasonCode = code.Error()
		return nil, r, code
	}
	if s.ctx.Err() != nil {
		return fail(s.ctx.Err())
	}
	read := func(n string, limit int) ([]byte, error) { return s.read(prefix, n, limit) }
	raw, err := read(name, 64<<10)
	if err != nil {
		return fail(err)
	}
	var m ProjectManifest
	if decodeSuiteMetadata(raw, 64<<10, &m) != nil || m.Format != ProjectFormat || m.Profile != ProjectProfile || !validLabel(m.ID) || !oneOf(m.Policy, "candidate-pass-v1", "both-pass-v1") {
		return fail(errors.New("PROJECT_INPUT_INVALID"))
	}
	r.ProjectID = m.ID
	r.Stages[stage].Status = "passed"
	stage++
	p := &preparedProject{manifest: m, check: r}
	raw, err = read(m.Suite, MaxSuitePlanBytes)
	if err != nil {
		return fail(err)
	}
	sp, err := DecodeSuite(raw)
	if err != nil {
		return fail(err)
	}
	p.suite, err = prepareSuiteWith(s.ctx, raw, maxSuiteInputBytes, maxSuiteInputBytes, read, s.openBundle)
	if err != nil {
		return fail(err)
	}
	raw, err = read(m.Expectation, MaxSuitePlanBytes)
	if err != nil {
		return fail(err)
	}
	p.expect, err = DecodeExpectation(raw)
	if err != nil {
		return fail(err)
	}
	// Output separation is enforced in the rooted reader even for campaign subroots.
	p.catalog, err = loadCatalogWith(s.ctx, m.TaskCatalog, "reserved-project-output", read)
	if err != nil {
		return fail(err)
	}
	p.cases, err = prepareCasesWith(s.ctx, m.Cases, read, s.openBundle)
	if err != nil {
		return fail(err)
	}
	r.PlannedCases = len(p.cases.inputs)
	r.PlannedPairs = len(p.suite.plan.Pairs)
	r.PlannedTrials = 2 * r.PlannedPairs
	r.Stages[stage].Status = "passed"
	stage++
	review, err := reviewPrepared(s.ctx, p.expect, p.catalog, p.suite, sp.MaxOutputTokens)
	if err != nil {
		return fail(err)
	}
	if review.Decision != "matched" {
		return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
	}
	if len(p.cases.tasks) != len(p.catalog.tasks) {
		return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
	}
	for _, pair := range sp.Pairs {
		p.actualTasks = append(p.actualTasks, pair.Task)
		if !same(p.cases.tasks[pair.TaskID], p.catalog.tasks[pair.TaskID]) {
			return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
		}
		for _, ref := range p.catalog.entries {
			if !s.separate(prefix, ref.Task, pair.Task) {
				return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
			}
			for _, c := range p.cases.manifest.Tasks {
				if !s.separate(prefix, ref.Task, c.Task) {
					return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
				}
			}
		}
	}
	for _, ref := range p.cases.manifest.Tasks {
		p.actualTasks = append(p.actualTasks, ref.Task)
	}
	for _, t := range p.cases.tasks {
		p.actualBundles = append(p.actualBundles, t.Bundle)
	}
	r.Stages[stage].Status = "passed"
	stage++
	p.worlds, err = loadWorlds(s, prefix, m.WorldCatalog, p)
	if err != nil {
		return fail(err)
	}
	r.Stages[stage].Status = "passed"
	stage++
	if s.ctx.Err() != nil {
		return fail(s.ctx.Err())
	}
	r.Stages[stage].Status = "passed"
	r.Status = "statically_valid"
	return p, r, nil
}
func CheckProject(parent context.Context, root, name string) (*ProjectCheck, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	_, r, err := prepareProject(newProjectSource(ctx, root, "", projectInputLimit), ".", name)
	return r, err
}
func projectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return errors.New("PROJECT_CANCELED_OR_TIMED_OUT")
	}
	s := err.Error()
	if strings.HasPrefix(s, "PROJECT_") {
		return err
	}
	if strings.Contains(s, "RESOURCE_LIMIT") {
		return errors.New("PROJECT_RESOURCE_LIMIT")
	}
	return errors.New("PROJECT_INPUT_INVALID")
}
