package client

import (
	"context"
	"fmt"

	"github.com/google/go-querystring/query"
)

type Connection struct {
	NodeA string            `json:"node_a,omitempty"`
	NodeB string            `json:"node_b,omitempty"`
	Props map[string]string `json:"props,omitempty"`
	Flags []string          `json:"flags,omitempty"`
	Port  *int32            `json:"port,omitempty"`
}

// ConnectionProvider acts as an abstraction for a ConnectionService. It can be swapped out
// for another ConnectionService implementation, for example for testing.
type ConnectionProvider interface {
	// GetNodeConnections lists all node connections, optionally limites to nodes A and B, if not empty.
	GetNodeConnections(ctx context.Context, nodeA, nodeB string) ([]Connection, error)
	// GetResourceConnections returns all connections of the given resource.
	GetResourceConnections(ctx context.Context, resource string) ([]Connection, error)
	// GetResourceConnection returns the connection between node A and B for the given resource.
	GetResourceConnection(ctx context.Context, resource, nodeA, nodeB string) (*Connection, error)
	// SetNodeConnection sets or updates the node connection between node A and B.
	SetNodeConnection(ctx context.Context, nodeA, nodeB string, props GenericPropsModify) error
	// SetResourceConnection sets or updates the connection between node A and B for a resource.
	SetResourceConnection(ctx context.Context, resource, nodeA, nodeB string, props GenericPropsModify) error
}

// ConnectionService is the service that deals with connection related tasks.
type ConnectionService struct {
	client *Client
}

func (c *ConnectionService) GetNodeConnections(ctx context.Context, nodeA, nodeB string) ([]Connection, error) {
	nodeA, nodeB = sortNodes(nodeA, nodeB)

	vals, err := query.Values(struct {
		NodeA string `url:"node_a,omitempty"`
		NodeB string `url:"node_b,omitempty"`
	}{NodeA: nodeA, NodeB: nodeB})
	if err != nil {
		return nil, fmt.Errorf("failed to encode node names: %w", err)
	}

	var conns []Connection
	_, err = c.client.doGET(ctx, "/v1/node-connections?"+vals.Encode(), &conns)
	return conns, err
}

func (c *ConnectionService) GetResourceConnections(ctx context.Context, resource string) ([]Connection, error) {
	var conns []Connection
	_, err := c.client.doGET(ctx, "/v1/resource-definitions/"+resource+"/resource-connections", &conns)
	return conns, err
}

func (c *ConnectionService) GetResourceConnection(ctx context.Context, resource, nodeA, nodeB string) (*Connection, error) {
	nodeA, nodeB = sortNodes(nodeA, nodeB)

	var conn Connection
	_, err := c.client.doGET(ctx, "/v1/resource-definitions/"+resource+"/resource-connections/"+nodeA+"/"+nodeB, &conn)
	if err != nil {
		return nil, err
	}
	return &conn, err
}

func (c *ConnectionService) SetNodeConnection(ctx context.Context, nodeA, nodeB string, props GenericPropsModify) error {
	nodeA, nodeB = sortNodes(nodeA, nodeB)
	_, err := c.client.doPUT(ctx, "/v1/node-connections/"+nodeA+"/"+nodeB, &props)
	return err
}

func (c *ConnectionService) SetResourceConnection(ctx context.Context, resource, nodeA, nodeB string, props GenericPropsModify) error {
	nodeA, nodeB = sortNodes(nodeA, nodeB)
	_, err := c.client.doPUT(ctx, "/v1/resource-definitions/"+resource+"/resource-connections/"+nodeA+"/"+nodeB, &props)
	return err
}

// Sort node parameters: LINSTOR is (sometimes) order-dependent with parameters.
func sortNodes(a, b string) (string, string) {
	if a < b {
		return a, b
	} else {
		return b, a
	}
}
