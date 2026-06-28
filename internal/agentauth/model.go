package agentauth

import (
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/schema"
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
	Agent Agent  `json:"agent"`
	Value string `json:"value" minLength:"32"`
}

type AgentSecretMutation struct {
	Value string `json:"value" minLength:"32"`
}

func (Agent) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(AgentValues...)
}

func (a Agent) Valid() bool {
	switch a {
	case AgentOrbit, AgentMunki, AgentSanta:
		return true
	default:
		return false
	}
}
