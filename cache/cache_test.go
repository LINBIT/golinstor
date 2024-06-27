package cache_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	linstor "github.com/LINBIT/golinstor"
	"github.com/LINBIT/golinstor/cache"
	"github.com/LINBIT/golinstor/client"
)

type TestResponse struct {
	Code int
	Body any
}

type TestServer struct {
	counter   map[string]int
	responses map[string]TestResponse
}

func (t *TestServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	req := request.Method + " " + request.URL.String()
	t.counter[req]++

	resp, ok := t.responses[req]
	if !ok {
		writer.WriteHeader(http.StatusInternalServerError)
	} else {
		writer.WriteHeader(resp.Code)
		_ = json.NewEncoder(writer).Encode(resp.Body)
	}
}

const (
	NodesRequest     = "GET /v1/nodes?cached=true&limit=0&offset=0"
	ReconnectRequest = "PUT /v1/nodes/node1/reconnect"
)

var (
	Node1 = client.Node{
		Name: "node1",
		Props: map[string]string{
			"Aux/key1": "val1",
			"Aux/key2": "val2",
		},
		Type: linstor.ValNodeTypeStlt,
	}
	Node2 = client.Node{
		Name: "node2",
		Props: map[string]string{
			"Aux/key2": "",
			"Aux/key3": "val3",
		},
		Type: linstor.ValNodeTypeCmbd,
	}
	AllNodes = []client.Node{Node1, Node2}
)

func TestNodeCache(t *testing.T) {
	testSrv := TestServer{
		counter: make(map[string]int),
		responses: map[string]TestResponse{
			NodesRequest:     {Code: http.StatusOK, Body: AllNodes},
			ReconnectRequest: {Code: http.StatusOK},
		},
	}
	srv := httptest.NewServer(&testSrv)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	assert.NoError(t, err)

	cl, err := client.NewClient(
		client.HTTPClient(srv.Client()),
		client.BaseURL(u),
		cache.WithCaches(&cache.NodeCache{Timeout: 1 * time.Second}),
	)
	assert.NoError(t, err)

	nodes, err := cl.Nodes.GetAll(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, AllNodes, nodes)

	node1, err := cl.Nodes.Get(context.Background(), "node1")
	assert.NoError(t, err)
	assert.Equal(t, Node1, node1)

	_, err = cl.Nodes.Get(context.Background(), "node3")
	assert.Equal(t, client.NotFoundError, err)

	// Assert that the request was only sent once
	assert.Equal(t, 1, testSrv.counter[NodesRequest])

	// Invalidate cache
	err = cl.Nodes.Reconnect(context.Background(), "node1")
	assert.NoError(t, err)

	node1, err = cl.Nodes.Get(context.Background(), "node1")
	assert.NoError(t, err)
	assert.Equal(t, Node1, node1)
	assert.Equal(t, 2, testSrv.counter[NodesRequest])

	// Wait for cache time out to be reached
	time.Sleep(1*time.Second + 100*time.Millisecond)
	node1, err = cl.Nodes.Get(context.Background(), "node1")
	assert.NoError(t, err)
	assert.Equal(t, Node1, node1)
	assert.Equal(t, 3, testSrv.counter[NodesRequest])
}

func TestNodeCachePropsFiltering(t *testing.T) {
	testSrv := TestServer{
		counter: make(map[string]int),
		responses: map[string]TestResponse{
			NodesRequest:     {Code: http.StatusOK, Body: AllNodes},
			ReconnectRequest: {Code: http.StatusOK},
		},
	}
	srv := httptest.NewServer(&testSrv)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	assert.NoError(t, err)

	cl, err := client.NewClient(
		client.HTTPClient(srv.Client()),
		client.BaseURL(u),
		cache.WithCaches(&cache.NodeCache{Timeout: 1 * time.Second}),
	)
	assert.NoError(t, err)

	// Filtering by presence of key
	nodes, err := cl.Nodes.GetAll(context.Background(), &client.ListOpts{Prop: []string{"Aux/key2"}})
	assert.NoError(t, err)
	assert.Equal(t, AllNodes, nodes)

	// Filtering by presence of key only on one node
	nodes, err = cl.Nodes.GetAll(context.Background(), &client.ListOpts{Prop: []string{"Aux/key3"}})
	assert.NoError(t, err)
	assert.Equal(t, []client.Node{Node2}, nodes)

	// Filtering by presence of key on no node
	nodes, err = cl.Nodes.GetAll(context.Background(), &client.ListOpts{Prop: []string{"Aux/other"}})
	assert.NoError(t, err)
	assert.Empty(t, nodes)

	// Filtering by presence of specific key=value
	nodes, err = cl.Nodes.GetAll(context.Background(), &client.ListOpts{Prop: []string{"Aux/key2=val2"}})
	assert.NoError(t, err)
	assert.Equal(t, []client.Node{Node1}, nodes)

	// Filtering by presence of specific key=""
	nodes, err = cl.Nodes.GetAll(context.Background(), &client.ListOpts{Prop: []string{"Aux/key2="}})
	assert.NoError(t, err)
	assert.Equal(t, []client.Node{Node2}, nodes)
}
