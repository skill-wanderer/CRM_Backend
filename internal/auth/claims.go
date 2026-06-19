package auth

import (
	"encoding/json"
	"strings"
)

type Claims struct {
	Subject           string      `json:"sub"`
	Issuer            string      `json:"iss"`
	Audience          Audience    `json:"aud"`
	AuthorizedParty   string      `json:"azp"`
	Email             string      `json:"email"`
	Name              string      `json:"name"`
	PreferredUsername string      `json:"preferred_username"`
	RealmAccess       RealmAccess `json:"realm_access"`
	ExpiresAt         int64       `json:"exp"`
	NotBefore         int64       `json:"nbf"`
}

type RealmAccess struct {
	Roles []string `json:"roles"`
}

type Audience []string

func (a *Audience) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = Audience{single}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*a = Audience(many)
	return nil
}

func (a Audience) Contains(value string) bool {
	for _, item := range a {
		if item == value {
			return true
		}
	}
	return false
}

func (c *Claims) DisplayName() string {
	if strings.TrimSpace(c.Name) != "" {
		return strings.TrimSpace(c.Name)
	}
	return strings.TrimSpace(c.PreferredUsername)
}

func (c *Claims) HasRealmRole(role string) bool {
	for _, item := range c.RealmAccess.Roles {
		if item == role {
			return true
		}
	}
	return false
}

func (c *Claims) MatchesAudience(expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	return c.Audience.Contains(expected) || c.AuthorizedParty == expected
}
