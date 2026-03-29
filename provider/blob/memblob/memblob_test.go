package memblob_test

import (
	"testing"

	"github.com/flames-hq/flames/provider/blob"
	"github.com/flames-hq/flames/provider/blob/memblob"
	"github.com/flames-hq/flames/providertest/blobtest"
)

func TestConformance(t *testing.T) {
	blobtest.Run(t, func() blob.BlobStore { return memblob.New() })
}
