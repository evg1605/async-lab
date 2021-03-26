package merger

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createSources1() []*Source {
	return []*Source{
		{
			GroupName: "gr1",
			ObjectsWithValue: map[string]string{
				"item1": "gr1val for item1",
				"item2": "gr1val for item2",
				"item3": "gr1val for item3"},
		}, {
			GroupName: "gr2",
			ObjectsWithValue: map[string]string{
				"item1": "gr2val for item1",
				"item2": "gr2val for item2",
				"item3": "gr2val for item3"},
		}, {
			GroupName: "gr3",
			ObjectsWithValue: map[string]string{
				"item3": "gr3val for item3"},
		}}
}

func checkResult1(t *testing.T, res Result) {

	require.NotNil(t, res)

	require.Len(t, res, 3)
	require.Contains(t, res, "item1")
	require.Contains(t, res, "item2")
	require.Contains(t, res, "item3")

	require.Len(t, res["item1"], 2)
	require.Len(t, res["item2"], 2)
	require.Len(t, res["item3"], 3)

	require.Contains(t, res["item3"], "gr1")
	require.Contains(t, res["item3"], "gr2")
	require.Contains(t, res["item3"], "gr3")

	require.Equal(t, res["item3"]["gr2"], "gr2val for item3")
}

func createBigSources(groupsCount, itemsCount int) []*Source {
	sources := make([]*Source, groupsCount)
	for groupNum := 0; groupNum < groupsCount; groupNum++ {
		src := &Source{
			GroupName:        fmt.Sprintf("group-%v", groupNum),
			ObjectsWithValue: make(map[string]string),
		}
		for itemNum := 0; itemNum < itemsCount; itemNum++ {
			src.ObjectsWithValue[fmt.Sprintf("item-%v", itemNum)] = fmt.Sprintf("item-%v val for group %v", itemNum, groupNum)
		}
		sources[groupNum] = src
	}
	return sources
}

func TestMergeSync(t *testing.T) {

	sources := createSources1()

	res, err := MergeSync(sources)

	require.NoError(t, err)
	checkResult1(t, res)
}

func TestMergeAsyncWithWp(t *testing.T) {

	sources := createSources1()

	res, err := MergeAsyncWithWp(sources)

	require.NoError(t, err)
	checkResult1(t, res)
}

func TestMergeAsync(t *testing.T) {

	sources := createSources1()

	res, err := MergeAsync(sources)

	require.NoError(t, err)
	checkResult1(t, res)
}

func TestMergeCompare(t *testing.T) {
	sources := createBigSources(1000, 1000)

	log.Println("MergeAsyncWithWp")
	now := time.Now()
	_, _ = MergeAsyncWithWp(sources)
	log.Println(time.Now().Sub(now).String())

	log.Println("MergeAsync")
	now = time.Now()
	_, _ = MergeAsync(sources)
	log.Println(time.Now().Sub(now).String())

	log.Println("MergeSync")
	now = time.Now()
	_, _ = MergeSync(sources)
	log.Println(time.Now().Sub(now).String())

}

func BenchmarkMergeAsyncWithWp(b *testing.B) {
	sources := createBigSources(1000, 500)
	b.ResetTimer()

	_, _ = MergeAsyncWithWp(sources)
}

func BenchmarkMergeAsync(b *testing.B) {
	sources := createBigSources(1000, 500)
	b.ResetTimer()

	_, _ = MergeAsync(sources)
}

func BenchmarkMergeSync(b *testing.B) {
	sources := createBigSources(1000, 500)
	b.ResetTimer()

	_, _ = MergeSync(sources)
}
