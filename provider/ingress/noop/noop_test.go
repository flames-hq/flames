package noop_test

import (
	"testing"

	"github.com/flames-hq/flames/provider/ingress"
	"github.com/flames-hq/flames/provider/ingress/noop"
	"github.com/flames-hq/flames/providertest/ingresstest"
)

func TestConformance(t *testing.T) {
	ingresstest.Run(t, func() ingress.IngressProvider { return noop.New() })
}
