package wgmesh

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

// Client provides a wrapper for communicating with the wg-mesh JSON-RPC API.
type Client struct {
	socketPath string
}

// NewClient creates a new wg-mesh client.
func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// --- JSON-RPC Structs based on user-provided source ---

// RPCRequest is a standard JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
	ID      int                    `json:"id"`
}

// RPCResponse is a standard JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// PeerInfo represents the structure of a peer from the 'peers.list' method.
type PeerInfo struct {
	Name     string `json:"name"`
	PubKey   string `json:"pubkey"`
	MeshIP   string `json:"mesh_ip"`
	Endpoint string `json:"endpoint"`
	LastSeen string `json:"last_seen"`
}

// PeersListResult is the nested result for a 'peers.list' call.
type PeersListResult struct {
	Peers []*PeerInfo `json:"peers"`
}

// GetPeers connects to the wg-mesh socket and fetches the list of peers.
func (c *Client) GetPeers() ([]*PeerInfo, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("could not connect to wg-mesh socket at %s: %w", c.socketPath, err)
	}
	defer conn.Close()

	request := RPCRequest{
		JSONRPC: "2.0",
		Method:  "peers.list",
		ID:      1,
	}

	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// wg-mesh expects newline-delimited requests
	_, err = conn.Write(append(reqBytes, '\n'))
	if err != nil {
		return nil, fmt.Errorf("failed to write to socket: %w", err)
	}

	reader := bufio.NewReader(conn)
	resBytes, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response from socket: %w", err)
	}

	var response RPCResponse
	if err := json.Unmarshal(resBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON-RPC response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("received error from wg-mesh: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	var peersResult PeersListResult
	if err := json.Unmarshal(response.Result, &peersResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nested peers result: %w", err)
	}

	return peersResult.Peers, nil
}
