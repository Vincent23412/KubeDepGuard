package resolver

// Query is the read-only cluster state available to dependency rules.
// Implementations may use informer caches, direct API calls, or test fakes.
type Query interface {
	List(ResourceScope) ([]Resource, error)
}
