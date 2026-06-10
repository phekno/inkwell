// Package cognito wraps the SRP flow against an AWS Cognito user pool and
// persists the resulting session locally.
package cognito

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	csrp "github.com/alexrudd/cognito-srp/v4"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	ciptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type Config struct {
	Region     string
	UserPoolID string
	ClientID   string
}

type Session struct {
	IDToken      string    `json:"id_token"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Email        string    `json:"email"`
}

type Client struct {
	cfg Config
	cip *cip.Client
}

func New(ctx context.Context, c Config) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.Region))
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	// Cognito's anonymous endpoints don't need creds, but the SDK still
	// wants a region; clear creds explicitly so SSO/IMDS lookups don't run.
	awsCfg.Credentials = aws.AnonymousCredentials{}
	return &Client{cfg: c, cip: cip.NewFromConfig(awsCfg)}, nil
}

func (c *Client) SignIn(ctx context.Context, email, password string) (*Session, error) {
	s, err := csrp.NewCognitoSRP(email, password, c.cfg.UserPoolID, c.cfg.ClientID, nil)
	if err != nil {
		return nil, fmt.Errorf("srp init: %w", err)
	}

	initResp, err := c.cip.InitiateAuth(ctx, &cip.InitiateAuthInput{
		AuthFlow:       ciptypes.AuthFlowTypeUserSrpAuth,
		ClientId:       aws.String(c.cfg.ClientID),
		AuthParameters: s.GetAuthParams(),
	})
	if err != nil {
		return nil, fmt.Errorf("initiate auth: %w", err)
	}
	if initResp.ChallengeName != ciptypes.ChallengeNameTypePasswordVerifier {
		return nil, fmt.Errorf("unexpected challenge: %s", initResp.ChallengeName)
	}

	resp, err := s.PasswordVerifierChallenge(initResp.ChallengeParameters, time.Now())
	if err != nil {
		return nil, fmt.Errorf("pw verifier: %w", err)
	}

	authResp, err := c.cip.RespondToAuthChallenge(ctx, &cip.RespondToAuthChallengeInput{
		ChallengeName:      ciptypes.ChallengeNameTypePasswordVerifier,
		ClientId:           aws.String(c.cfg.ClientID),
		ChallengeResponses: resp,
	})
	if err != nil {
		return nil, fmt.Errorf("respond challenge: %w", err)
	}
	if authResp.AuthenticationResult == nil {
		return nil, fmt.Errorf("unexpected post-challenge: %s", authResp.ChallengeName)
	}
	r := authResp.AuthenticationResult
	return &Session{
		IDToken:      aws.ToString(r.IdToken),
		AccessToken:  aws.ToString(r.AccessToken),
		RefreshToken: aws.ToString(r.RefreshToken),
		ExpiresAt:    time.Now().Add(time.Duration(r.ExpiresIn) * time.Second),
		Email:        email,
	}, nil
}

func (c *Client) Refresh(ctx context.Context, s *Session) error {
	resp, err := c.cip.InitiateAuth(ctx, &cip.InitiateAuthInput{
		AuthFlow:       ciptypes.AuthFlowTypeRefreshTokenAuth,
		ClientId:       aws.String(c.cfg.ClientID),
		AuthParameters: map[string]string{"REFRESH_TOKEN": s.RefreshToken},
	})
	if err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	if resp.AuthenticationResult == nil {
		return fmt.Errorf("refresh: no result")
	}
	r := resp.AuthenticationResult
	s.IDToken = aws.ToString(r.IdToken)
	s.AccessToken = aws.ToString(r.AccessToken)
	s.ExpiresAt = time.Now().Add(time.Duration(r.ExpiresIn) * time.Second)
	return nil
}

func sessionPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "inkwell", "session.json"), nil
}

func LoadSession() (*Session, error) {
	p, err := sessionPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p) // #nosec G304 -- path is the app's own config dir
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveSession(s *Session) error {
	p, err := sessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ") // #nosec G117 -- session-token persistence is intentional
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

func ClearSession() error {
	p, err := sessionPath()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
