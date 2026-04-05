package microservice

import "encoding/json"

// Serializer transforms outgoing messages into bytes.
type Serializer interface {
	Serialize(data any) ([]byte, error)
}

// Deserializer transforms incoming bytes into structured data.
type Deserializer interface {
	Deserialize(data []byte) (any, error)
}

// IdentitySerializer passes data through json.Marshal.
type IdentitySerializer struct{}

func (s *IdentitySerializer) Serialize(data any) ([]byte, error) {
	return json.Marshal(data)
}

// IdentityDeserializer passes data through json.Unmarshal into a generic map.
type IdentityDeserializer struct{}

func (d *IdentityDeserializer) Deserialize(data []byte) (any, error) {
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// IncomingRequest represents a deserialized incoming request.
type IncomingRequest struct {
	Pattern Pattern         `json:"pattern"`
	Data    json.RawMessage `json:"data"`
	ID      string          `json:"id,omitempty"`
	IsEvent bool            `json:"isEvent,omitempty"`
}

// IncomingResponse represents a deserialized incoming response.
type IncomingResponse struct {
	ID    string          `json:"id,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// IncomingRequestDeserializer deserializes bytes into IncomingRequest.
type IncomingRequestDeserializer struct{}

func (d *IncomingRequestDeserializer) Deserialize(data []byte) (*IncomingRequest, error) {
	var req IncomingRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// IncomingResponseDeserializer deserializes bytes into IncomingResponse.
type IncomingResponseDeserializer struct{}

func (d *IncomingResponseDeserializer) Deserialize(data []byte) (*IncomingResponse, error) {
	var resp IncomingResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
