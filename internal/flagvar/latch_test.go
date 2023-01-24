package flagvar

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLatch(t *testing.T) {
	myBool := false
	val := Latch(Value(&myBool))
	require := require.New(t)

	first := "true"
	for _, set := range []string{first, "false"} {
		require.NoError(val.Set(set))
		require.Equal(first, val.String())
		require.True(myBool)
	}
}
