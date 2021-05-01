package last_progress

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProgress(t *testing.T) {
	pr := NewProgress()

	pr.SetLastProgress("val1")
	pr.SetLastProgress("val2")
	pr.SetLastProgress("val3")

	v := <-pr.GetLastProgress()
	vStr := v.(string)
	require.NotEqual(t, "val1", vStr)
}
