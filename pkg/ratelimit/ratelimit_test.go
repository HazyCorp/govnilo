package ratelimit

import (
	"context"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"
)

func new(times uint64, per time.Duration) *Limiter {
	return New(Spec{Times: times, Per: per})
}

func TestBlockedRateLimit(t *testing.T) {
	defer goleak.VerifyNone(t)

	limit := new(1, time.Minute)
	defer limit.Stop(context.Background())

	require.NoError(t, limit.Acquire(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := limit.Acquire(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSimpleLimitCancel(t *testing.T) {
	defer goleak.VerifyNone(t)

	limit := new(1, time.Minute)
	defer limit.Stop(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	require.NoError(t, limit.Acquire(ctx))

	err := limit.Acquire(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestAcquireAfterDelay(t *testing.T) {
	defer goleak.VerifyNone(t)

	e := 2
	N := 5
	limit := new(uint64(N), time.Second)
	defer limit.Stop(context.Background())

	for epoch := 0; epoch < e; epoch++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		for i := 0; i < N; i++ {
			require.NoError(t, limit.Acquire(ctx))
		}
		cancel()
		time.Sleep(time.Second * 2)
	}
}

func TestAcquireAfterStopped(t *testing.T) {
	defer goleak.VerifyNone(t)

	const nTries = 100

	limit := new(200, time.Second)
	limit.Stop(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for i := 0; i < nTries; i++ {
		err := limit.Acquire(ctx)
		require.ErrorIs(t, err, ErrStopped)
	}
}

func TestTimeDistribution(t *testing.T) {
	defer goleak.VerifyNone(t)

	limit := new(100, time.Second)
	defer limit.Stop(context.Background())

	var lock sync.Mutex
	okTimes := []time.Duration{}
	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		time.Sleep(time.Millisecond * 5)

		wg.Add(1)
		go func() {
			defer wg.Done()

			dt := time.Duration(rand.Float64() * float64(time.Second))
			ctx, cancel := context.WithTimeout(context.Background(), dt)
			defer cancel()

			err := limit.Acquire(ctx)
			if err != nil {
				return
			}

			lock.Lock()
			defer lock.Unlock()
			okTimes = append(okTimes, time.Since(start))
		}()
	}

	wg.Wait()

	require.Greater(t, len(okTimes), 200, "At least 200 goroutines should succeed")

	sort.Slice(okTimes, func(i, j int) bool {
		return okTimes[i] < okTimes[j]
	})

	for i, dt := range okTimes {
		j := sort.Search(len(okTimes)-i, func(j int) bool {
			return okTimes[i+j] > dt+time.Second
		})

		require.Lessf(
			t,
			j,
			130,
			"%d goroutines acquired semaphore on interval [%v, %v)",
			j,
			dt,
			dt+time.Second,
		)
	}

	// Uncomment this line to see full distribution
	spew.Fdump(os.Stderr, okTimes)
}

func TestStressBlocking(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		N = 100
		G = 100
	)

	limit := new(N, time.Millisecond*10)
	defer limit.Stop(context.Background())

	var eg errgroup.Group
	for i := 0; i < G; i++ {
		eg.Go(func() error {
			for j := 0; j < N; j++ {
				if err := limit.Acquire(context.Background()); err != nil {
					return err
				}
			}

			return nil
		})
	}

	require.NoError(t, eg.Wait())
}

func TestStressNoBlocking(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		N = 100
		G = 100
	)

	limit := new(N, time.Millisecond*10)
	defer limit.Stop(context.Background())

	var eg errgroup.Group
	for i := 0; i < G; i++ {
		eg.Go(func() error {
			for j := 0; j < N; j++ {
				if err := limit.Acquire(context.Background()); err != nil {
					return err
				}

				time.Sleep(time.Millisecond * 11)
			}

			return nil
		})
	}

	require.NoError(t, eg.Wait())
}

func BenchmarkNoBlocking(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(1)

	limit := new(1, 0)
	defer limit.Stop(context.Background())

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := limit.Acquire(ctx); err != nil {
				b.Errorf("acquire failed: %v", err)
			}
		}
	})
}

func BenchmarkReferenceMutex(b *testing.B) {
	var mu sync.Mutex

	var j int
	for i := 0; i < b.N; i++ {
		mu.Lock()
		j++
		mu.Unlock()
	}
}
