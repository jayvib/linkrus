package oauth_test

import (
	"golang.org/x/oauth2"
	gc "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

var _ = gc.Suite(new(AuthHandlerTestSuite))

func Test(t *testing.T) {
	gc.TestingT(t)
}

type AuthHandlerTestSuite struct {
	srv        *httptest.Server
	srvHandler http.HandlerFunc

	authHandler *oauth.Flow
}

func (s *AuthHandlerTestSuite) SetUpTest(c *gc.C) {
	// Create a dummy server
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.srvHandler != nil {
			s.srvHandler(w, r)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))

	// Initialize auth flow
	ah, err := oauth.NewOAuthFlow(oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.srv.URL + "/oauth/authorize",
			TokenURL: s.srv.URL + "/oauth/access_token",
		},
	}, "localhost:0", "")

	c.Assert(err, gc.IsNil)

	s.authHandler = ah
}
