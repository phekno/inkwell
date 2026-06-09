// Package crypto implements envelope encryption for journal entries.
//
// Encryption flow:
//  1. Generate a 256-bit DEK with kms:GenerateDataKey under the inkwell CMK
//     and a per-user EncryptionContext{"sub": userID}.
//  2. Encrypt the entry body with XChaCha20-Poly1305 using the plaintext DEK.
//  3. Store the AAD-bound ciphertext alongside the wrapped DEK in DynamoDB.
//     The plaintext DEK is zeroed and never persisted.
//
// Decryption reverses the process via kms:Decrypt with the same context.
package crypto

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/kms"
)

type Envelope struct {
	Ciphertext []byte
	Nonce      []byte
	WrappedDEK []byte
	KeyID      string
}

type Sealer struct {
	KMS   *kms.Client
	KeyID string
}

func (s *Sealer) Seal(ctx context.Context, userID string, plaintext []byte) (*Envelope, error) {
	return nil, errors.New("not implemented")
}

func (s *Sealer) Open(ctx context.Context, userID string, env *Envelope) ([]byte, error) {
	return nil, errors.New("not implemented")
}
