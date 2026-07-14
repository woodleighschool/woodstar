package agentauth

import (
	"errors"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

type Agent string

const (
	AgentOrbit Agent = "orbit"
	AgentMunki Agent = "munki"
	AgentSanta Agent = "santa"
)

var AgentValues = []Agent{AgentOrbit, AgentMunki, AgentSanta}

var (
	ErrInvalidAgent  = errors.New("invalid agent")
	ErrInvalidSecret = errors.New("invalid agent secret")
)

type AgentSecret struct {
	ID        int64     `json:"id"`
	Agent     Agent     `json:"agent"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentSecretCreate struct {
	Agent Agent  `json:"agent" validate:"required,oneof=orbit munki santa"`
	Value string `json:"value" validate:"required,notblank,min=32"         minLength:"32"`
}

type AgentSecretMutation struct {
	Value string `json:"value" minLength:"32" validate:"required,notblank,min=32"`
}

func (params *AgentSecretCreate) normalize() {
	params.Agent = Agent(strings.TrimSpace(string(params.Agent)))
}

func (params *AgentSecretCreate) validate() error {
	if !params.Agent.Valid() {
		return ErrInvalidAgent
	}
	if err := validation.Struct(params); err != nil {
		return ErrInvalidSecret
	}
	return nil
}

func (params AgentSecretMutation) validate() error {
	if err := validation.Struct(params); err != nil {
		return ErrInvalidSecret
	}
	return nil
}

func (Agent) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(AgentValues...)
}

func (a Agent) Valid() bool {
	switch a {
	case AgentOrbit, AgentMunki, AgentSanta:
		return true
	default:
		return false
	}
}
