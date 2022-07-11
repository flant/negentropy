package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_HttpClient(t *testing.T) {
	testURL := "/bush/negentropy/backdoor"
	testKey := []byte("user/123456")
	testObj := []byte("{\"uuid\":\"123456\", \"identifier\":\"test_user\"}")

	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		require.Equal(t, req.URL.String(), testURL)
		// Send response to be tested
		body := req.Body
		defer body.Close()
		var data map[string]interface{}

		err := json.NewDecoder(req.Body).Decode(&data)
		require.NoError(t, err)

		require.NotNil(t, data["key"])
		require.Equal(t, string(testKey), data["key"])
		require.NotNil(t, data["object"])
		require.Equal(t, string(testObj), data["object"])
		rw.WriteHeader(200)
	}))

	defer server.Close()

	c := HTTPClient{
		Client: server.Client(),
		URL:    server.URL + testURL,
	}
	err := c.ProceedMessage(testObj, testKey)

	require.NoError(t, err)
}
