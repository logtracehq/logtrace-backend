package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	"gitlab.com/logbase/logbase"
	logbase_mocks "gitlab.com/logbase/logbase/mocks"
	"go.uber.org/mock/gomock"
)

var (
	testOrgID      = uuid.MustParse("8ce0f580-4d6d-429e-9d0e-a78eb99f62c2")
	testUserID     = uuid.MustParse("37f41afb-afff-45cc-bcc0-71249d95df90")
	testResourceID = uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	testAuditLogID = uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-f12345678901")
)

func verifyAuditLogMatch(t *testing.T, rr *httptest.ResponseRecorder) {
	g := goldie.New(t, goldie.WithFixtureDir("./testdata"))
	g.Assert(t, t.Name(), rr.Body.Bytes())
}

func TestAuditLogHandler_Create(t *testing.T) {
	for _, v := range generateCreateAuditLogTestTable() {
		t.Run(v.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			auditLogRepo := logbase_mocks.NewMockAuditLogRepository(controller)

			v.mockFn(auditLogRepo)

			a := &auditLogHandler{
				auditLogRepo: auditLogRepo,
			}

			b := bytes.NewBuffer(nil)
			require.NoError(t, json.NewEncoder(b).Encode(&v.req))

			rr := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodPost, "/", b)
			ctx := chi.NewRouteContext()
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
			req.Header.Add("Content-Type", "application/json")

			req = req.WithContext(writeUserToCtx(req.Context(), &logbase.User{ID: testUserID}))
			req = req.WithContext(writeOrganizationToCtx(req.Context(), &logbase.Organization{ID: testOrgID}))

			WrapLogbaseHTTPHandler(getLogger(t), a.Create, getConfig(), "AuditLog.Create").
				ServeHTTP(rr, req)

			require.Equal(t, v.expectedStatusCode, rr.Code)
			verifyAuditLogMatch(t, rr)
		})
	}
}

func generateCreateAuditLogTestTable() []struct {
	name               string
	mockFn             func(auditLogRepo *logbase_mocks.MockAuditLogRepository)
	expectedStatusCode int
	req                createAuditLogRequest
} {
	return []struct {
		name               string
		mockFn             func(auditLogRepo *logbase_mocks.MockAuditLogRepository)
		expectedStatusCode int
		req                createAuditLogRequest
	}{
		{
			name: "missing action",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
			},
			expectedStatusCode: http.StatusBadRequest,
			req: createAuditLogRequest{
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				ResourceID: testResourceID.String(),
				UserID:     testUserID.String(),
			},
		},
		{
			name: "missing timestamp",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
			},
			expectedStatusCode: http.StatusBadRequest,
			req: createAuditLogRequest{
				Action:     "user.login",
				ResourceID: testResourceID.String(),
				UserID:     testUserID.String(),
			},
		},
		{
			name: "missing resource id",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
			},
			expectedStatusCode: http.StatusBadRequest,
			req: createAuditLogRequest{
				Action:    "user.login",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				UserID:    testUserID.String(),
			},
		},
		{
			name: "missing user id",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
			},
			expectedStatusCode: http.StatusBadRequest,
			req: createAuditLogRequest{
				Action:     "user.login",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				ResourceID: testResourceID.String(),
			},
		},
		{
			name: "invalid resource id",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
			},
			expectedStatusCode: http.StatusBadRequest,
			req: createAuditLogRequest{
				Action:     "user.login",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				ResourceID: "invalid-uuid",
				UserID:     testUserID.String(),
			},
		},
		{
			name: "failed to create audit log",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return(errors.New("database error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			req: createAuditLogRequest{
				Action:     "user.login",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				ResourceID: testResourceID.String(),
				UserID:     testUserID.String(),
				IPAddress:  "192.168.1.1",
				RequestID:  "req_123",
				Metadata: &logbase.Metadata{
					Event:       "login",
					Type:        "authentication",
					Description: "User logged in",
				},
			},
		},
		{
			name: "successfully created audit log",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedStatusCode: http.StatusCreated,
			req: createAuditLogRequest{
				Action:     "user.login",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				ResourceID: testResourceID.String(),
				UserID:     testUserID.String(),
				IPAddress:  "192.168.1.1",
				RequestID:  "req_123",
				Metadata: &logbase.Metadata{
					Event:       "login",
					Type:        "authentication",
					Description: "User logged in",
				},
			},
		},
	}
}

