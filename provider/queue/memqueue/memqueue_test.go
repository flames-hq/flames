package memqueue_test

import (
	"testing"

	"github.com/flames-hq/flames/provider/queue"
	"github.com/flames-hq/flames/provider/queue/memqueue"
	"github.com/flames-hq/flames/providertest/queuetest"
)

func TestConformance(t *testing.T) {
	queuetest.Run(t, func() queue.WorkQueue { return memqueue.New() })
}
