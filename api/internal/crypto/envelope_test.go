package crypto

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// fakeKMS keeps a single DEK in memory and pretends to wrap/unwrap it. The
// "wrapped" form is just a fixed marker + the context — that's enough to
// verify the sealer plumbs context through and that wrong contexts fail.
type fakeKMS struct {
	dek []byte
}

func newFakeKMS(t *testing.T) *fakeKMS {
	t.Helper()
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		t.Fatal(err)
	}
	return &fakeKMS{dek: dek}
}

func (f *fakeKMS) dekCopy() []byte {
	out := make([]byte, len(f.dek))
	copy(out, f.dek)
	return out
}

func (f *fakeKMS) GenerateDataKey(_ context.Context, in *kms.GenerateDataKeyInput, _ ...func(*kms.Options)) (*kms.GenerateDataKeyOutput, error) {
	wrapped := append([]byte("WRAP|"), []byte(in.EncryptionContext["sub"])...)
	return &kms.GenerateDataKeyOutput{
		Plaintext:      f.dekCopy(),
		CiphertextBlob: wrapped,
		KeyId:          aws.String("alias/inkwell-entries"),
	}, nil
}

func (f *fakeKMS) Decrypt(_ context.Context, in *kms.DecryptInput, _ ...func(*kms.Options)) (*kms.DecryptOutput, error) {
	expected := append([]byte("WRAP|"), []byte(in.EncryptionContext["sub"])...)
	if string(in.CiphertextBlob) != string(expected) {
		return nil, errors.New("kms: encryption context mismatch")
	}
	return &kms.DecryptOutput{Plaintext: f.dekCopy()}, nil
}

func TestSealOpenRoundTrip(t *testing.T) {
	s := &Sealer{KMS: newFakeKMS(t), KeyID: "alias/inkwell-entries"}
	plaintext := []byte("today I wrote a journal entry")

	env, err := s.Seal(context.Background(), "user-1", plaintext)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if len(env.Ciphertext) == 0 || len(env.Nonce) == 0 || len(env.WrappedDEK) == 0 {
		t.Fatalf("Seal produced empty fields: %+v", env)
	}
	if string(env.Ciphertext) == string(plaintext) {
		t.Fatalf("ciphertext equals plaintext")
	}

	got, err := s.Open(context.Background(), "user-1", env)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("plaintext mismatch: got %q want %q", got, plaintext)
	}
}

func TestOpenWithWrongUserFails(t *testing.T) {
	s := &Sealer{KMS: newFakeKMS(t), KeyID: "alias/inkwell-entries"}
	env, err := s.Seal(context.Background(), "user-1", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(context.Background(), "attacker", env); err == nil {
		t.Fatalf("expected error when opening with a different sub")
	}
}

func TestOpenWithTamperedCiphertextFails(t *testing.T) {
	s := &Sealer{KMS: newFakeKMS(t), KeyID: "alias/inkwell-entries"}
	env, err := s.Seal(context.Background(), "user-1", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	env.Ciphertext[0] ^= 0xff
	if _, err := s.Open(context.Background(), "user-1", env); err == nil {
		t.Fatalf("expected error on tampered ciphertext")
	}
}