func TestAuditLogHandler_List(t *testing.T) {
	for _, v := range generateListAuditLogTestTable() {
		t.Run(v.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			auditLogRepo := logbase_mocks.NewMockAuditLogRepository(controller)

			v.mockFn(auditLogRepo)

			a := &auditLogHandler{
				auditLogRepo: auditLogRepo,
			}

			rr := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodGet, "/"+v.reference, nil)
			ctx := chi.NewRouteContext()
			ctx.URLParams.Add("reference", v.reference)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

			req = req.WithContext(writeUserToCtx(req.Context(), &logbase.User{ID: testUserID}))
			req = req.WithContext(writeOrganizationToCtx(req.Context(), &logbase.Organization{ID: testOrgID}))

			WrapLogbaseHTTPHandler(getLogger(t), a.List, getConfig(), "AuditLog.List").
				ServeHTTP(rr, req)

			require.Equal(t, v.expectedStatusCode, rr.Code)
			verifyAuditLogMatch(t, rr)
		})
	}
}

func generateListAuditLogTestTable() []struct {
	name               string
	reference          string
	mockFn             func(auditLogRepo *logbase_mocks.MockAuditLogRepository)
	expectedStatusCode int
} {
	return []struct {
		name               string
		reference          string
		mockFn             func(auditLogRepo *logbase_mocks.MockAuditLogRepository)
		expectedStatusCode int
	}{
		{
			name:      "invalid audit log id",
			reference: "invalid-uuid",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:      "failed to fetch audit log",
			reference: testAuditLogID.String(),
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errors.New("database error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:      "audit log not found",
			reference: testAuditLogID.String(),
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, nil)
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:      "successfully fetched audit log",
			reference: testAuditLogID.String(),
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).
					Return(&logbase.AuditLog{
						ID:             testAuditLogID,
						Action:         "user.login",
						Timestamp:      time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
						ResourceID:     testResourceID,
						IPAddress:      "192.168.1.1",
						UserID:         testUserID,
						OrganizationID: testOrgID,
						RequestID:      "req_123",
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
	}
}

func TestAuditLogHandler_ListAll(t *testing.T) {
	for _, v := range generateListAllAuditLogsTestTable() {
		t.Run(v.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			auditLogRepo := logbase_mocks.NewMockAuditLogRepository(controller)

			v.mockFn(auditLogRepo)

			a := &auditLogHandler{
				auditLogRepo: auditLogRepo,
			}

			rr := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodGet, "/?page=1&per_page=10", nil)
			ctx := chi.NewRouteContext()
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

			req = req.WithContext(writeUserToCtx(req.Context(), &logbase.User{ID: testUserID}))
			req = req.WithContext(writeOrganizationToCtx(req.Context(), &logbase.Organization{ID: testOrgID}))

			WrapLogbaseHTTPHandler(getLogger(t), a.ListAll, getConfig(), "AuditLog.ListAll").
				ServeHTTP(rr, req)

			require.Equal(t, v.expectedStatusCode, rr.Code)
			verifyAuditLogMatch(t, rr)
		})
	}
}

func generateListAllAuditLogsTestTable() []struct {
	name               string
	mockFn             func(auditLogRepo *logbase_mocks.MockAuditLogRepository)
	expectedStatusCode int
} {
	return []struct {
		name               string
		mockFn             func(auditLogRepo *logbase_mocks.MockAuditLogRepository)
		expectedStatusCode int
	}{
		{
			name: "failed to list audit logs",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().ListAll(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, int64(0), errors.New("database error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "successfully listed audit logs with empty result",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().ListAll(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]*logbase.AuditLog{}, int64(0), nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "successfully listed audit logs",
			mockFn: func(auditLogRepo *logbase_mocks.MockAuditLogRepository) {
				auditLogRepo.EXPECT().ListAll(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]*logbase.AuditLog{
						{
							ID:             testAuditLogID,
							Action:         "user.login",
							Timestamp:      time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
							ResourceID:     testResourceID,
							IPAddress:      "192.168.1.1",
							UserID:         testUserID,
							OrganizationID: testOrgID,
							RequestID:      "req_123",
						},
					}, int64(1), nil)
			},
			expectedStatusCode: http.StatusOK,
		},
	}
}
