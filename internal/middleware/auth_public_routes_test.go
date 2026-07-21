package middleware

import (
	"net/http"
	"testing"
)

func TestActiveAnnouncementsArePublic(t *testing.T) {
	for _, path := range []string{
		"/api/v1/sdp/ops/announcements/active",
		"/api/v1/smartknora/ops/announcements/active",
	} {
		t.Run(path, func(t *testing.T) {
			if !isNoAuthAPI(path, http.MethodGet) {
				t.Fatalf("GET %s should be public", path)
			}
			if isNoAuthAPI(path, http.MethodPost) {
				t.Fatalf("POST %s should not be public", path)
			}
		})
	}
}
