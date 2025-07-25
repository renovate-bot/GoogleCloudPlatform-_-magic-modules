package acctest_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/hashicorp/terraform-provider-google/google/acctest"
)

func TestNewVcrMatcherFunc_canDetectMatches(t *testing.T) {

	// Everything should be determined as a match in this test
	cases := map[string]struct {
		httpRequest     requestDescription
		cassetteRequest requestDescription
	}{
		"matches POST requests with empty bodies": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{}",
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{}",
			},
		},
		"matches POST requests with exact matching bodies": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field\":\"value\"}",
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field\":\"value\"}",
			},
		},
		"matches POST requests with matching but re-ordered bodies, but only if Content-Type contains 'application/json'": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				body: "{\"field1\":\"value1\",\"field2\":\"value2\"}", // 1 before 2
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				body: "{\"field2\":\"value2\",\"field1\":\"value1\"}", // 2 before 1
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Make matcher
			ctx := context.Background()
			req := prepareHttpRequest(tc.httpRequest)
			cassetteReq := prepareCassetteRequest(tc.cassetteRequest)
			matcher := acctest.NewVcrMatcherFunc(ctx)

			// Act - use matcher
			matchDetected := matcher(req, cassetteReq)

			// Assert match
			if !matchDetected {
				t.Fatalf("expected matcher to match the requests")
			}
		})
	}
}

func TestNewVcrMatcherFunc_canDetectMismatches(t *testing.T) {

	// All these cases are expected to end with no match detected
	cases := map[string]struct {
		httpRequest     requestDescription
		cassetteRequest requestDescription
	}{
		"requests using different schemes": {
			httpRequest: requestDescription{
				scheme: "http",
				method: "GET",
				host:   "example.com",
				path:   "foobar",
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "GET",
				host:   "example.com",
				path:   "foobar",
			},
		},
		"requests using different hosts": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "GET",
				host:   "example.com",
				path:   "foobar",
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "GET",
				host:   "google.com",
				path:   "foobar",
			},
		},
		"requests using different paths": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "GET",
				host:   "example.com",
				path:   "foobar1",
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "GET",
				host:   "example.com",
				path:   "foobar2",
			},
		},
		"requests with different methods": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{}",
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "PUT",
				host:   "example.com",
				path:   "foobar",
				body:   "{}",
			},
		},
		"POST requests with different bodies": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field\":\"value is ABCDEFG\"}",
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field\":\"value is MNLOP\"}",
			},
		},
		"POST requests with matching but re-ordered bodies aren't matching if Content-Type header is not 'application/json'": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				headers: map[string]string{
					"Content-Type": "foobar",
				},
				body: "{\"field1\":\"value1\",\"field2\":\"value2\"}", // 1 before 2
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				headers: map[string]string{
					"Content-Type": "foobar",
				},
				body: "{\"field2\":\"value2\",\"field1\":\"value1\"}", // 2 before 1
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Make matcher
			ctx := context.Background()
			req := prepareHttpRequest(tc.httpRequest)
			cassetteReq := prepareCassetteRequest(tc.cassetteRequest)
			matcher := acctest.NewVcrMatcherFunc(ctx)

			// Act - use matcher
			matchDetected := matcher(req, cassetteReq)

			// Assert match
			if matchDetected {
				t.Fatalf("expected matcher to not match the requests")
			}
		})
	}
}

// Currently there is no code to actively force the matcher to overlook differing User-Agent values.
// It isn't checked at any point in the matcher logic.
func TestNewVcrMatcherFunc_ignoresDifferentUserAgents(t *testing.T) {

	cases := map[string]struct {
		httpRequest     requestDescription
		cassetteRequest requestDescription
	}{
		"GET requests with different useragents are matched": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "GET",
				host:   "example.com",
				path:   "foobar",
				headers: map[string]string{
					"User-Agent": "user-agent-HTTP",
				},
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "GET",
				host:   "example.com",
				path:   "foobar",
				headers: map[string]string{
					"User-Agent": "user-agent-CASSETTE",
				},
			},
		},
		"POST requests with identical bodies and different useragents are matched": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field\":\"value\"}",
				headers: map[string]string{
					"User-Agent": "user-agent-HTTP",
				},
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field\":\"value\"}",
				headers: map[string]string{
					"User-Agent": "user-agent-CASSETTE",
				},
			},
		},
		"POST requests with reordered but matching bodies and different useragents are matched if Content-Type contains 'application/json'": {
			httpRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field1\":\"value1\",\"field2\":\"value2\"}",
				headers: map[string]string{
					"User-Agent":   "user-agent-HTTP",
					"Content-Type": "application/json",
				},
			},
			cassetteRequest: requestDescription{
				scheme: "https",
				method: "POST",
				host:   "example.com",
				path:   "foobar",
				body:   "{\"field2\":\"value2\",\"field1\":\"value1\"}",
				headers: map[string]string{
					"User-Agent":   "user-agent-CASSETTE",
					"Content-Type": "application/json",
				},
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Make matcher
			ctx := context.Background()
			req := prepareHttpRequest(tc.httpRequest)
			cassetteReq := prepareCassetteRequest(tc.cassetteRequest)
			matcher := acctest.NewVcrMatcherFunc(ctx)

			// Act - use matcher
			matchDetected := matcher(req, cassetteReq)

			// Assert match
			if !matchDetected {
				t.Fatalf("expected matcher to match the requests")
			}
		})
	}
}

