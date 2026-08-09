package targeting

// LabelSet is a plain include/exclude target set.
type LabelSet struct {
	Include []LabelRef
	Exclude []LabelRef
}

func EmptyLabelSet() LabelSet {
	return LabelSet{
		Include: []LabelRef{},
		Exclude: []LabelRef{},
	}
}

func NormalizeLabelSet(targets LabelSet) LabelSet {
	if targets.Include == nil {
		targets.Include = []LabelRef{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []LabelRef{}
	}
	return targets
}
