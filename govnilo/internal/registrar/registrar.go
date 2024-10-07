package registrar

var constructors []interface{}

func GetRegistered() []interface{} {
	return constructors
}

// Register function must be called in package init function to register needed constructors.
// Primarly used to register checkers.
func Register(constructor interface{}) {
	constructors = append(constructors, constructor)
}
