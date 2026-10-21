---
id: how-to-oidc-control-deck
type: how-to
status: stable
tags: [auth, oidc, how-to]
---

# OIDC sign-in for the Control Deck

## Goal

The dedicated Control Deck can use an OpenID Connect provider for browser sign-in. OIDC protects only the Control Deck listener; it does not add authentication to tuner, stream, or media-server endpoints.

## Preconditions

- Publish the Control Deck behind HTTPS and know its external base URL.
- Create a confidential web client in the identity provider.
- Decide which verified email addresses or domains may sign in. The default
  policy denies users who are not explicitly admitted.

## Configure the provider

Set these server-side environment variables:

```dotenv
IPTV_TUNERR_WEBUI_OIDC_ISSUER_URL=https://id.example.com/realms/media
IPTV_TUNERR_WEBUI_OIDC_CLIENT_ID=iptvtunerr
IPTV_TUNERR_WEBUI_OIDC_CLIENT_SECRET=replace-with-the-client-secret
IPTV_TUNERR_WEBUI_OIDC_REDIRECT_URI=https://tunerr.example.com/auth/oidc/callback
IPTV_TUNERR_WEBUI_OIDC_ALLOWED_EMAILS=operator@example.com
IPTV_TUNERR_WEBUI_OIDC_PROVIDER=authentik
```

Register the exact redirect URI with the identity provider. For ordinary providers, the client must be a confidential web client that supports authorization code flow and PKCE. The app requests `openid profile email`, validates the ID token and nonce on the server, and requires a verified email admitted by the exact-address or domain allow-list. Any verified user is denied unless `IPTV_TUNERR_WEBUI_OIDC_ALLOW_ANY_VERIFIED_USER=true` is explicitly set.

The OIDC callback creates the same cookie-backed Control Deck session as local password sign-in. Session-bound CSRF checks still apply to state-changing browser requests. Existing password sign-in and HTTP Basic access for scripts remain available. The app does not persist provider tokens or map provider groups to privileges.

HTTPS is required for the issuer and redirect URI by default. For an intentionally private identity provider, set `IPTV_TUNERR_WEBUI_OIDC_ALLOW_PRIVATE_ISSUER=true`; this permits private-network endpoints, including HTTP. Keep that setting off for public deployments.

`IPTV_TUNERR_WEBUI_OIDC_PROVIDER` is an optional display hint. The login page
shows a local provider icon and a direct official setup link for `google`,
`microsoft-entra`, `okta`, `auth0`, `keycloak`, `authentik`, `zitadel`,
`amazon-cognito`, `gitlab`, `fusionauth`, `onelogin`, `pingone`,
`jumpcloud`, and `apple-sign-in`. Unknown values use generic OIDC presentation;
the configured issuer remains the authority for discovery and validation.

The [shared provider catalog](https://github.com/snapetech-org/seerrng/blob/main/docs/maintainers/oidc-provider-catalog.json)
lists each provider's example issuer, scopes, official setup link, and tenant
or claim notes. It covers Google, Microsoft Entra ID, Okta, Auth0, Keycloak,
authentik, ZITADEL, Amazon Cognito, GitLab, FusionAuth, OneLogin, PingOne,
JumpCloud, and Apple. The [cross-project OIDC contract](https://github.com/snapetech-org/seerrng/blob/main/docs/maintainers/oidc-cross-project-template.md)
defines the common flow and security rules.

### Sign in with Apple

Set the issuer to exactly `https://appleid.apple.com`, use the Apple Services ID
as `IPTV_TUNERR_WEBUI_OIDC_CLIENT_ID`, and set
`IPTV_TUNERR_WEBUI_OIDC_CLIENT_SECRET` to the signed client-secret JWT. Register
`https://<public-host>/auth/oidc/callback` as the exact HTTPS return URL. The
adapter requests only `openid email`, uses `response_mode=form_post`, and omits
PKCE for Apple's documented web flow. Apple may encode `email_verified` as a
boolean or string; only `true` is admitted. Rotate the JWT before its six-month
maximum expiry. See Apple's [environment setup](https://developer.apple.com/documentation/signinwithapple/configuring-your-environment-for-sign-in-with-apple)
and [web authorization flow](https://developer.apple.com/documentation/signinwithapple/incorporating-sign-in-with-apple-into-other-platforms).

## Verify

1. Confirm the Control Deck login page shows the configured provider label and
   local mark, and that its setup link opens the provider's official guide.
2. Sign in with an explicitly allowed account and confirm the browser receives
   the normal Control Deck session cookie.
3. Confirm an unverified or unadmitted account is denied and state-changing
   requests still pass the session-bound CSRF checks.
4. Confirm tuner, stream, and media-server endpoints retain their existing
   authentication behavior.

Automated tests cover the OIDC handler and provider presentation. A real
provider sign-in still requires an operator-owned test tenant and registered
callback; no live identity-provider credentials are included in this repo.

## Rollback

Remove the `IPTV_TUNERR_WEBUI_OIDC_*` variables from the service environment
and restart IPTV Tunerr. Local Control Deck password sign-in and HTTP Basic
access remain available.

## Troubleshooting

- Check that the configured issuer exactly matches the `issuer` in the
  provider's discovery document.
- Check that the provider has the exact HTTPS redirect URI registered.
- Check that the ID token includes `email` and affirmative `email_verified`
  claims, and that the normalized address or domain is admitted.
- For an intentionally private issuer, set
  `IPTV_TUNERR_WEBUI_OIDC_ALLOW_PRIVATE_ISSUER=true` only for the trusted
  deployment.

See also
--------
- [How-to index](index.md) · [Docs index](../index.md) · [Feature map](../features.md).
- [Shared provider catalog](https://github.com/snapetech-org/seerrng/blob/main/docs/maintainers/oidc-provider-catalog.json).
- [Cross-project OIDC contract](https://github.com/snapetech-org/seerrng/blob/main/docs/maintainers/oidc-cross-project-template.md).
