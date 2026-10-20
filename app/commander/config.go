package commander

// Config contains the API listener settings and native service configurations.
// Service configs are passed directly to their registered constructors.
type Config struct {
	Tag     string
	Listen  string
	Service []any
}

// ReflectionConfig enables gRPC server reflection.
type ReflectionConfig struct{}
