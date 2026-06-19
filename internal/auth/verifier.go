package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"crm-backend/internal/config"
	"github.com/coreos/go-oidc/v3/oidc"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrBadAudience  = errors.New("token audience is not allowed")
)

type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (*Claims, error)
}

type RealmVerifier struct {
	realm    config.RealmConfig
	skew     time.Duration
	cacheTTL time.Duration

	mu       sync.Mutex
	verifier *oidc.IDTokenVerifier
	loadedAt time.Time
}

func NewRealmVerifier(realm config.RealmConfig, skew, cacheTTL time.Duration) *RealmVerifier {
	return &RealmVerifier{
		realm:    realm,
		skew:     skew,
		cacheTTL: cacheTTL,
	}
}

func (v *RealmVerifier) Verify(ctx context.Context, rawToken string) (*Claims, error) {
	verifier, err := v.getVerifier(ctx)
	if err != nil {
		return nil, err
	}

	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if err := v.validateTimes(claims); err != nil {
		return nil, err
	}
	if !claims.MatchesAudience(v.realm.Audience) {
		return nil, ErrBadAudience
	}

	return &claims, nil
}

func (v *RealmVerifier) getVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.verifier != nil && (v.cacheTTL <= 0 || time.Since(v.loadedAt) < v.cacheTTL) {
		return v.verifier, nil
	}

	provider, err := oidc.NewProvider(ctx, v.realm.Issuer)
	if err != nil {
		return nil, fmt.Errorf("initialize OIDC provider for realm %q: %w", v.realm.Name, err)
	}

	v.verifier = provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   true,
	})
	v.loadedAt = time.Now()
	return v.verifier, nil
}

func (v *RealmVerifier) validateTimes(claims Claims) error {
	now := time.Now()
	if claims.ExpiresAt == 0 {
		return fmt.Errorf("%w: missing exp claim", ErrInvalidToken)
	}
	if now.After(time.Unix(claims.ExpiresAt, 0).Add(v.skew)) {
		return fmt.Errorf("%w: token is expired", ErrInvalidToken)
	}
	if claims.NotBefore != 0 && now.Add(v.skew).Before(time.Unix(claims.NotBefore, 0)) {
		return fmt.Errorf("%w: token is not valid yet", ErrInvalidToken)
	}
	return nil
}
