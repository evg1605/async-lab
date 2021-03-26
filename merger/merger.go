package merger

import (
	"github.com/evg1605/async-lab/workers_pool"
	"runtime"
	"sync"
)

type Source struct {
	GroupName        string
	ObjectsWithValue map[string]string
}

type resultWithCount struct {
	countGroups int
	result      Result
}

type Result = map[string]map[string]string

func MergeSync(sources []*Source) (Result, error) {
	res := make(Result)

	for _, src := range sources {

		for name, val := range src.ObjectsWithValue {
			if _, ok := res[name]; !ok {
				res[name] = make(map[string]string)
			}
			res[name][src.GroupName] = val
		}
	}
	return res, nil
}

func MergeAsyncWithWp(sources []*Source) (Result, error) {
	res := make(Result)

	wp := workers_pool.StartNewPool(runtime.NumCPU())
	for _, src := range sources {

		for name, _ := range src.ObjectsWithValue {
			if _, ok := res[name]; ok {
				continue
			}

			objName := name
			m := make(map[string]string)
			res[objName] = m
			_ = wp.AddJob(func() {
				for _, src := range sources {
					if objVal, ok := src.ObjectsWithValue[objName]; ok {
						m[src.GroupName] = objVal
					}
				}
			})
		}
	}
	wp.WaitJobsAndStop()
	return res, nil
}

func MergeAsync(sources []*Source) (Result, error) {
	res := make(Result)

	countWorkers := runtime.NumCPU()
	objCh := make(chan struct {
		name string
		m    map[string]string
	}, countWorkers*2)
	wg := &sync.WaitGroup{}
	wg.Add(countWorkers)

	for i := 0; i < countWorkers; i++ {
		go func() {
			defer wg.Done()
			for obj := range objCh {
				for _, src := range sources {
					if objVal, ok := src.ObjectsWithValue[obj.name]; ok {
						obj.m[src.GroupName] = objVal
					}
				}
			}
		}()
	}

	for _, src := range sources {
		for name, _ := range src.ObjectsWithValue {
			if _, ok := res[name]; ok {
				continue
			}

			m := make(map[string]string)
			res[name] = m
			objCh <- struct {
				name string
				m    map[string]string
			}{name: name, m: m}
		}
	}
	close(objCh)

	wg.Wait()
	return res, nil
}
