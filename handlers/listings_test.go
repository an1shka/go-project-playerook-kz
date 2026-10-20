package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindListingValidatesCategoryAndPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "valid listing",
			body:       `{"title":"Account","game":"Game","category":"account","description":"Details","price_kzt":1000}`,
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unsupported category",
			body:       `{"title":"Account","game":"Game","category":"other","description":"Details","price_kzt":1000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-positive price",
			body:       `{"title":"Account","game":"Game","category":"account","description":"Details","price_kzt":0}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			_, ok := bindListing(context)
			if ok != test.wantOK {
				t.Fatalf("bindListing() ok = %v, want %v", ok, test.wantOK)
			}
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}
