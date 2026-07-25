package workflowinstallation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/artifact"
)

var (
	ErrInstallationNotFound = errors.New("Workflow Installation not found")
	ErrInstallationConflict = errors.New("Workflow Installation conflict")
	ErrReleaseConflict      = errors.New("Workflow Release identity collision")
)

type Repository interface {
	Commit(context.Context, ReleaseRecord, InstallationRecord) error
	GetInstallation(context.Context, string) (InstallationRecord, bool, error)
	ListInstallations(context.Context) ([]InstallationRecord, error)
	GetRelease(context.Context, artifact.Digest) (ReleaseRecord, bool, error)
}

type Options struct {
	Now   func() time.Time
	NewID func() string
}

// Module is the command/query boundary for Workflow Installations. Verified
// release persistence and installation creation are one repository commit.
type Module struct {
	repository Repository
	now        func() time.Time
	newID      func() string
}

func New(repository Repository, options Options) (*Module, error) {
	if repository == nil {
		return nil, errors.New("Workflow Installation module requires a repository")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewID == nil {
		options.NewID = uuid.NewString
	}
	return &Module{repository: repository, now: options.Now, newID: options.NewID}, nil
}

func (m *Module) InstallVerified(
	ctx context.Context,
	release ReleaseRecord,
	name string,
) (InstallationRecord, error) {
	if ctx == nil {
		return InstallationRecord{}, errors.New("install verified Workflow Release requires a context")
	}
	if err := ctx.Err(); err != nil {
		return InstallationRecord{}, err
	}
	if err := ValidateReleaseRecord(release); err != nil {
		return InstallationRecord{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = release.WorkflowName
	}
	now := m.now().UTC()
	installation := InstallationRecord{
		ID: m.newID(), ReleaseID: release.ID, Name: name,
		Lifecycle: LifecycleActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := ValidateInstallationRecord(installation); err != nil {
		return InstallationRecord{}, err
	}
	if err := m.repository.Commit(ctx, release, installation); err != nil {
		return InstallationRecord{}, err
	}
	return installation, nil
}

func (m *Module) Get(ctx context.Context, installationID string) (InstallationRecord, error) {
	if ctx == nil || !installationIDPattern.MatchString(installationID) {
		return InstallationRecord{}, ErrInstallationNotFound
	}
	installation, found, err := m.repository.GetInstallation(ctx, installationID)
	if err != nil {
		return InstallationRecord{}, err
	}
	if !found {
		return InstallationRecord{}, ErrInstallationNotFound
	}
	return installation, nil
}

func (m *Module) List(ctx context.Context) ([]InstallationRecord, error) {
	if ctx == nil {
		return nil, errors.New("list Workflow Installations requires a context")
	}
	return m.repository.ListInstallations(ctx)
}

func (m *Module) Readiness(
	ctx context.Context,
	installationID string,
	environment ReadinessEnvironment,
) (ReadinessReport, error) {
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return ReadinessReport{}, err
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return ReadinessReport{}, err
	}
	if !found {
		return ReadinessReport{}, errors.New("Workflow Installation references a missing Release")
	}
	return EvaluateReadiness(installation, release, environment)
}
