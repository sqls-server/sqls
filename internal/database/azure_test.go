package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

func TestAzureAuthConfig_method(t *testing.T) {
	tests := []struct {
		name string
		cfg  *AzureAuthConfig
		want string
	}{
		{name: "nil defaults to the azure cli", cfg: nil, want: AzureAuthMethodAzureCLI},
		{name: "empty defaults to the azure cli", cfg: &AzureAuthConfig{}, want: AzureAuthMethodAzureCLI},
		{name: "case insensitive", cfg: &AzureAuthConfig{Method: "AzCli"}, want: AzureAuthMethodAzureCLI},
		{name: "dashes are ignored", cfg: &AzureAuthConfig{Method: "managed-identity"}, want: AzureAuthMethodManagedIdentity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.method(); got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestAzureAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *AzureAuthConfig
		wantErr bool
	}{
		{name: "empty", cfg: &AzureAuthConfig{}},
		{name: "azcli", cfg: &AzureAuthConfig{Method: AzureAuthMethodAzureCLI}},
		{name: "default", cfg: &AzureAuthConfig{Method: AzureAuthMethodDefault}},
		{name: "devcli", cfg: &AzureAuthConfig{Method: AzureAuthMethodAzureDeveloperCLI}},
		{name: "managedidentity", cfg: &AzureAuthConfig{Method: AzureAuthMethodManagedIdentity}},
		{name: "environment", cfg: &AzureAuthConfig{Method: AzureAuthMethodEnvironment}},
		{name: "unknown", cfg: &AzureAuthConfig{Method: "kerberos"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}
		})
	}
}

type fakeCredential struct {
	calls  int
	scopes []string
	expiry time.Duration
	err    error
}

func (c *fakeCredential) GetToken(_ context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.calls++
	c.scopes = append(c.scopes, opts.Scopes...)
	if c.err != nil {
		return azcore.AccessToken{}, c.err
	}
	return azcore.AccessToken{
		Token:     "token-" + opts.Scopes[0],
		ExpiresOn: time.Now().Add(c.expiry),
	}, nil
}

func newTestProvider(cred azcore.TokenCredential, defaultScope string, pinned bool) *azureTokenProvider {
	return &azureTokenProvider{
		cred:         cred,
		defaultScope: defaultScope,
		pinnedScope:  pinned,
		tokens:       map[string]azcore.AccessToken{},
	}
}

func TestAzureTokenProvider_Token(t *testing.T) {
	ctx := context.Background()

	t.Run("caches per scope until close to expiry", func(t *testing.T) {
		cred := &fakeCredential{expiry: time.Hour}
		p := newTestProvider(cred, AzureScopeOSSRDBMS, false)

		first, err := p.Token(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Token(ctx, ""); err != nil {
			t.Fatal(err)
		}
		if cred.calls != 1 {
			t.Errorf("want 1 credential call, got %d", cred.calls)
		}

		other, err := p.Token(ctx, "https://database.windows.net/")
		if err != nil {
			t.Fatal(err)
		}
		if cred.calls != 2 {
			t.Errorf("want 2 credential calls for a second scope, got %d", cred.calls)
		}
		if first == other {
			t.Error("want a distinct token per scope")
		}
	})

	t.Run("refreshes a token inside the expiry margin", func(t *testing.T) {
		cred := &fakeCredential{expiry: azureTokenExpiryMargin / 2}
		p := newTestProvider(cred, AzureScopeOSSRDBMS, false)

		if _, err := p.Token(ctx, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Token(ctx, ""); err != nil {
			t.Fatal(err)
		}
		if cred.calls != 2 {
			t.Errorf("want 2 credential calls, got %d", cred.calls)
		}
	})

	t.Run("appends the default suffix to a bare resource", func(t *testing.T) {
		cred := &fakeCredential{expiry: time.Hour}
		p := newTestProvider(cred, "", false)

		if _, err := p.Token(ctx, "https://ossrdbms-aad.database.windows.net"); err != nil {
			t.Fatal(err)
		}
		if len(cred.scopes) != 1 || cred.scopes[0] != AzureScopeOSSRDBMS {
			t.Errorf("want scope %q, got %v", AzureScopeOSSRDBMS, cred.scopes)
		}
	})

	// The server SPN of Azure SQL already ends in a slash, so the scope ends up
	// with a double slash. Azure accepts that, and it is what the driver's own
	// azuread package sends, so keep the resource untouched.
	t.Run("keeps the trailing slash of a server spn", func(t *testing.T) {
		cred := &fakeCredential{expiry: time.Hour}
		p := newTestProvider(cred, "", false)

		if _, err := p.Token(ctx, "https://database.windows.net/"); err != nil {
			t.Fatal(err)
		}
		want := "https://database.windows.net//.default"
		if len(cred.scopes) != 1 || cred.scopes[0] != want {
			t.Errorf("want scope %q, got %v", want, cred.scopes)
		}
	})

	t.Run("a pinned scope wins over the requested one", func(t *testing.T) {
		cred := &fakeCredential{expiry: time.Hour}
		p := newTestProvider(cred, "https://custom.example/.default", true)

		if _, err := p.Token(ctx, "https://database.windows.net/"); err != nil {
			t.Fatal(err)
		}
		if len(cred.scopes) != 1 || cred.scopes[0] != "https://custom.example/.default" {
			t.Errorf("want the pinned scope, got %v", cred.scopes)
		}
	})

	t.Run("errors without any scope", func(t *testing.T) {
		p := newTestProvider(&fakeCredential{expiry: time.Hour}, "", false)
		if _, err := p.Token(ctx, ""); err == nil {
			t.Error("want an error when no scope is known")
		}
	})

	t.Run("wraps credential failures", func(t *testing.T) {
		want := errors.New("please run az login")
		p := newTestProvider(&fakeCredential{err: want}, AzureScopeOSSRDBMS, false)
		_, err := p.Token(ctx, "")
		if !errors.Is(err, want) {
			t.Errorf("want the credential error wrapped, got %v", err)
		}
	})
}
