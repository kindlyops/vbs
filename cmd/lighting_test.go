// Copyright © 2025 Kindly Ops, LLC <support@kindlyops.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func TestHandleOSC_MethodNotAllowed(t *testing.T) {
	// Setup test configuration
	viper.Set("companion_buttons", map[string]string{
		"green": "/press/bank/20/10",
	})

	// Create a GET request (should fail)
	req := httptest.NewRequest(http.MethodGet, "/api/light/green", nil)
	w := httptest.NewRecorder()

	handleOSC(w, req, "/api/light/", viper.GetStringMapString("companion_buttons"))

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Should have JSON content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestHandleOSC_ValidButton(t *testing.T) {
	// Setup test configuration
	viper.Set("companion", "127.0.0.1")
	viper.Set("companion_buttons", map[string]string{
		"green": "/press/bank/20/10",
		"blue":  "/press/bank/20/11",
	})

	testCases := []struct {
		name       string
		url        string
		prefix     string
		wantStatus int
	}{
		{
			name:       "valid green light button",
			url:        "/api/light/green",
			prefix:     "/api/light/",
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid blue light button",
			url:        "/api/light/blue",
			prefix:     "/api/light/",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.url, nil)
			w := httptest.NewRecorder()

			handleOSC(w, req, tc.prefix, viper.GetStringMapString("companion_buttons"))

			if w.Code != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

func TestHandleOSC_UnknownButton(t *testing.T) {
	// Setup test configuration
	viper.Set("companion_buttons", map[string]string{
		"green": "/press/bank/20/10",
	})

	// Create a POST request with unknown button
	req := httptest.NewRequest(http.MethodPost, "/api/light/purple", nil)
	w := httptest.NewRecorder()

	handleOSC(w, req, "/api/light/", viper.GetStringMapString("companion_buttons"))

	// Should return 400 Bad Request for unknown button
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSwitcher_ServeHTTP(t *testing.T) {
	// Setup test configuration
	viper.Set("companion", "127.0.0.1")
	viper.Set("companion_buttons", map[string]string{
		"ftb": "/press/bank/20/4",
		"dsk": "/press/bank/20/5",
	})

	switcher := &Switcher{}

	testCases := []struct {
		name       string
		url        string
		method     string
		wantStatus int
	}{
		{
			name:       "valid FTB button",
			url:        "/api/switcher/ftb",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid DSK button",
			url:        "/api/switcher/dsk",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid method",
			url:        "/api/switcher/ftb",
			method:     http.MethodGet,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown button",
			url:        "/api/switcher/unknown",
			method:     http.MethodPost,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			switcher.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestLighting_ServeHTTP(t *testing.T) {
	// Setup test configuration
	viper.Set("companion", "127.0.0.1")
	viper.Set("companion_buttons", map[string]string{
		"green":  "/press/bank/20/10",
		"blue":   "/press/bank/20/11",
		"red":    "/press/bank/20/12",
		"yellow": "/press/bank/20/13",
		"off":    "/press/bank/20/14",
	})

	lighting := &Lighting{}

	testCases := []struct {
		name       string
		url        string
		method     string
		wantStatus int
	}{
		{
			name:       "valid green light",
			url:        "/api/light/green",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid off button",
			url:        "/api/light/off",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid method",
			url:        "/api/light/green",
			method:     http.MethodGet,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown color",
			url:        "/api/light/purple",
			method:     http.MethodPost,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			lighting.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestSendOKResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/light/green", nil)
	w := httptest.NewRecorder()

	sendOKResponse(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	body := w.Body.String()
	if body != "{}" {
		t.Errorf("Expected body '{}', got '%s'", body)
	}
}

func TestSendFailureResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/light/invalid", nil)
	w := httptest.NewRecorder()

	sendFailureResponse(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	body := w.Body.String()
	if body != "{}" {
		t.Errorf("Expected body '{}', got '%s'", body)
	}
}
