package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// Azure AD (Microsoft Entra ID) credential sources accepted in
// connections[].azureAuth.method.
const (
	// AzureAuthMethodAzureCLI reuses the session created by `az login`.
	AzureAuthMethodAzureCLI = "azcli"
	// AzureAuthMethodAzureDeveloperCLI reuses the session created by `azd auth login`.
	AzureAuthMethodAzureDeveloperCLI = "devcli"
	// AzureAuthMethodDefault walks the DefaultAzureCredential chain, which
	// ends with the Azure CLI session.
	AzureAuthMethodDefault = "default"
	// AzureAuthMethodManagedIdentity uses the identity assigned to the host.
	AzureAuthMethodManagedIdentity = "managedidentity"
	// AzureAuthMethodEnvironment uses the AZURE_* service principal variables.
	AzureAuthMethodEnvironment = "environment"
)

// AzureScopeOSSRDBMS is the token scope used by Azure Database for PostgreSQL
// and Azure Database for MySQL.
const AzureScopeOSSRDBMS = "https://ossrdbms-aad.database.windows.net/.default"

const azureScopeSuffix = "/.default"

// azureTokenExpiryMargin is how long before its actual expiry a cached token
// is considered stale, so a connection is never opened with a token that
// expires mid-handshake.
const azureTokenExpiryMargin = 5 * time.Minute

// AzureAuthConfig enables Azure AD authentication for a connection. An empty
// method reuses the `az login` session.
type AzureAuthConfig struct {
	Method   string `json:"method" yaml:"method"`
	TenantID string `json:"tenantId" yaml:"tenantId"`
	ClientID string `json:"clientId" yaml:"clientId"`
	Scope    string `json:"scope" yaml:"scope"`
}

func (c *AzureAuthConfig) method() string {
	if c == nil || c.Method == "" {
		return AzureAuthMethodAzureCLI
	}
	return strings.ToLower(strings.ReplaceAll(c.Method, "-", ""))
}

func (c *AzureAuthConfig) Validate() error {
	switch c.method() {
	case AzureAuthMethodAzureCLI,
		AzureAuthMethodAzureDeveloperCLI,
		AzureAuthMethodDefault,
		AzureAuthMethodManagedIdentity,
		AzureAuthMethodEnvironment:
		return nil
	default:
		return fmt.Errorf(
			"invalid: connections[].azureAuth.method, %q is not one of %s, %s, %s, %s, %s",
			c.Method,
			AzureAuthMethodAzureCLI,
			AzureAuthMethodAzureDeveloperCLI,
			AzureAuthMethodDefault,
			AzureAuthMethodManagedIdentity,
			AzureAuthMethodEnvironment,
		)
	}
}

// Credential builds the token source described by the config.
func (c *AzureAuthConfig) Credential() (azcore.TokenCredential, error) {
	switch c.method() {
	case AzureAuthMethodAzureCLI:
		return azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{
			TenantID: c.TenantID,
		})
	case AzureAuthMethodAzureDeveloperCLI:
		return azidentity.NewAzureDeveloperCLICredential(&azidentity.AzureDeveloperCLICredentialOptions{
			TenantID: c.TenantID,
		})
	case AzureAuthMethodDefault:
		return azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{
			TenantID: c.TenantID,
		})
	case AzureAuthMethodManagedIdentity:
		opts := &azidentity.ManagedIdentityCredentialOptions{}
		if c.ClientID != "" {
			opts.ID = azidentity.ClientID(c.ClientID)
		}
		return azidentity.NewManagedIdentityCredential(opts)
	case AzureAuthMethodEnvironment:
		return azidentity.NewEnvironmentCredential(nil)
	default:
		return nil, c.Validate()
	}
}

// TokenProvider returns a caching token source. defaultScope is used whenever
// the caller does not know the scope up front and the config does not pin one.
func (c *AzureAuthConfig) TokenProvider(defaultScope string) (*azureTokenProvider, error) {
	cred, err := c.Credential()
	if err != nil {
		return nil, fmt.Errorf("cannot create azure credential, %w", err)
	}
	if c.Scope != "" {
		defaultScope = c.Scope
	}
	return &azureTokenProvider{
		cred:         cred,
		defaultScope: defaultScope,
		pinnedScope:  c.Scope != "",
		tenantID:     c.TenantID,
		tokens:       map[string]azcore.AccessToken{},
	}, nil
}

// azureTokenProvider hands out access tokens, caching each scope's token until
// it is close to expiring. Connections are opened rarely enough that holding
// the lock across the token request is cheaper than letting a burst of new
// connections each ask Azure for their own token.
type azureTokenProvider struct {
	cred         azcore.TokenCredential
	defaultScope string
	pinnedScope  bool
	tenantID     string

	mu     sync.Mutex
	tokens map[string]azcore.AccessToken
}

// Token returns an access token for scope. An empty scope, or any scope at all
// when the connection config pins one, falls back to the default scope.
func (p *azureTokenProvider) Token(ctx context.Context, scope string) (string, error) {
	if scope == "" || p.pinnedScope {
		scope = p.defaultScope
	}
	if scope == "" {
		return "", errors.New("cannot request azure token without a scope")
	}
	if !strings.HasSuffix(scope, azureScopeSuffix) {
		scope += azureScopeSuffix
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if token, ok := p.tokens[scope]; ok && time.Until(token.ExpiresOn) > azureTokenExpiryMargin {
		return token.Token, nil
	}

	token, err := p.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes:   []string{scope},
		TenantID: p.tenantID,
	})
	if err != nil {
		return "", fmt.Errorf("cannot acquire azure access token for %s, %w", scope, err)
	}
	p.tokens[scope] = token
	return token.Token, nil
}
