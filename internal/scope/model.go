package scope

import (
	"database/sql/driver"
	"fmt"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/humaschema"
)

type TargetLabelEffect string

const (
	TargetLabelInclude TargetLabelEffect = "include"
	TargetLabelExclude TargetLabelEffect = "exclude"
)

var TargetLabelEffectValues = []TargetLabelEffect{
	TargetLabelInclude,
	TargetLabelExclude,
}

type TargetLabel struct {
	LabelID int64             `json:"label_id"`
	Effect  TargetLabelEffect `json:"effect"`
}

func (TargetLabelEffect) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(TargetLabelEffectValues...)
}

func (e *TargetLabelEffect) Scan(src any) error {
	switch value := src.(type) {
	case string:
		*e = TargetLabelEffect(value)
	case []byte:
		*e = TargetLabelEffect(value)
	default:
		return fmt.Errorf("scope: unsupported target label effect scan type %T", src)
	}
	return nil
}

func (e TargetLabelEffect) Value() (driver.Value, error) {
	return string(e), nil
}

func ValidTargetLabelEffect(effect TargetLabelEffect) bool {
	switch effect {
	case TargetLabelInclude, TargetLabelExclude:
		return true
	default:
		return false
	}
}
