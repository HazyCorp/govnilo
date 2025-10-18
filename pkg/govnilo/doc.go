// Package govnilo provides a framework for implementing keep-alive checkers and sploits.
//
// Govnilo is a framework for testing service consistency and running attacks:
//   - Checkers verify service consistency over time using CHECK/GET pattern
//   - Sploits run attacks against target services
//   - Infrastructure handles orchestration, rate limiting, and data persistence
//
// Checker Pattern:
//   - Check(): Runs service flow and returns data that should persist
//   - Get(): Verifies that data from Check still exists in service
//   - Tests service consistency and data persistence over time
//   - Each operation gets a unique trace ID for debugging
//
// Sploit Pattern:
//   - RunAttack(): Executes attacks against target services
//   - Used for testing service resilience and attack resistance
//   - Each operation gets a unique trace ID for debugging
//
// Registration:
//   - Use govnilo.RegisterChecker() to register checker implementations
//   - Use govnilo.RegisterSploit() to register sploit implementations
//   - Registration typically happens in init() functions
//
// Example Usage:
//
//	import "github.com/HazyCorp/govnilo/pkg/govnilo"
//
//	func init() {
//	    govnilo.RegisterChecker(NewMyChecker)
//	    govnilo.RegisterSploit(NewMySploit)
//	}
//
//	// In your Check/Get/RunAttack methods:
//	func (c *MyChecker) Check(ctx context.Context, target string) ([]byte, error) {
//	    // Trace ID is automatically included in logs via hzlog context
//	    c.l.DebugContext(ctx, "Starting CHECK operation")
//	    // Your logic here...
//	}
//
// For more examples, see the _example/ directory.
package govnilo
