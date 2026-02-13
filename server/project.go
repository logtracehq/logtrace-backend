package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type projectHandler struct {
	cfg         config.Config
	userRepo    logbase.UserRepository
	projectRepo logbase.ProjectRepository
}

type createProjectRequest struct {
	GenericRequest

	Name string
	Type string
}

func (r *createProjectRequest) Validate() error {
	if util.IsStringEmpty(r.Name) {
		return errors.New("project name is required")
	}
	if util.IsStringEmpty(r.Type) {
		return errors.New("project type is required")
	}

	return nil
}

// @Description Create a new project
// @Tags Project
// @Accept json
// @Produce json
// @Param project body createProjectRequest true "Project creation request"
// @Success 201 {object} APIStatus "Project created successfully"
// @Failure 400 {object} APIStatus "Invalid request body"
// @Failure 500 {object} APIStatus "Could not create project at this time. an error occurred"
// @Router /v1/projects [post]
func (res *projectHandler) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating new project")

	req := new(createProjectRequest)
	if err := render.Bind(r, req); err != nil {
		logger.Error("failed to bind create project request", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "failed to create project request"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		logger.Error("invalid create project request", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "failed to validate payload"), StatusFailed
	}

	project := &logbase.Project{
		Name:           req.Name,
		Type:           req.Type,
		OrganizationID: getOrganizationFromContext(ctx).ID,
	}

	if err := res.projectRepo.Create(ctx, project); err != nil {
		logger.Error("failed to create project", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to create project"), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Project created successfully"), StatusSuccess
}

// @Description Get project details
// @Tags Project
// @Accept json
// @Produce json
// @Param reference path string true "Project reference (UUID)"
// @Success 200 {object} fetchProject "Project details fetched successfully"
// @Failure 400 {object} APIStatus "Invalid project reference"
// @Failure 500 {object} APIStatus "Failed to fetch project details"
// @Router /v1/projects/{reference} [get]
func (res *projectHandler) List(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	projectID := chi.URLParam(r, "reference")
	projectIdentifier, err := uuid.Parse(projectID)
	if err != nil {
		logger.Error("invalid project id", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid project reference"), StatusFailed
	}

	project, err := res.projectRepo.List(ctx, projectIdentifier)
	if err != nil {
		logger.Error("failed to fetch project", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch project"), StatusFailed
	}

	return fetchProject{
		Project:   *project,
		APIStatus: newAPIStatus(http.StatusOK, "project fetched successfully"),
	}, StatusSuccess
}

// @Description Delete a project
// @Tags Project
// @Accept json
// @Produce json
// @Param reference path string true "Project reference (UUID)"
// @Success 200 {object} APIStatus "Project has been deleted successfully"
// @Failure 400 {object} APIStatus "Invalid project reference"
// @Failure 500 {object} APIStatus "Failed to delete project"
// @Router /v1/projects/{reference} [delete]
func (res *projectHandler) Delete(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	projectID := chi.URLParam(r, "reference")
	projectIdentifier, err := uuid.Parse(projectID)
	if err != nil {
		logger.Error("invalid project id", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid project id"), StatusFailed
	}

	if err := res.projectRepo.Delete(ctx, projectIdentifier); err != nil {
		logger.Error("failed to delete project", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to delete project"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "Project has been deleted successfully"), StatusSuccess
}

// @Description Get list of all projects in an organization
// @Tags Project
// @Accept json
// @Produce json
// @Param page query int false "Page number for pagination"
// @Param per_page query int false "Number of items per page for pagination"
// @Success 200 {object} fetchAllProjects "Projects fetched successfully"
// @Failure 400 {object} APIStatus "Invalid pagination parameters"
// @Failure 500 {object} APIStatus "Failed to fetch projects"
// @Router /v1/projects [get]
func (res *projectHandler) ListAll(ctx context.Context, span trace.Span, logger *zap.Logger, w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	opts := logbase.ListProjectOptions{
		Paginator: logbase.PaginatorFromRequest(r),
	}

	span.SetAttributes(opts.Paginator.OTELAttributes()...)

	projects, totalCount, err := res.projectRepo.ListAll(ctx, opts)
	if err != nil {
		logger.Error("failed to fetch all projects", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch all projects"), StatusFailed
	}

	projectsResponse := make([]*Project, 0, len(projects))
	for _, r := range projects {
		dto := &Project{
			ID:        r.ID,
			Name:      r.Name,
			Type:      r.Type,
			CreatedAt: r.CreatedAt,
		}

		projectsResponse = append(projectsResponse, dto)
	}

	return fetchAllProjects{
		Projects: projectsResponse,
		Meta: meta{
			Paging: pagingInfo{
				Total:   totalCount,
				Page:    int64(opts.Paginator.Page),
				PerPage: int64(opts.Paginator.PerPage),
			},
		},
		APIStatus: newAPIStatus(http.StatusOK, "Projects fetched successfully"),
	}, StatusSuccess
}
