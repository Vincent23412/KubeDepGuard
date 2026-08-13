package resolver

// Target identifies the resource a source field points to.
type Target struct{ Kind, Namespace, Name string }

// TargetResolver hides the backing store used to find a dependency target.
type TargetResolver interface{ Exists(Target) (bool, error) }
