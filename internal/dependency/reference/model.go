package reference

// Reference identifies one direct dependency and its source field.
type Reference struct {
	SourceKind string
	TargetKind string
	Namespace  string
	Name       string
	FieldPath  string
	Key        string
	Optional   bool
}