type requestDescription struct {
	scheme  string
	method  string
	host    string
	path    string
	body    string
	headers map[string]string
}

func prepareHttpRequest(d requestDescription) *http.Request {
	url := &url.URL{
		Scheme: d.scheme,
		Host:   d.host,
		Path:   d.path,
	}

	req := &http.Request{
		Method: d.method,
		URL:    url,
	}

	// Conditionally set a body
	if d.body != "" {
		body := io.NopCloser(bytes.NewBufferString(d.body))
		req.Body = body
	}
	// Conditionally set headers
	if len(d.headers) > 0 {
		req.Header = http.Header{}
		for k, v := range d.headers {
			req.Header.Set(k, v)
		}
	}

	return req
}

func prepareCassetteRequest(d requestDescription) cassette.Request {
	fullUrl := fmt.Sprintf("%s://%s/%s", d.scheme, d.host, d.path)

	req := cassette.Request{
		Method: d.method,
		URL:    fullUrl,
	}

	// Conditionally set a body
	if d.body != "" {
		req.Body = d.body
	}
	// Conditionally set headers
	if len(d.headers) > 0 {
		req.Headers = http.Header{}
		for k, v := range d.headers {
			req.Headers.Add(k, v)
		}
	}

	return req
}

func TestReformConfigWithProvider(t *testing.T) {

	type testCase struct {
		name             string
		initialConfig    string
		providerToInsert string
		expectedConfig   string
	}

	cases := map[string]testCase{
		"replaces_google_beta_with_local": {
			name: "Replaces 'google-beta' provider with 'google-local'",
			initialConfig: `resource "google_new_resource" {
      provider = google-beta
}`,
			providerToInsert: "google-local",
			expectedConfig: `resource "google_new_resource" {
      provider = google-local
}`,
		},
		"inserts_local_provider_into_empty_config": {
			name: "Inserts 'google-local' provider when no provider block exists",
			initialConfig: `resource "google_alloydb_cluster" "default" {
    location   = "us-central1"
    network_config {
        network = google_compute_network.default.id
    }
}`,
			providerToInsert: "google-local",
			expectedConfig: `resource "google_alloydb_cluster" "default" {
  provider = google-local

    location   = "us-central1"
    network_config {
        network = google_compute_network.default.id
    }
}`,
		},
		"no_change_if_target_provider_already_present": {
			name: "Does not change config if target provider is already present",
			initialConfig: `resource "google_new_resource" {
      provider = google-local
}`,
			providerToInsert: "google-local",
			expectedConfig: `resource "google_new_resource" {
      provider = google-local
}`,
		},
		"inserts_provider_with_other_attributes": {
			name: "Inserts provider into a resource block with other attributes but no existing provider",
			initialConfig: `resource "google_compute_instance" "test" {
  name         = "test-instance"
  machine_type = "e2-medium"
}`,
			providerToInsert: "google-local",
			expectedConfig: `resource "google_compute_instance" "test" {
  provider = google-local

  name         = "test-instance"
  machine_type = "e2-medium"
}`,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			newConfig := acctest.ReformConfigWithProvider(tc.initialConfig, tc.providerToInsert)

			if newConfig != tc.expectedConfig {
				t.Fatalf("Test Case: %s\nExpected config to be reformatted to:\n%q\nbut got:\n%q", tc.name, tc.expectedConfig, newConfig)
			}
			t.Logf("Test Case: %s\nReformed config:\n%s", tc.name, newConfig)
		})
	}
}
