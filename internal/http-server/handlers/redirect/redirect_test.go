package redirect

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type mockURLGetter struct {
	getURLFunc func(alias string) (string, error)
}

func (m *mockURLGetter) GetURL(alias string) (string, error) {
	return m.getURLFunc(alias)
}

func TestRedirectHandler(t *testing.T) {
	cases := []struct {
		name       string
		alias      string
		mockError  error
		mockURL    string
		respError  string
		wantStatus int
	}{
		{
			name:       "Success",
			alias:      "home",
			mockURL:    "https://example.com",
			wantStatus: http.StatusFound,
		},
		{
			name:       "Empty alias",
			alias:      "",
			respError:  "not found alias",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "URL not found",
			alias:      "unknown",
			mockError:  storage.ErrURLNotFound,
			respError:  "not found url",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Database error",
			alias:      "home",
			mockError:  errors.New("database connection failed"),
			respError:  "internal error",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := &mockURLGetter{
				getURLFunc: func(alias string) (string, error) {
					if tc.mockError != nil {
						return "", tc.mockError
					}
					return tc.mockURL, nil
				},
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			handler := New(logger, mock)

			router := chi.NewRouter()
			router.Get("/{alias}", handler)

			req, err := http.NewRequest(http.MethodGet, "/"+tc.alias, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)

			if tc.respError != "" && tc.wantStatus == http.StatusOK {
				var resp struct {
					Error string `json:"error"`
				}
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
				require.Equal(t, tc.respError, resp.Error)
			}

			if tc.wantStatus == http.StatusFound {
				require.Equal(t, tc.mockURL, rr.Header().Get("Location"))
			}
		})
	}
}

// TODO: поправить!
