package authenticators

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/dadrus/heimdall/internal/heimdall"
	"github.com/dadrus/heimdall/internal/x/stringsx"
)

type Session struct {
	SubjectFrom    string `yaml:"subject_from"`
	AttributesFrom string `yaml:"attributes_from"`
}

func (s *Session) GetSubject(rawData json.RawMessage) (*heimdall.Subject, error) {
	var (
		subjectId  string
		attributes map[string]interface{}
	)

	rawSubjectId := []byte(stringsx.Coalesce(gjson.GetBytes(rawData, s.SubjectFrom).Raw, "null"))
	if err := json.Unmarshal(rawSubjectId, &subjectId); err != nil {
		return nil, fmt.Errorf("configured subject_from GJSON path returned an error on JSON output: %w", err)
	}

	rawAttributes := []byte(stringsx.Coalesce(gjson.GetBytes(rawData, s.AttributesFrom).Raw, "null"))
	if err := json.Unmarshal(rawAttributes, &attributes); err != nil {
		return nil, fmt.Errorf("configured attributes_from GJSON path returned an error on JSON output: %w", err)
	}

	return &heimdall.Subject{
		Id:         subjectId,
		Attributes: attributes,
	}, nil
}
