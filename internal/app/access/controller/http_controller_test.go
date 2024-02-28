package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/raw-leak/configleam/internal/app/access/controller"
	"github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/app/access/repository"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockAccessService struct {
	mock.Mock
}

func (m *MockAccessService) GenerateAccessKey(ctx context.Context, perms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error) {
	args := m.Called(ctx, perms)
	return args.Get(0).(dto.AccessKeyPermissionsDto), args.Error(1)
}

func (m *MockAccessService) DeleteAccessKeys(ctx context.Context, keys []string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockAccessService) PaginateAccessKeys(ctx context.Context, page, size int) (*repository.PaginatedAccessKeys, error) {
	args := m.Called(ctx, page, size)
	return args.Get(0).(*repository.PaginatedAccessKeys), args.Error(1)
}

type EndpointSuite struct {
	suite.Suite
	service   *MockAccessService
	endpoints *controller.AccessEndpoints
}

func (suite *EndpointSuite) SetupTest() {
	suite.service = new(MockAccessService)
	suite.endpoints = controller.New(suite.service)
}

func TestEndpointSuite(t *testing.T) {
	suite.Run(t, new(EndpointSuite))
}

func (suite *EndpointSuite) TestGenerateAccessKeyHandler() {
	// Define test cases
	testCases := []struct {
		name                 string
		dto                  dto.AccessKeyPermissionsDto
		invalidDto           string
		expectedBody         dto.AccessKeyPermissionsDto
		expectedErr          error
		expectedStatus       int
		expectedErrorMessage string
	}{
		{
			name: "When request contains a valid body",
			dto: dto.AccessKeyPermissionsDto{
				Envs: map[string]dto.EnvironmentPermissions{
					"dev": {
						ReadConfig:       true,
						RevealSecrets:    false,
						CloneEnvironment: false,
						CreateSecrets:    true,
						AccessDashboard:  true,
					},
				},
			},
			expectedBody: dto.AccessKeyPermissionsDto{
				Envs: map[string]dto.EnvironmentPermissions{
					"dev": {
						ReadConfig:       true,
						RevealSecrets:    false,
						CloneEnvironment: false,
						CreateSecrets:    true,
						AccessDashboard:  true,
					},
				},
				AccessKey: "key",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "When GenerateAccessKey returns an error",
			dto: dto.AccessKeyPermissionsDto{
				Envs: map[string]dto.EnvironmentPermissions{
					"dev": {
						ReadConfig:       true,
						RevealSecrets:    false,
						CloneEnvironment: false,
						CreateSecrets:    true,
						AccessDashboard:  true,
					},
				},
			},
			expectedBody:         dto.AccessKeyPermissionsDto{},
			expectedErr:          errors.New("some error"),
			expectedErrorMessage: "Error generating access-key\n",
			expectedStatus:       http.StatusInternalServerError,
		},
		{
			name:                 "When request contains a valid body",
			invalidDto:           "{ some invalid body",
			expectedErrorMessage: "Error decoding request body\n",
			expectedStatus:       http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange

			var requestBody []byte
			var err error

			if tc.invalidDto != "" {
				requestBody = []byte(tc.invalidDto)
			} else {
				requestBody, err = json.Marshal(tc.dto)
				suite.NoError(err)
			}

			req, err := http.NewRequest("POST", "/", strings.NewReader(string(requestBody)))
			suite.NoError(err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(suite.endpoints.GenerateAccessKeyHandler)

			suite.service.On("GenerateAccessKey", req.Context(), tc.dto).Once().Return(tc.expectedBody, tc.expectedErr)

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			suite.Equal(tc.expectedStatus, rr.Code)

			if rr.Code == http.StatusOK {
				var actualBody dto.AccessKeyPermissionsDto
				err = json.NewDecoder(rr.Body).Decode(&actualBody)
				suite.NoError(err)

				suite.Equal(tc.expectedBody, actualBody)
			} else {
				actualBodyMessage := rr.Body.String()
				suite.Equal(tc.expectedErrorMessage, actualBodyMessage)
			}

		})
	}
}

func (suite *EndpointSuite) TestDeleteAccessKeysHandler() {
	// Define test cases
	testCases := []struct {
		name                 string
		keys                 []string
		expectedErr          error
		expectedStatus       int
		expectedResponseBody string
	}{
		{
			name:                 "Successful Deletion",
			keys:                 []string{"key1", "key2"},
			expectedErr:          nil,
			expectedStatus:       http.StatusOK,
			expectedResponseBody: `{"message":"Access-keys deleted successfully"}`,
		},
		{
			name:                 "Service Returns Error",
			keys:                 []string{"invalidKey"},
			expectedErr:          errors.New("deletion error"),
			expectedStatus:       http.StatusInternalServerError,
			expectedResponseBody: `Error deleting access-keys`,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			queryString := "?key=" + strings.Join(tc.keys, "&key=")
			req, err := http.NewRequest("DELETE", "/delete"+queryString, nil)
			suite.NoError(err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(suite.endpoints.DeleteAccessKeysHandler)

			suite.service.On("DeleteAccessKeys", mock.Anything, tc.keys).Once().Return(tc.expectedErr)

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			suite.Equal(tc.expectedStatus, rr.Code)

			if rr.Code == http.StatusOK {
				var actualResponseBody map[string]string
				err = json.NewDecoder(rr.Body).Decode(&actualResponseBody)
				suite.NoError(err)
				expectedResponseBody := map[string]string{}
				err = json.Unmarshal([]byte(tc.expectedResponseBody), &expectedResponseBody)
				suite.NoError(err)

				suite.Equal(expectedResponseBody, actualResponseBody)
			} else {
				actualErrorMessage := strings.TrimSpace(rr.Body.String())
				suite.Equal(tc.expectedResponseBody, actualErrorMessage)
			}
		})
	}
}

func (suite *EndpointSuite) TestPaginateAccessKeysHandler() {
	// Define test cases
	testCases := []struct {
		name                 string
		page                 int
		size                 int
		mockResponse         *repository.PaginatedAccessKeys
		expectedErr          error
		expectedStatus       int
		expectedResponseBody string
	}{
		{
			name:        "Successful pagination with empty response",
			page:        1,
			size:        10,
			expectedErr: nil,
			mockResponse: &repository.PaginatedAccessKeys{
				Total: 0,
				Pages: 0,
				Page:  1,
				Size:  10,
				Items: []repository.AccessKeyMetadata{},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:                 "Service Returns Error",
			page:                 1,
			size:                 10,
			mockResponse:         nil,
			expectedErr:          errors.New("paginating error"),
			expectedStatus:       http.StatusInternalServerError,
			expectedResponseBody: `Error paginating access-keys`,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			queryString := "?size=" + strconv.Itoa(tc.size) + "&page=" + strconv.Itoa(tc.page)
			req, err := http.NewRequest("GET", "/"+queryString, nil)
			suite.NoError(err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(suite.endpoints.PaginateAccessKeysHandler)

			suite.service.On("PaginateAccessKeys", mock.Anything, tc.page, tc.size).Once().Return(tc.mockResponse, tc.expectedErr)

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			suite.Equal(tc.expectedStatus, rr.Code)

			if rr.Code == http.StatusOK {
				var actualResponseBody repository.PaginatedAccessKeys
				err = json.NewDecoder(rr.Body).Decode(&actualResponseBody)
				suite.NoError(err)

				suite.Equal(*tc.mockResponse, actualResponseBody)
			} else {
				actualErrorMessage := strings.TrimSpace(rr.Body.String())
				suite.Equal(tc.expectedResponseBody, actualErrorMessage)
			}
		})
	}
}
