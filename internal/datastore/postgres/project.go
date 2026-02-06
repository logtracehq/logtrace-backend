package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
)

type projectRepo struct {
	inner *bun.DB
}

func NewProjectRepo(db *bun.DB) logbase.ProjectRepository {
	return &projectRepo{
		inner: db,
	}
}

func (res *projectRepo) Create(ctx context.Context, project *logbase.Project) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := res.inner.NewInsert().
		Model(project).
		Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (res *projectRepo) List(ctx context.Context, id uuid.UUID) (*logbase.Project, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	project := &logbase.Project{}
	err := res.inner.NewSelect().
		Model(project).
		Where("id = ?", id).
		Scan(ctx)

	return project, err
}

func (res *projectRepo) ListAll(ctx context.Context, opts logbase.ListProjectOptions) ([]*logbase.Project, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	projects := []*logbase.Project{}
	count := int64(0)
	countTotal, err := res.inner.NewSelect().Model(&(projects)).Where("deleted_at IS NULL").Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	count = int64(countTotal)

	return projects, count, res.inner.NewSelect().
		Model(&projects).
		Where("deleted_at IS NULL").
		Limit(int(opts.Paginator.PerPage)).
		Offset(int(opts.Paginator.Offset())).Scan(ctx)
}

func (res *projectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := res.inner.NewDelete().
		Model(&logbase.Project{}).
		Where("id = ?", id).
		Exec(ctx)
	return err
}
