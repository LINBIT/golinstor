package client

import (
	"context"
	"fmt"
)

type KV struct {
	Name  string            `json:"name"`
	Props map[string]string `json:"props"`
}

type KeyValueStoreProvider interface {
	List(ctx context.Context) ([]KV, error)
	Get(ctx context.Context, kv string) (*KV, error)
	CreateOrModify(ctx context.Context, kv string, modify GenericPropsModify) error
	Delete(ctx context.Context, kv string) error
}

var _ KeyValueStoreProvider = &KeyValueStoreService{}

type KeyValueStoreService struct {
	client *Client
}

// List returns the name of key-value stores and their values
func (k *KeyValueStoreService) List(ctx context.Context) ([]KV, error) {
	var ret []KV

	_, err := k.client.doGET(ctx, "/v1/key-value-store", &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (k *KeyValueStoreService) Get(ctx context.Context, kv string) (*KV, error) {
	var ret []KV

	_, err := k.client.doGET(ctx, "/v1/key-value-store/"+kv, &ret)
	if err != nil {
		return nil, err
	}

	if len(ret) != 1 {
		return nil, fmt.Errorf("expected exactly one KV, got %d", len(ret))
	}

	return &ret[0], nil
}

func (k *KeyValueStoreService) CreateOrModify(ctx context.Context, kv string, modify GenericPropsModify) error {
	_, err := k.client.doPUT(ctx, "/v1/key-value-store/"+kv, modify)
	return err
}

func (k *KeyValueStoreService) Delete(ctx context.Context, kv string) error {
	_, err := k.client.doDELETE(ctx, "/v1/key-value-store/"+kv, nil)
	return err
}
