package merger

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMergeSync(t *testing.T) {

	sources := []*Source{
		{
			GroupName: "gr1",
			ObjProperties: map[string]string{
				"item1": "gr1val for item1",
				"item2": "gr1val for item2",
				"item3": "gr1val for item3"},
		}, {
			GroupName: "gr2",
			ObjProperties: map[string]string{
				"item1": "gr2val for item1",
				"item2": "gr2val for item2",
				"item3": "gr2val for item3"},
		}, {
			GroupName: "gr3",
			ObjProperties: map[string]string{
				"item3": "gr3val for item3"},
		}}

	res, err := MergeSync(sources)
	require.NoError(t, err)
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
