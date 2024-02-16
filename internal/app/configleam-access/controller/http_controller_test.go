package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam-access/controller"
	"github.com/raw-leak/configleam/internal/app/configleam-access/dto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockConfigleamAccessService struct {
	mock.Mock
}

func (m *MockConfigleamAccessService) GenerateAccessKey(ctx context.Context, perms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error) {
	args := m.Called(ctx, perms)
	return args.Get(0).(dto.AccessKeyPermissionsDto), args.Error(1)
}

func (m *MockConfigleamAccessService) DeleteAccessKeys(ctx context.Context, keys []string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

type EndpointSuite struct {
	suite.Suite
	service   *MockConfigleamAccessService
	endpoints *controller.ConfigleamAccessEndpoints
}

func (suite *EndpointSuite) SetupTest() {
	suite.service = new(MockConfigleamAccessService)
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
