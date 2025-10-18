package hazycheck

import (
	"context"

	"github.com/HazyCorp/govnilo/internal/registrar"

	"go.uber.org/fx"
)

// RegisterSploit registers a sploit to be run by checker.
// If sploit is not registered by this method, it will not be accessible in runtime.
// To register a sploit, you need to provide it's constructor function. All arguments of the constructor
// will be fullfilled by the infrastructure. You can, for example, ask *slog.Logger to be provided to your constructor.
func RegisterSploit(constructor interface{}) {
	registrar.Register(
		fx.Annotate(
			constructor,
			fx.As(new(Sploit)),
			fx.ResultTags(`group:"sploits"`),
		),
	)
}

type SploitID struct {
	Service string `json:"service"`
	Name    string `json:"name"`
}

// Sploit is an interface, wich needs to be implemented by any exploit of the service.
type Sploit interface {
	// RunAttack must run an attack on the service, located at target.
	// RunAttack will be called by the infrastructure code, so you don't need to run any background jobs
	// yourself.
	// The context contains a trace ID for debugging: use govnilo.GetTraceID(ctx) to retrieve it.
	RunAttack(ctx context.Context, target string) error

	// SploitID must return the service name, which this sploit is responsible for
	// and the name of the sploit itself, e.g.
	// SploitID{Service: "example", Name: "race_condition_drop_database"}.
	// This method needed by infrastructure code to identify the sploit.
	SploitID() SploitID
}
