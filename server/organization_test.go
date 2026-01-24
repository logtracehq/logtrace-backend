package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gitlab.com/logbase/logbase"
	logbase_mocks "gitlab.com/logbase/logbase/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var orgID = uuid.MustParse("8ce0f580-4d6d-429e-9d0e-a78eb99f62c2")

func getLogger(t *testing.T) *zap.Logger {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	return logger
}

func TestOrganizationContextOperations(t *testing.T) {
	t.Run("write and read organization from context", func(t *testing.T) {
		ctx := context.Background()
		org := &logbase.Organization{
			ID:                   orgID,
			Name:                 "Test Organization",
			IsActive:             true,
			PlanName:             "free",
			IsSubscriptionActive: true,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}

		ctx = writeOrganizationToCtx(ctx, org)

		require.True(t, doesOrganizationExistInContext(ctx))

		retrievedOrg := getOrganizationFromContext(ctx)
		require.NotNil(t, retrievedOrg)
		require.Equal(t, org.ID, retrievedOrg.ID)
		require.Equal(t, org.Name, retrievedOrg.Name)
		require.Equal(t, org.IsActive, retrievedOrg.IsActive)
		require.Equal(t, org.PlanName, retrievedOrg.PlanName)
		require.Equal(t, org.IsSubscriptionActive, retrievedOrg.IsSubscriptionActive)
	})

	t.Run("organization does not exist in empty context", func(t *testing.T) {
		ctx := context.Background()
		require.False(t, doesOrganizationExistInContext(ctx))
	})

	t.Run("organization context with nil organization", func(t *testing.T) {
		ctx := context.Background()
		ctx = writeOrganizationToCtx(ctx, nil)

		// When nil is written, it should still be retrievable
		require.True(t, doesOrganizationExistInContext(ctx))
	})
}

