package limits

import (
	"math"
	"testing"
)

func TestLimitsArePositive(t *testing.T) {
	values := map[string]int64{
		"MaxControlFrameBytes":   MaxControlFrameBytes,
		"MaxDiscoveryDatagram":   MaxDiscoveryDatagram,
		"MaxFileNameBytes":       MaxFileNameBytes,
		"MaxFilesPerOffer":       MaxFilesPerOffer,
		"MaxTextBytes":           MaxTextBytes,
		"MaxFileBytes":           MaxFileBytes,
		"MaxOfferBytes":          MaxOfferBytes,
		"MaxConcurrentTransfers": MaxConcurrentTransfers,
		"MaxDiscoveredDevices":   MaxDiscoveredDevices,
	}
	for name, value := range values {
		if value <= 0 {
			t.Errorf("%s = %d, want a positive value", name, value)
		}
	}
}

func TestLimitsAreConsistent(t *testing.T) {
	if MaxDiscoveryDatagram > MaxControlFrameBytes {
		t.Errorf("UDP datagram (%d) must not exceed the control frame (%d)", MaxDiscoveryDatagram, MaxControlFrameBytes)
	}
	if MaxControlFrameBytes > math.MaxUint32 {
		t.Errorf("control frame does not fit in the 32-bit header")
	}
	if MaxFileNameBytes > 255 {
		t.Errorf("file name (%d bytes) exceeds the common file system limit", MaxFileNameBytes)
	}
}
