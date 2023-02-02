package benchmark

import "testing"

func Run(b *testing.B, setup, op func()) {
	b.StopTimer()
	b.Helper()
	if setup != nil {
		setup()
	}

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		op()
	}
	b.StopTimer()
}
