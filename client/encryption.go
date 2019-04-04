package client

import "context"

// custom code

type EncryptionService struct {
	client *Client
}

type Passphrase struct {
	NewPassphrase string `json:"new_passphrase,omitempty"`
	OldPassphrase string `json:"old_passphrase,omitempty"`
}

func (n *EncryptionService) Create(ctx context.Context, passphrase Passphrase) error {
	_, err := n.client.doPOST(ctx, "/v1/encryption/passphrase", passphrase)
	return err
}

func (n *EncryptionService) Modify(ctx context.Context, passphrase Passphrase) error {
	_, err := n.client.doPUT(ctx, "/v1/encryption/passphrase", passphrase)
	return err
}

func (n *EncryptionService) Enter(ctx context.Context, password string) error {
	_, err := n.client.doPATCH(ctx, "/v1/encryption/passphrase", password)
	return err
}
