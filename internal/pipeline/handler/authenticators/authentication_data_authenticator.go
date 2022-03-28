package authenticators

import (
	"context"
	"errors"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/dadrus/heimdall/internal/errorsx"
	"github.com/dadrus/heimdall/internal/heimdall"
	"github.com/dadrus/heimdall/internal/pipeline/endpoint"
	"github.com/dadrus/heimdall/internal/pipeline/handler"
)

type authenticationDataAuthenticator struct {
	Endpoint         Endpoint
	SubjectExtractor SubjectExtrator
	AuthDataGetter   AuthDataGetter
}

func NewAuthenticationDataAuthenticatorFromYAML(rawConfig []byte) (*authenticationDataAuthenticator, error) {
	type _config struct {
		Endpoint       endpoint.Endpoint        `yaml:"identity_info_endpoint"`
		AuthDataSource AuthenticationDataSource `yaml:"authentication_data_source"`
		Session        Session                  `yaml:"session"`
	}

	var c _config
	if err := yaml.Unmarshal(rawConfig, &c); err != nil {
		return nil, &errorsx.ArgumentError{
			Message: "failed to unmarshal config",
			Cause:   err,
		}
	}

	return &authenticationDataAuthenticator{
		Endpoint:         c.Endpoint,
		AuthDataGetter:   c.AuthDataSource.Strategy(),
		SubjectExtractor: &c.Session,
	}, nil
}

func (a *authenticationDataAuthenticator) Authenticate(ctx context.Context, as handler.RequestContext, sc *heimdall.SubjectContext) error {
	authDataRef, err := a.AuthDataGetter.GetAuthData(as)
	if err != nil {
		return &errorsx.ArgumentError{Message: "failed to extract authentication data", Cause: err}
	}

	rawBody, err := a.Endpoint.SendRequest(ctx, strings.NewReader(authDataRef))
	if err != nil {
		return err
	}

	if sc.Subject, err = a.SubjectExtractor.GetSubject(rawBody); err != nil {
		return err
	}

	return nil
}

func (a *authenticationDataAuthenticator) WithConfig(_ []byte) (handler.Authenticator, error) {
	// this authenticator does not allow configuration from a rule
	return nil, errors.New("reconfiguration not allowed")
}
