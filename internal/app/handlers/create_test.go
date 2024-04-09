package handlers_test

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang/mock/gomock"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"

	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateShortenedURLHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		FindByOriginalURL(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(models.Record{}, storage.ErrNotFound)
	storageMock.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	generatorMock := new(mockRandHexStringGenerator)
	urlCreateService := services.NewCreateURLService(8, generatorMock, storageMock)
	userAuthenticator := new(userAuthenticatorMock)
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("text/plain", "application/x-gzip"),
	)
	router.Post("/", handler.CreateURL(urlCreateService, userAuthenticator))
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	type generatorCallResult struct {
		error       error
		returnValue string
	}
	testCases := []struct {
		name                string
		httpMethod          string
		path                string
		contentType         string
		generatorCallResult generatorCallResult
		authOrRegisterRes   authOrRegisterResult
		want                want
	}{
		{
			name:        "responses with created status",
			httpMethod:  http.MethodPost,
			path:        "/",
			contentType: "text/plain",
			authOrRegisterRes: authOrRegisterResult{
				user:   models.User{ID: 1},
				jwtStr: "123",
				err:    nil,
			},
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/123",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "responses with method not allowed if method is not POST",
			httpMethod:  http.MethodGet,
			path:        "/",
			contentType: "text/plain",
			authOrRegisterRes: authOrRegisterResult{
				user:   models.User{ID: 1},
				jwtStr: "123",
				err:    nil,
			},
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name:        `responses with bad request if content-type is not "text/plain"`,
			httpMethod:  http.MethodPost,
			path:        "/",
			contentType: "application/json",
			authOrRegisterRes: authOrRegisterResult{
				user:   models.User{ID: 1},
				jwtStr: "123",
				err:    nil,
			},
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusUnsupportedMediaType,
				response:    "",
				contentType: "",
			},
		},
		{
			name:        "responses with unprocessable entity status",
			httpMethod:  http.MethodPost,
			path:        "/",
			contentType: "text/plain",
			authOrRegisterRes: authOrRegisterResult{
				user:   models.User{ID: 1},
				jwtStr: "123",
				err:    nil,
			},
			generatorCallResult: generatorCallResult{returnValue: "", error: errors.New("error")},
			want: want{
				code:        http.StatusUnprocessableEntity,
				response:    "failed to generate shortened path: error\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			genCall := generatorMock.On("Call", mock.Anything).Return(
				tc.generatorCallResult.returnValue,
				tc.generatorCallResult.error,
			)
			authCall := userAuthenticator.On("AuthOrRegister", mock.Anything, mock.Anything).
				Return(tc.authOrRegisterRes.user, tc.authOrRegisterRes.jwtStr, tc.authOrRegisterRes.err)
			defer genCall.Unset()
			defer authCall.Unset()

			request, err := http.NewRequest(
				tc.httpMethod,
				testServer.URL+tc.path,
				strings.NewReader("http://example.com"),
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", tc.contentType)
			request.Header.Set("Accept-Encoding", "identity")

			response, err := testServer.Client().Do(request)
			require.NoError(t, err)
			resBody, err := io.ReadAll(response.Body)
			defer func() {
				err = response.Body.Close()
				require.NoError(t, err)
			}()

			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, tc.want.response, string(resBody))
			assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))
		})
	}
}
