// Package activity records the small, curated stream of durable Woodstar actions.
package activity

import (
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/openapischema"
)

// Area identifies the capability that owns an activity.
type Area string

const (
	AreaHosts   Area = "hosts"
	AreaOsquery Area = "osquery"
)

var areaValues = []Area{AreaHosts, AreaOsquery}

// Schema documents the accepted activity areas.
func (Area) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(areaValues...)
}

// ActorKind distinguishes administrator actions from Woodstar actions.
type ActorKind string

const (
	ActorKindUser   ActorKind = "user"
	ActorKindSystem ActorKind = "system"
)

var actorKindValues = []ActorKind{ActorKindUser, ActorKindSystem}

// Schema documents activity actor kinds.
func (ActorKind) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(actorKindValues...)
}

// Action is a stable activity type rendered by the frontend.
type Action string

const (
	ActionOrbitHostEnrolled          Action = "orbit_host_enrolled"
	ActionOsqueryHostEnrolled        Action = "osquery_host_enrolled"
	ActionHostDeleted                Action = "host_deleted"
	ActionHostsDeleted               Action = "hosts_deleted"
	ActionHostInventoryRequested     Action = "host_inventory_requested"
	ActionHostPrimaryUserSet         Action = "host_primary_user_set"
	ActionHostPrimaryUserCleared     Action = "host_primary_user_cleared"
	ActionPolicyCreated              Action = "policy_created"
	ActionPolicyUpdated              Action = "policy_updated"
	ActionPolicyDeleted              Action = "policy_deleted"
	ActionPoliciesDeleted            Action = "policies_deleted"
	ActionPolicyRemediationRequested Action = "policy_remediation_requested"
	ActionReportCreated              Action = "report_created"
	ActionReportUpdated              Action = "report_updated"
	ActionReportDeleted              Action = "report_deleted"
	ActionReportsDeleted             Action = "reports_deleted"
	ActionLiveQueryStarted           Action = "live_query_started"
	ActionLiveQueryStopped           Action = "live_query_stopped"
)

var actionValues = []Action{
	ActionOrbitHostEnrolled,
	ActionOsqueryHostEnrolled,
	ActionHostDeleted,
	ActionHostsDeleted,
	ActionHostInventoryRequested,
	ActionHostPrimaryUserSet,
	ActionHostPrimaryUserCleared,
	ActionPolicyCreated,
	ActionPolicyUpdated,
	ActionPolicyDeleted,
	ActionPoliciesDeleted,
	ActionPolicyRemediationRequested,
	ActionReportCreated,
	ActionReportUpdated,
	ActionReportDeleted,
	ActionReportsDeleted,
	ActionLiveQueryStarted,
	ActionLiveQueryStopped,
}

// Schema documents activity actions currently emitted by Woodstar.
func (Action) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(actionValues...)
}

// Actor is the user or Woodstar process responsible for an activity.
type Actor struct {
	Kind   ActorKind `json:"kind"`
	UserID *int64    `json:"user_id,omitempty"`
	Name   string    `json:"name"`
	Email  string    `json:"email,omitempty" format:"email"`
}

// Subject identifies the resource affected by an activity.
type Subject struct {
	Type string `json:"type"`
	ID   *int64 `json:"id,omitempty"`
	Name string `json:"name"`
}

// ActivityEvent is one durable activity projection.
type ActivityEvent struct {
	ID         int64     `json:"id"`
	Area       Area      `json:"area"`
	Action     Action    `json:"action"`
	Actor      Actor     `json:"actor"`
	Subject    Subject   `json:"subject"`
	OccurredAt time.Time `json:"occurred_at"`
}

// NewEvent is the activity data supplied by a producer.
type NewEvent struct {
	Area    Area
	Action  Action
	Actor   Actor
	Subject Subject
}

// ListParams filters the activity timeline.
type ListParams struct {
	ListParams listing.Params
	Area       Area
}

func (params *ListParams) normalize() {
	params.ListParams = listing.Normalize(params.ListParams)
}

func (params *ListParams) validate() error {
	if err := listing.Validate(params.ListParams); err != nil {
		return err
	}
	switch params.Area {
	case "", AreaHosts, AreaOsquery:
	default:
		return fmt.Errorf("%w: unknown activity area %q", fault.ErrInvalidInput, params.Area)
	}
	return nil
}
