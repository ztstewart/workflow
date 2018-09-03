package workflow

const (
	_invalidGraphMsg = "dependency graph is unsolvable; check for cycles or missing dependencies"
)

// An InvalidGraphError is use to mark a dependency graph as being invalid.
// It is returned when a graph has unmet dependencies or cycles.
type InvalidGraphError struct{}

func (ige InvalidGraphError) Error() string {
	return _invalidGraphMsg
}
