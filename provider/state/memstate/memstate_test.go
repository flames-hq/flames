package memstate_test

import (
	"testing"

	"github.com/flames-hq/flames/provider/state"
	"github.com/flames-hq/flames/provider/state/memstate"
	"github.com/flames-hq/flames/providertest/statetest"
)

func TestConformance(t *testing.T) {
	statetest.Run(t, func() state.StateStore { return memstate.New() })
}
