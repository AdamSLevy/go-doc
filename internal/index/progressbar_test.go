package index

import (
	"testing"
	"time"
)

func TestProgressBar(t *testing.T) {
	t.Skip()
	t.Log("testing progress bar...")
	pb := newProgressBar(options{}, 1000, "syncing...")
	for i := 0; i < 1000; i++ {
		pb.Add(1)
		time.Sleep(2 * time.Millisecond)
	}
	pb.Finish()
}
