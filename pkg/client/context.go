package client

import (
	"context"
)

// contextKey is the key type for storing values in context
type contextKey string

const (
	clientContextKey contextKey = "n9e_client"
)

// GetClientFunc is the function type for getting Client from context
type GetClientFunc func(ctx context.Context) *Client

// ContextWithClient injects Client into context
func ContextWithClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, clientContextKey, client)
}

// ClientFromContext gets Client from context
func ClientFromContext(ctx context.Context) *Client {
	client, ok := ctx.Value(clientContextKey).(*Client)
	if !ok {
		return nil
	}
	return client
}

// MustClientFromContext gets Client from context, panics if not found
func MustClientFromContext(ctx context.Context) *Client {
	client := ClientFromContext(ctx)
	if client == nil {
		panic("n9e client not found in context")
	}
	return client
}

// DefaultGetClient is the default function to get Client
func DefaultGetClient(ctx context.Context) *Client {
	return ClientFromContext(ctx)
}
