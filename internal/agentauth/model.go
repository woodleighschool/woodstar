package agentauth

import (
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/humaschema"
)

type Agent string

const (
	AgentOrbit Agent = "orbit"
	AgentSanta Agent = "santa"
)

var AgentValues = []Agent{AgentOrbit, AgentSanta}

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

func (Agent) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(AgentValues...)
}

func ParseAgent(value string) (Agent, error) {
	switch Agent(value) {
	case AgentOrbit:
		return AgentOrbit, nil
	case AgentSanta:
		return AgentSanta, nil
	default:
		return "", ErrInvalidAgent
	}
}

func (a Agent) Valid() bool {
	switch a {
	case AgentOrbit, AgentSanta:
		return true
	default:
		return false
	}
}
