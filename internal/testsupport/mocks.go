package testsupport

import (
	"context"
	"errors"

	"github.com/stretchr/testify/mock"

	"github.com/dadrus/heimdall/internal/heimdall"
	"github.com/dadrus/heimdall/internal/pipeline/handler"
	"github.com/dadrus/heimdall/internal/pipeline/handler/subject"
)

var ErrTestPurpose = errors.New("error raised in a test")

type MockSubjectExtractor struct {
	mock.Mock
}

func (m *MockSubjectExtractor) GetSubject(data []byte) (*subject.Subject, error) {
	args := m.Called(data)

	if val := args.Get(0); val != nil {
		res, ok := val.(*subject.Subject)
		if !ok {
			panic("*heimdal.Subject expected")
		}

		return res, args.Error(1)
	}

	return nil, args.Error(1)
}

type MockAuthenticator struct {
	mock.Mock
}

func (a *MockAuthenticator) Authenticate(ctx heimdall.Context) (*subject.Subject, error) {
	args := a.Called(ctx)

	if val := args.Get(0); val != nil {
		res, ok := val.(*subject.Subject)
		if !ok {
			panic("*subject.Subject expected")
		}

		return res, args.Error(1)
	}

	return nil, args.Error(1)
}

func (a *MockAuthenticator) WithConfig(c map[string]interface{}) (handler.Authenticator, error) {
	args := a.Called(c)

	if val := args.Get(0); val != nil {
		res, ok := val.(handler.Authenticator)
		if !ok {
			panic("handler.Authenticator expected")
		}

		return res, args.Error(1)
	}

	return nil, args.Error(1)
}

type MockContext struct {
	mock.Mock
}

func (m *MockContext) RequestHeader(name string) string {
	args := m.Called(name)

	return args.String(0)
}

func (m *MockContext) RequestCookie(name string) string {
	args := m.Called(name)

	return args.String(0)
}

func (m *MockContext) RequestQueryParameter(name string) string {
	args := m.Called(name)

	return args.String(0)
}

func (m *MockContext) RequestFormParameter(name string) string {
	args := m.Called(name)

	return args.String(0)
}

func (m *MockContext) RequestBody() []byte {
	args := m.Called()

	if i := args.Get(0); i != nil {
		val, ok := i.([]byte)
		if !ok {
			panic("[]byte expected")
		}

		return val
	}

	return nil
}

func (m *MockContext) AppContext() context.Context {
	args := m.Called()

	if i := args.Get(0); i != nil {
		val, ok := i.(context.Context)
		if !ok {
			panic("context.Context")
		}

		return val
	}

	return nil
}

func (m *MockContext) SetPipelineError(err error) {
	m.Called(err)
}

func (m *MockContext) AddResponseHeader(name, value string) {
	m.Called(name, value)
}

func (m *MockContext) SetSubject(sub *subject.Subject) {
	m.Called(sub)
}

func (m *MockContext) Subject() *subject.Subject {
	args := m.Called()

	if i := args.Get(0); i != nil {
		val, ok := i.(*subject.Subject)
		if !ok {
			panic("*heimdall.Subject")
		}

		return val
	}

	return nil
}

type MockClaimAsserter struct {
	mock.Mock
}

func (a *MockClaimAsserter) AssertIssuer(issuer string) error {
	args := a.Called(issuer)

	return args.Error(0)
}

func (a *MockClaimAsserter) AssertAudience(audience []string) error {
	args := a.Called(audience)

	return args.Error(0)
}

func (a *MockClaimAsserter) AssertScopes(scopes []string) error {
	args := a.Called(scopes)

	return args.Error(0)
}

func (a *MockClaimAsserter) AssertValidity(nbf, exp int64) error {
	args := a.Called(nbf, exp)

	return args.Error(0)
}

func (a *MockClaimAsserter) IsAlgorithmAllowed(alg string) bool {
	args := a.Called(alg)

	return args.Bool(0)
}
