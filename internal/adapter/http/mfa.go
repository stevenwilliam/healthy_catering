package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
)

// registerMFA mounts second-factor enrolment and the login challenge step.
func registerMFA(g *gin.RouterGroup, d Deps) {
	if d.MFA == nil {
		return
	}

	limited := g.Group("", RateLimit(d.Limiter, "auth", 10))

	// The challenge step is rate-limited like login itself: it is a six-digit
	// secret, and without a limiter a million guesses is an afternoon's work.
	limited.POST("/auth/mfa", func(c *gin.Context) {
		var body struct {
			MFAToken string `json:"mfa_token"`
			Code     string `json:"code"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send mfa_token and code."))
			return
		}
		if body.MFAToken == "" || body.Code == "" {
			Fail(c, apierror.Validation("Send mfa_token and code.", nil))
			return
		}
		session, err := d.Auth.CompleteMFA(c.Request.Context(), body.MFAToken, body.Code,
			c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, session)
	})

	authed := g.Group("", RequireAuth(d.Auth, d.Signer))

	authed.GET("/me/mfa", func(c *gin.Context) {
		ident, _ := Authenticated(c)
		status, err := d.MFA.Status(c.Request.Context(), ident)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, status)
	})

	// Enrolment is deliberately two calls. The secret is useless until a code
	// generated FROM it comes back, which is the only proof the user actually
	// scanned it and can still get in tomorrow.
	authed.POST("/me/mfa/start", func(c *gin.Context) {
		ident, _ := Authenticated(c)
		out, err := d.MFA.Start(c.Request.Context(), ident, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		// Never cached: this response carries the secret itself.
		c.Header("Cache-Control", "no-store")
		OK(c, http.StatusOK, out)
	})

	authed.POST("/me/mfa/confirm", func(c *gin.Context) {
		var body struct {
			Code string `json:"code"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send the six-digit code."))
			return
		}
		ident, _ := Authenticated(c)
		out, err := d.MFA.Confirm(c.Request.Context(), ident, body.Code, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		c.Header("Cache-Control", "no-store")
		OK(c, http.StatusOK, out)
	})

	authed.POST("/me/mfa/disable", func(c *gin.Context) {
		var body struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send your password to confirm."))
			return
		}
		ident, _ := Authenticated(c)
		if err := d.MFA.Disable(c.Request.Context(), ident, body.Password, actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"enabled": false,
			"message": "Two-factor authentication is off."})
	})
}
