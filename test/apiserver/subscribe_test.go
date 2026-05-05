// nolint: errcheck
package apiserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/subscribe"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var wsUpgrader = websocket.Upgrader{} //nolint:exhaustruct

const timeoutInSeconds = 3 * time.Second

// newSubscribeRouter builds a ServeMux that routes collection and resource
// endpoints to s, plus a /v1/subscribe WebSocket endpoint backed directly by
// subscribe.NewWatchSession.
func newSubscribeRouter(s *server.Server) *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("/{prefix}/{type}", s)
	router.Handle("/{prefix}/{type}/{name}", s)
	router.HandleFunc("/v1/subscribe", func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			// If this fails, other tests will fail, but emit the error message for debugging
			fmt.Printf("Failed to upgrade to a websocket: %s\n", err.Error())
			return
		}
		defer conn.Close()

		apiOp := &types.APIRequest{
			Request:       r,
			Response:      w,
			Schemas:       s.Schemas,
			AccessControl: &server.SchemaBasedAccess{},
		}

		session := subscribe.NewWatchSession(apiOp, subscribe.DefaultGetter)
		defer session.Close()

		for event := range session.Watch(conn) {
			if event.Error != nil {
				event.Name = "resource.error"
				event.Data = map[string]interface{}{"error": event.Error.Error()}
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		}
	})
	return router
}

// readWSEvent reads the next WebSocket message and decodes it as a map.
func readWSEvent(t *testing.T, conn *websocket.Conn, timeout time.Duration) map[string]interface{} {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(timeout)))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	var event map[string]interface{}
	require.NoError(t, json.Unmarshal(msg, &event))
	return event
}

// TestSubscribe_DogAndCatEvents verifies that a WebSocket subscriber receives
// resource.create, resource.change, and resource.remove events when dogs and
// cats are mutated via the REST API.
func TestSubscribe_DogAndCatEvents(t *testing.T) {
	s := server.DefaultAPIServer()
	dogStore := NewDogStore(nil)
	catStore := NewCatStore(nil)

	s.Schemas.MustImportAndCustomize(Dog{}, func(schema *types.APISchema) {
		schema.Store = dogStore
		schema.CollectionMethods = []string{http.MethodGet, http.MethodPost}
		schema.ResourceMethods = []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	})
	s.Schemas.MustImportAndCustomize(Cat{}, func(schema *types.APISchema) {
		schema.Store = catStore
		schema.CollectionMethods = []string{http.MethodGet, http.MethodPost}
		schema.ResourceMethods = []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	})

	ts := httptest.NewServer(newSubscribeRouter(s))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/subscribe"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Subscribe to dog events.
	subMsg, _ := json.Marshal(map[string]string{"resourceType": "dog"})
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, subMsg))

	event := readWSEvent(t, conn, timeoutInSeconds)
	assert.Equal(t, "resource.start", event["name"], "expected resource.start for dog subscription")
	assert.Equal(t, "dog", event["resourceType"])

	// Subscribe to cat events.
	subMsg, _ = json.Marshal(map[string]string{"resourceType": "cat"})
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, subMsg))

	event = readWSEvent(t, conn, timeoutInSeconds)
	assert.Equal(t, "resource.start", event["name"], "expected resource.start for cat subscription")
	assert.Equal(t, "cat", event["resourceType"])

	t.Run("create dog emits resource.create", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/v1/dogs", "application/json",
			bytes.NewBufferString(`{"id":"mcruff","name":"fleethound"}`))
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		event = readWSEvent(t, conn, timeoutInSeconds)
		assert.Equal(t, "resource.create", event["name"])
		assert.Equal(t, "dog", event["resourceType"])
	})

	t.Run("update dog emits resource.change", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/v1/dogs/pluto",
			bytes.NewBufferString(`{"name":"drisney"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		event = readWSEvent(t, conn, timeoutInSeconds)
		assert.Equal(t, "resource.change", event["name"])
		assert.Equal(t, "dog", event["resourceType"])
	})

	t.Run("delete dog emits resource.remove", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/v1/dogs/krypto", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		event = readWSEvent(t, conn, timeoutInSeconds)
		assert.Equal(t, "resource.remove", event["name"])
		assert.Equal(t, "dog", event["resourceType"])
	})

	t.Run("create cat emits resource.create", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/v1/cats", "application/json",
			bytes.NewBufferString(`{"id":"shnopsker","name":"nutbag"}`))
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		event = readWSEvent(t, conn, timeoutInSeconds)
		assert.Equal(t, "resource.create", event["name"])
		assert.Equal(t, "cat", event["resourceType"])
	})

	t.Run("delete cat emits resource.remove", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/v1/cats/felix", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		event = readWSEvent(t, conn, timeoutInSeconds)
		assert.Equal(t, "resource.remove", event["name"])
		assert.Equal(t, "cat", event["resourceType"])
	})

	t.Run("update cat emits resource.change", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/v1/cats/fritz",
			bytes.NewBufferString(`{"name":"updated-zap"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		event = readWSEvent(t, conn, timeoutInSeconds)
		assert.Equal(t, "resource.change", event["name"])
		assert.Equal(t, "cat", event["resourceType"])
	})
}