func TestOrganizationSubscriptionMiddleware(t *testing.T) {
	cfg := getConfig()

	t.Run("blocks request when subscription is not active", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		org := &logbase.Organization{
			ID:                   orgID,
			Name:                 "Test Org",
			IsActive:             true,
			IsSubscriptionActive: false,
		}

		req = req.WithContext(writeOrganizationToCtx(req.Context(), org))

		handler := requireOrganizationValidSubscription(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusPaymentRequired, rr.Code)
	})

	t.Run("allows request when subscription is active", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		org := &logbase.Organization{
			ID:                   orgID,
			Name:                 "Test Org",
			IsActive:             true,
			IsSubscriptionActive: true,
		}

		req = req.WithContext(writeOrganizationToCtx(req.Context(), org))

		handler := requireOrganizationValidSubscription(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("organization with inactive status but active subscription", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		org := &logbase.Organization{
			ID:                   orgID,
			Name:                 "Test Org",
			IsActive:             false,
			IsSubscriptionActive: true,
		}

		req = req.WithContext(writeOrganizationToCtx(req.Context(), org))

		handler := requireOrganizationValidSubscription(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestOrganizationModel(t *testing.T) {
	t.Run("organization with all fields", func(t *testing.T) {
		now := time.Now()
		org := logbase.Organization{
			ID:                   orgID,
			Name:                 "My Organization",
			IsActive:             true,
			PlanName:             "pro",
			CreatedAt:            now,
			IsSubscriptionActive: true,
			UpdatedAt:            now,
			DeletedAt:            nil,
		}

		require.Equal(t, orgID, org.ID)
		require.Equal(t, "My Organization", org.Name)
		require.True(t, org.IsActive)
		require.Equal(t, "pro", org.PlanName)
		require.True(t, org.IsSubscriptionActive)
		require.Nil(t, org.DeletedAt)
	})

	t.Run("organization with deleted timestamp", func(t *testing.T) {
		now := time.Now()
		deletedAt := now.Add(-time.Hour)
		org := logbase.Organization{
			ID:                   orgID,
			Name:                 "Deleted Organization",
			IsActive:             false,
			PlanName:             "free",
			CreatedAt:            now.Add(-24 * time.Hour),
			IsSubscriptionActive: false,
			UpdatedAt:            now,
			DeletedAt:            &deletedAt,
		}

		require.NotNil(t, org.DeletedAt)
		require.Equal(t, deletedAt, *org.DeletedAt)
		require.False(t, org.IsActive)
	})
}

func TestFindOrganizationOptions(t *testing.T) {
	t.Run("find by ID", func(t *testing.T) {
		opts := logbase.FindOrganizationOptions{
			ID: orgID,
		}

		require.Equal(t, orgID, opts.ID)
		require.Empty(t, opts.Name)
	})

	t.Run("find by name", func(t *testing.T) {
		opts := logbase.FindOrganizationOptions{
			Name: "Test Organization",
		}

		require.Equal(t, uuid.Nil, opts.ID)
		require.Equal(t, "Test Organization", opts.Name)
	})

	t.Run("find by both ID and name", func(t *testing.T) {
		opts := logbase.FindOrganizationOptions{
			ID:   orgID,
			Name: "Test Organization",
		}

		require.Equal(t, orgID, opts.ID)
		require.Equal(t, "Test Organization", opts.Name)
	})
}

func TestListOrganizationOptions(t *testing.T) {
	t.Run("with pagination", func(t *testing.T) {
		opts := logbase.ListOrganizationOptions{
			Paginator: logbase.Paginator{
				PerPage: 10,
				Page:    1,
			},
		}

		require.Equal(t, int64(10), opts.Paginator.PerPage)
		require.Equal(t, int64(1), opts.Paginator.Page)
	})

	t.Run("default pagination", func(t *testing.T) {
		opts := logbase.ListOrganizationOptions{}

		require.Equal(t, int64(0), opts.Paginator.PerPage)
		require.Equal(t, int64(0), opts.Paginator.Page)
	})
}

func TestOrganizationRepository(t *testing.T) {
	t.Run("create organization successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		newOrg := &logbase.Organization{
			Name:                 "New Organization",
			IsActive:             true,
			PlanName:             "free",
			IsSubscriptionActive: true,
		}

		expectedOrg := &logbase.Organization{
			ID:                   orgID,
			Name:                 "New Organization",
			IsActive:             true,
			PlanName:             "free",
			IsSubscriptionActive: true,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}

		orgRepo.EXPECT().
			Create(gomock.Any(), newOrg).
			Times(1).
			Return(expectedOrg, nil)

		result, err := orgRepo.Create(context.Background(), newOrg)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, orgID, result.ID)
		require.Equal(t, "New Organization", result.Name)
	})

	t.Run("create organization fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		newOrg := &logbase.Organization{
			Name:     "New Organization",
			IsActive: true,
			PlanName: "free",
		}

		orgRepo.EXPECT().
			Create(gomock.Any(), newOrg).
			Times(1).
			Return(nil, errors.New("database error"))

		result, err := orgRepo.Create(context.Background(), newOrg)
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, "database error", err.Error())
	})

	t.Run("get organization by ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		opts := logbase.FindOrganizationOptions{ID: orgID}
		expectedOrg := &logbase.Organization{
			ID:                   orgID,
			Name:                 "Test Organization",
			IsActive:             true,
			PlanName:             "pro",
			IsSubscriptionActive: true,
		}

		orgRepo.EXPECT().
			List(gomock.Any(), opts).
			Times(1).
			Return(expectedOrg, nil)

		result, err := orgRepo.List(context.Background(), opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, orgID, result.ID)
		require.Equal(t, "Test Organization", result.Name)
	})

	t.Run("get organization not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		opts := logbase.FindOrganizationOptions{ID: uuid.New()}

		orgRepo.EXPECT().
			List(gomock.Any(), opts).
			Times(1).
			Return(nil, logbase.OrganizationNotFound)

		result, err := orgRepo.List(context.Background(), opts)
		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, logbase.OrganizationNotFound)
	})

	t.Run("list organizations for user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)
		userID := uuid.New()
		user := &logbase.User{ID: userID}

		expectedOrgs := []logbase.Organization{
			{
				ID:                   orgID,
				Name:                 "Organization 1",
				IsActive:             true,
				PlanName:             "free",
				IsSubscriptionActive: true,
			},
			{
				ID:                   uuid.New(),
				Name:                 "Organization 2",
				IsActive:             true,
				PlanName:             "pro",
				IsSubscriptionActive: true,
			},
		}

		orgRepo.EXPECT().
			List(gomock.Any(), user).
			Times(1).
			Return(expectedOrgs, nil)

		result, err := orgRepo.ListAll(context.Background(), user)
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "Organization 1", result[0].Name)
		require.Equal(t, "Organization 2", result[1].Name)
	})

	t.Run("list organizations returns empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)
		userID := uuid.New()
		user := &logbase.User{ID: userID}

		orgRepo.EXPECT().
			ListAll(gomock.Any(), user).
			Times(1).
			Return([]logbase.Organization{}, nil)

		result, err := orgRepo.ListAll(context.Background(), user)
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("delete organization successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		opts := &logbase.FindOrganizationOptions{ID: orgID}

		orgRepo.EXPECT().
			Delete(gomock.Any(), opts).
			Times(1).
			Return(nil)

		err := orgRepo.Delete(context.Background(), opts)
		require.NoError(t, err)
	})

	t.Run("delete organization not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		opts := &logbase.FindOrganizationOptions{ID: uuid.New()}

		orgRepo.EXPECT().
			Delete(gomock.Any(), opts).
			Times(1).
			Return(logbase.OrganizationNotFound)

		err := orgRepo.Delete(context.Background(), opts)
		require.Error(t, err)
		require.ErrorIs(t, err, logbase.OrganizationNotFound)
	})
}
