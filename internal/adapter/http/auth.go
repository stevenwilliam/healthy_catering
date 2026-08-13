package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

const ctxIdentity = "identity"

// Authenticated pulls the resolved caller off the context. Handlers behind
// RequireAuth can rely on it; anything else must check the second return.
func Authenticated(c *gin.Context) (app.Identity, bool) {
	v, ok := c.Get(ctxIdentity)
	if !ok {
		return app.Identity{}, false
	}
	ident, ok := v.(app.Identity)
	return ident, ok
}

// RequireAuth verifies the bearer token and resolves the caller's authorization
// FROM THE DATABASE on every request.
//
// The token proves who you are; it is never trusted for what you may do. A
// deactivated account or a revoked role takes effect on the next request rather
// than when a 15-minute token happens to expire.
func RequireAuth(auth *app.Auth, signer *security.TokenSigner) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearer(c)
		if raw == "" {
			Fail(c, apierror.Unauthorized("Please sign in."))
			return
		}
		claims, err := signer.Parse(raw)
		if err != nil {
			Fail(c, apierror.Unauthorized("Please sign in again."))
			return
		}
		// A purpose-bound token — the half-authenticated one handed out between
		// a correct password and a correct 2FA code — is NOT a session. Without
		// this check the challenge token would open every endpoint the account
		// can reach, and the second factor would be decorative.
		if claims.Purpose != "" {
			Fail(c, apierror.Unauthorized("Finish signing in first."))
			return
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			Fail(c, apierror.Unauthorized("Please sign in again."))
			return
		}
		ident, err := auth.Resolve(c.Request.Context(), userID)
		if err != nil {
			Fail(c, err)
			return
		}
		c.Set(ctxIdentity, ident)
		c.Next()
	}
}

// RequirePermission is the deny-by-default gate (99 §7).
//
// A route that declares no permission serves nobody, because it never reaches
// this middleware and there is no other way into an authenticated group. The
// permission is declared next to the route, so reading the router tells you the
// whole authorization model.
func RequirePermission(perms ...security.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		ident, ok := Authenticated(c)
		if !ok {
			Fail(c, apierror.Unauthorized("Please sign in."))
			return
		}
		if !ident.Permissions.HasAny(perms...) {
			// Deliberately identical whether the resource exists or not, and
			// carrying no hint about what permission was missing.
			Fail(c, apierror.Forbidden(apierror.CodeForbidden,
				"You do not have access to this."))
			return
		}
		c.Next()
	}
}

// RequireVerifiedEmail gates the first order behind a verified address
// (docs/03 Q-15). Browsing does not need it; ordering does.
func RequireVerifiedEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		ident, ok := Authenticated(c)
		if !ok {
			Fail(c, apierror.Unauthorized("Please sign in."))
			return
		}
		if !ident.EmailVerified {
			Fail(c, apierror.Forbidden(apierror.CodeForbidden,
				"Please confirm your email address before ordering. We sent you a link."))
			return
		}
		c.Next()
	}
}

func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// registerAuth mounts the authentication endpoints.
func registerAuth(g *gin.RouterGroup, d Deps) {
	// Rate limited hard: these are the brute-forceable endpoints (99 §7).
	limited := g.Group("", RateLimit(d.Limiter, "auth", 10))

	limited.POST("/auth/register", func(c *gin.Context) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			FullName string `json:"full_name"`
			Phone    string `json:"phone"`
			Locale   string `json:"locale"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send email, password and full_name."))
			return
		}

		res, err := d.Auth.Register(c.Request.Context(), app.RegisterInput{
			Email: body.Email, Password: body.Password, FullName: body.FullName,
			Phone: body.Phone, Locale: body.Locale,
			IP: c.ClientIP(), UA: c.Request.UserAgent(),
		})
		if err != nil {
			Fail(c, err)
			return
		}

		// The reply is IDENTICAL whether or not the address was already
		// registered. Registration must not become a way to test which
		// addresses have accounts (99 §7).
		if res.AlreadyRegistered {
			d.Log.Info("registration attempted for an existing address",
				"request_id", c.GetString(ctxRequestID))
		} else if d.OnVerificationToken != nil {
			d.OnVerificationToken(res.UserID, res.VerificationToken)
		}

		OK(c, http.StatusAccepted, gin.H{
			"message": "Check your email for a confirmation link.",
		})
	})

	limited.POST("/auth/login", func(c *gin.Context) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send email and password."))
			return
		}
		session, err := d.Auth.Login(c.Request.Context(), app.LoginInput{
			Email: body.Email, Password: body.Password,
			IP: c.ClientIP(), UA: c.Request.UserAgent(),
		})
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, session)
	})

	limited.POST("/auth/refresh", func(c *gin.Context) {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == "" {
			Fail(c, apierror.Validation("Send refresh_token.", nil))
			return
		}
		session, err := d.Auth.Refresh(c.Request.Context(), body.RefreshToken,
			c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, session)
	})

	limited.POST("/auth/verify-email", func(c *gin.Context) {
		var body struct {
			Token string `json:"token"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Token == "" {
			Fail(c, apierror.Validation("Send token.", nil))
			return
		}
		if err := d.Auth.VerifyEmail(c.Request.Context(), body.Token); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"message": "Your email address is confirmed."})
	})

	// Authenticated endpoints.
	authed := g.Group("", RequireAuth(d.Auth, d.Signer))

	authed.POST("/auth/logout", func(c *gin.Context) {
		ident, _ := Authenticated(c)
		if err := d.Auth.Logout(c.Request.Context(), ident.UserID); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"message": "Signed out."})
	})

	authed.GET("/me", func(c *gin.Context) {
		ident, _ := Authenticated(c)
		OK(c, http.StatusOK, gin.H{
			"user_id":        ident.UserID,
			"customer_id":    ident.CustomerID,
			"email":          ident.Email,
			"is_staff":       ident.IsStaff,
			"roles":          ident.Roles,
			"permissions":    ident.Permissions.Codes(),
			"kitchen_id":     ident.KitchenID,
			"email_verified": ident.EmailVerified,
		})
	})
}
