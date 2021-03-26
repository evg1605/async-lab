package workers_pool

import (
	"errors"
	"log"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimple(t *testing.T) {
	countJobs := runtime.GOMAXPROCS(-1)
	wp := StartNewPool(3)

	res := int64(0)
	for i := 0; i < countJobs; i++ {
		jobNum := i
		err := wp.AddJob(func() {
			log.Printf("start job %v", jobNum)
			atomic.AddInt64(&res, 1)
			time.Sleep(2 * time.Second)
			atomic.AddInt64(&res, 1)
			log.Printf("finish job %v", jobNum)
		})
		require.NoError(t, err)
	}
	log.Println("all jobs added")

	wp.WaitJobsAndStop()
	require.Equal(t, int64(countJobs*2), res)
}

func TestEmpty(t *testing.T) {
	wp := StartNewPool(3)
	wp.WaitJobsAndStop()
}

func TestAddError(t *testing.T) {
	wp := StartNewPool(3)
	wp.WaitJobsAndStop()
	err := wp.AddJob(func() {
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAddAfterClose))
}

func TestDoubleStop(t *testing.T) {
	wp := StartNewPool(3)
	wp.WaitJobsAndStop()
	wp.WaitJobsAndStop()
}
