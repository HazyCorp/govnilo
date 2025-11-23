package hazycheck

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/HazyCorp/govnilo/internal/registrar"

	"go.uber.org/fx"
)

var registeredCheckers []CheckerID

func GetRegistered() []CheckerID {
	return registeredCheckers
}

// RegisterChecker registers constructor of the checker with needed annotations.
// If you need to register checker, use THIS function.
// If you need to register some dependencies for your checker, use registrar.RegisterChecker
func RegisterChecker(constructor any) {
	constr := reflect.ValueOf(constructor)
	constrType := constr.Type()

	if constrType.Kind() != reflect.Func {
		panic("not func provided to register")
	}

	if constrType.NumOut() != 1 && constrType.NumOut() != 2 {
		panic("func must return either checker or (checker, error)")
	}

	if constrType.NumOut() == 2 {
		errType := constrType.Out(1)
		if !errType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			panic("func must return either checker or (checker, error)")
		}
	}

	retType := constrType.Out(0)
	if !retType.Implements(reflect.TypeOf((*Checker)(nil)).Elem()) {
		panic("func returns not checker")
	}

	checkerID := reflect.New(retType).Elem().Interface().(Checker).CheckerID()
	registeredCheckers = append(registeredCheckers, checkerID)

	newConstr := reflect.MakeFunc(constrType, func(args []reflect.Value) []reflect.Value {
		newVals := make([]reflect.Value, 0, len(args))
		for _, val := range args {
			t := val.Type()

			if t == reflect.TypeOf((*slog.Logger)(nil)) {
				l := val.Interface().(*slog.Logger)
				l = l.With(
					slog.String("component", "business:checker"),
					slog.Any("checker_id", checkerID),
				)

				val = reflect.ValueOf(l)
			}

			newVals = append(newVals, val)
		}

		return constr.Call(newVals)
	})

	registrar.Register(
		fx.Annotate(
			newConstr.Interface(),
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

// CheckerID is a struct containing information about what checker is responsible for.
// Every checker checks only one service, and has a name, usually name
// is just a user flow, for example stupido_user_flow.
type CheckerID struct {
	Service string
	Name    string
}

// Checker is an interface, which must be implemented by every keep-alive checker.
// Checkers verify service consistency and availability by running operations that create and verify data persistence.
//
// The Check() method is called very frequently and must be concurrently safe.
// In most cases, Check() should create some data in the service (e.g., create a user
// and register with provided credentials) and store any necessary state in Redis
// for later verification.
//
// State persistence: Any state created during Check() that needs to be verified later
// must be stored in Redis using the redis client, that can be provided within constructor.
// So, to create Check() that verifies data persistence, you need to get some random data from redis
// and check that data is present and not corrupted.
// You may use RedisStorage (see govnilo.RedisStorage) for that.
//
// All checkers must return the service name via CheckerID(). This name will be used in logs, metrics and some game mechanics.
// CheckerID() MUST work with nil receiver.
//
// Trace ID is provided in the context for debugging purposes.
// Use govnilo.GetLogger(ctx, logger) which automatically includes trace
// and span IDs in log output.
type Checker interface {
	// Check must run the most common flow of your service to verify that all components are working.
	// If you have multiple flows to check, you need to run these checks concurrently and wait
	// until their completion. You may use sync.WaitGroup or errgroup.Group for this.
	//
	// Any state that needs to be verified later must be stored in Redis using RedisStorage.
	// The context contains a trace ID for debugging: use govnilo.GetTraceID(ctx) to retrieve it,
	// or govnilo.GetLogger(ctx, logger) for automatic trace/span ID inclusion in logs.
	// You may use slog.InfoContext, slog.DebugContext and so on to include the trace and span ids in logs.
	Check(ctx context.Context, target string) error

	// CheckerID returns the id of the checker. This method is used for internal checker registration.
	// This method MUST work with nil receiver.
	CheckerID() CheckerID
}
