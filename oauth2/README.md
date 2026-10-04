# OAuth2.0 Module

Talks to OAuth2.0 / OIDC providers: builds the login URL, exchanges the code, fetches userinfo, revokes tokens

## Packages

| Package | Purpose |
|---|---|
| `oauthclient` | Login flow: `AuthURL`, `Exchange`, `GetUser`, `Revoke`, with PKCE |
| `providers` | Per-provider userinfo and revoke implementations |

## Providers

| Name | Created with |
|---|---|
| `discord`, `github`, `google` | `oauthclient.New`, built-in endpoints |
| any other name (generic) | `oauthclient.New`, requires `AuthURL`, `TokenURL`, `UserinfoURL` |
| OIDC | `oauthclient.NewOIDC`, endpoints from `{issuer}/.well-known/openid-configuration` |

## Email Verification

- No email from the provider: `ErrMissingEmail`
- Unverified email: `ErrUnverifiedEmail`; relax with `providers.WithAllowUnverifiedEmail(true)`

## Testing

```bash
go test ./...
```
