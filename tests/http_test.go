package tests

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/wyw14/cry045/internal/application"
	"github.com/wyw14/cry045/internal/repository"
	httptransport "github.com/wyw14/cry045/internal/transport/http"
)

func TestHealthAndRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := repository.NewDemoStore(time.Now())
	r := httptransport.NewServer(application.NewComplianceService(store, application.RealClock{}), nil).Engine
	req := httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Request-ID", "test-request")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "test-request", w.Header().Get("X-Request-ID"))
}
