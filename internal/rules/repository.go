package rules

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"

	"github.com/dadrus/heimdall/internal/config"
	"github.com/dadrus/heimdall/internal/heimdall"
	"github.com/dadrus/heimdall/internal/pipeline"
	"github.com/dadrus/heimdall/internal/pipeline/handler"
	"github.com/dadrus/heimdall/internal/rules/provider"
)

var ErrNoRuleFound = errors.New("no rule found")

type Repository interface {
	FindRule(*url.URL) (Rule, error)
}

func NewRepository(
	queue provider.RuleSetChangedEventQueue,
	config config.Configuration,
	hf pipeline.HandlerFactory,
	logger zerolog.Logger,
) (Repository, error) {
	return &repository{
		hf:     hf,
		dpc:    config.Rules.Default,
		logger: logger,
		queue:  queue,
		quit:   make(chan bool),
	}, nil
}

type repository struct {
	hf     pipeline.HandlerFactory
	dpc    config.Pipeline
	logger zerolog.Logger

	rules []*rule
	mutex sync.RWMutex

	queue provider.RuleSetChangedEventQueue
	quit  chan bool
}

func (r *repository) FindRule(requestURL *url.URL) (Rule, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, rule := range r.rules {
		if rule.MatchesURL(requestURL) {
			return rule, nil
		}
	}

	return nil, ErrNoRuleFound
}

func (r *repository) Start() error {
	r.logger.Info().Msg("Starting rule definition loader")

	go r.watchRuleSetChanges()

	return nil
}

func (r *repository) Stop() error {
	r.logger.Info().Msg("Tearing down rule definition loader")

	r.quit <- true

	close(r.queue)

	return nil
}

func (r *repository) watchRuleSetChanges() {
	for {
		select {
		case evt, ok := <-r.queue:
			if !ok {
				r.logger.Debug().Msg("Rule set definition queue closed")
			}

			if evt.ChangeType == provider.Create {
				r.onRuleSetCreated(evt.Src, evt.Definition)
			} else if evt.ChangeType == provider.Remove {
				r.onRuleSetDeleted(evt.Src)
			}
		case <-r.quit:
			r.logger.Info().Msg("Rule definition loader stopped")

			return
		}
	}
}

func (r *repository) loadRules(srcID string, definition json.RawMessage) ([]*rule, error) {
	rcs, err := parseRuleSetFromYaml(definition)
	if err != nil {
		return nil, err
	}

	rules := make([]*rule, len(rcs))

	for idx, rc := range rcs {
		rule, err := r.newRule(srcID, rc)
		if err != nil {
			return nil, err
		}

		rules[idx] = rule
	}

	return rules, nil
}

func (r *repository) addRule(rule *rule) {
	r.mutex.Lock()
	r.rules = append(r.rules, rule)
	r.mutex.Unlock()

	r.logger.Debug().Str("src", rule.srcID).Str("id", rule.id).Msg("Rule added")
}

func (r *repository) removeRules(srcID string) {
	r.logger.Info().Str("src", srcID).Msg("Removing rules")

	// TODO: implement remove rule
}

func (r *repository) onRuleSetCreated(srcID string, definition json.RawMessage) {
	// create rules
	r.logger.Info().Str("src", srcID).Msg("Loading rule set")

	rules, err := r.loadRules(srcID, definition)
	if err != nil {
		r.logger.Error().Err(err).Str("src", srcID).Msg("Failed loading rule set")
	}

	// add them
	for _, rule := range rules {
		r.addRule(rule)
	}
}

func (r *repository) onRuleSetDeleted(src string) {
	r.removeRules(src)
}

func (r *repository) newRule(srcID string, ruleConfig config.RuleConfig) (*rule, error) {
	authenticator, err := r.hf.CreateAuthenticator(ruleConfig.Authenticators)
	if err != nil {
		return nil, err
	}

	authorizer, err := r.hf.CreateAuthorizer(ruleConfig.Authorizer)
	if err != nil {
		return nil, err
	}

	hydrator, err := r.hf.CreateHydrator(ruleConfig.Hydrators)
	if err != nil {
		return nil, err
	}

	mutator, err := r.hf.CreateMutator(ruleConfig.Mutators)
	if err != nil {
		return nil, err
	}

	errorHandler, err := r.hf.CreateErrorHandler(ruleConfig.ErrorHandlers)
	if err != nil {
		return nil, err
	}

	return &rule{
		id:      ruleConfig.ID,
		url:     ruleConfig.URL,
		methods: ruleConfig.Methods,
		srcID:   srcID,
		an:      authenticator,
		az:      authorizer,
		h:       hydrator,
		m:       mutator,
		eh:      errorHandler,
	}, nil
}

func parseRuleSetFromYaml(data []byte) ([]config.RuleConfig, error) {
	// parser := koanf.new(".")
	//
	// err := parser.load(rawbytes.provider(data), yaml.parser())
	// if err != nil {
	// 	return nil, fmt.errorf("failed to read config: %w", err)
	// }

	var rcs []config.RuleConfig

	if err := yaml.UnmarshalStrict(data, &rcs); err != nil {
		return nil, err
	}

	return rcs, nil
}

type rule struct {
	id      string
	url     string
	methods []string
	srcID   string
	an      handler.Authenticator
	az      handler.Authorizer
	h       handler.Hydrator
	m       handler.Mutator
	eh      handler.ErrorHandler
}

func (r *rule) Execute(ctx context.Context, reqCtx handler.RequestContext) (*heimdall.SubjectContext, error) {
	logger := zerolog.Ctx(ctx)

	subjectCtx := &heimdall.SubjectContext{}

	if err := r.an.Authenticate(ctx, reqCtx, subjectCtx); err != nil {
		logger.Info().Err(err).Msg("Authentication failed")

		return nil, r.eh.HandleError(ctx, err)
	}

	if err := r.az.Authorize(ctx, reqCtx, subjectCtx); err != nil {
		logger.Info().Err(err).Msg("Authorization failed")

		return nil, r.eh.HandleError(ctx, err)
	}

	if err := r.h.Hydrate(ctx, subjectCtx); err != nil {
		logger.Info().Err(err).Msg("Hydration failed")

		return nil, r.eh.HandleError(ctx, err)
	}

	if err := r.m.Mutate(ctx, subjectCtx); err != nil {
		logger.Info().Err(err).Msg("Mutation failed")

		return nil, r.eh.HandleError(ctx, err)
	}

	return subjectCtx, nil
}

func (r *rule) MatchesURL(requestURL *url.URL) bool {
	return true
}

func (r *rule) MatchesMethod(method string) bool {
	return slices.Contains(r.methods, method)
}

func (r *rule) ID() string {
	return r.id
}
