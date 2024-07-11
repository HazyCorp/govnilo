package checker

import (
	"context"

	"go.uber.org/fx"

	"github.com/HazyCorp/checker/internal/registrar"
)

// RegisterChecker registers constructor of the checker with needed annotations.
// If you need to register checker, use THIS function.
// If you need to register some dependencies for your checker, use registrar.RegisterChecker
func RegisterChecker(constructor interface{}) {
	registrar.Register(
		fx.Annotate(
			constructor,
			fx.As(new(Checker)),
			fx.ResultTags(`group:"checkers"`),
		),
	)
}

// SLA is a struct, that is used to estimate how well service is behaving.
// NOT FINAL
// using struct to be able to provide more information than single float
type SLA struct {
	TotalAttempts       int
	SuccessfullAttempts int
}

// Checker is an interface, which must be implemented by every keep-alive checker.
// Checker will be called like this: initially Checker.Check will be called. Check method
// will be called very frequently. It need's to be concurrently safe.
//
// In most of the cases, Checker.Check must create some data in the service, and check, that that data
// persists in the service. Example: create user and try to register with provided credentials.
//
// []byte array returned from Checker.Check is data, specific to this Checker.Check call. Example: login and password
// of the created user. This data will be used in future Checker.Get calls.
//
// Checker.Get is a method, that will be called periodically, not so frequently, compared to Checker.Check.
// This method must check, that service didn't drop data of the corresponding Checker.Check call.
// Not all the Checker.Check calls will be verified, only some randomly chosen. But checker author doesn't need
// to think about it. Data, provided to Checker.Get call is exactly the same, that was returned from Checker.Check.
//
// All checkers must return name of the service, this checker was written for. This name will be used in configs.
type Checker interface {
	// Check must run most common flow of your service, to check, that all of it's components are working.
	// If you have multiple flows to check, you need to run these checks concurrently and wait untill their end.
	// You may use sync.WaitGroup to achieve that result, or event errgroup.Group, if you want to.
	Check(ctx context.Context, target string) ([]byte, error)
	// Get must verify, that data, created in Check is still exists in service. In other words: Get checks service's
	// consistency over time.
	Get(ctx context.Context, target string, data []byte) error

	// Name of the service. This method is used for internal checker registration.
	ServiceName() string
}
