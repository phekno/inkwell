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
	"crypto/rand"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"golang.org/x/crypto/chacha20poly1305"
)

type Envelope struct {
	Ciphertext []byte
	Nonce      []byte
	WrappedDEK []byte
}

// KMSAPI is the subset of *kms.Client the Sealer needs. Letting callers
// inject a mock keeps tests off the network.
type KMSAPI interface {
	GenerateDataKey(ctx context.Context, params *kms.GenerateDataKeyInput, optFns ...func(*kms.Options)) (*kms.GenerateDataKeyOutput, error)
	Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

type Sealer struct {
	KMS   KMSAPI
	KeyID string
}

func (s *Sealer) Seal(ctx context.Context, userID string, plaintext []byte) (*Envelope, error) {
	out, err := s.KMS.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:             aws.String(s.KeyID),
		KeySpec:           kmstypes.DataKeySpecAes256,
		EncryptionContext: map[string]string{"sub": userID},
	})
	if err != nil {
		return nil, fmt.Errorf("kms: generate data key: %w", err)
	}
	defer zero(out.Plaintext)

	aead, err := chacha20poly1305.NewX(out.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("aead: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	// AD binds the ciphertext to the user; tampering with WrappedDEK
	// across users still fails to decrypt because KMS won't unwrap with
	// the wrong sub, but binding here is belt-and-suspenders.
	ad := []byte("inkwell|sub=" + userID)
	ct := aead.Seal(nil, nonce, plaintext, ad)

	return &Envelope{
		Ciphertext: ct,
		Nonce:      nonce,
		WrappedDEK: out.CiphertextBlob,
	}, nil
}

func (s *Sealer) Open(ctx context.Context, userID string, env *Envelope) ([]byte, error) {
	out, err := s.KMS.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob:    env.WrappedDEK,
		EncryptionContext: map[string]string{"sub": userID},
	})
	if err != nil {
		return nil, fmt.Errorf("kms: decrypt dek: %w", err)
	}
	defer zero(out.Plaintext)

	aead, err := chacha20poly1305.NewX(out.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("aead: %w", err)
	}

	ad := []byte("inkwell|sub=" + userID)
	pt, err := aead.Open(nil, env.Nonce, env.Ciphertext, ad)
	if err != nil {
		return nil, fmt.Errorf("aead open: %w", err)
	}
	return pt, nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
