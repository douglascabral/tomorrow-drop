package limits

import (
	"math"
	"testing"
)

func TestLimitesSaoPositivos(t *testing.T) {
	valores := map[string]int64{
		"MaxControlFrameBytes":   MaxControlFrameBytes,
		"MaxDiscoveryDatagram":   MaxDiscoveryDatagram,
		"MaxFileNameBytes":       MaxFileNameBytes,
		"MaxFilesPerOffer":       MaxFilesPerOffer,
		"MaxTextBytes":           MaxTextBytes,
		"MaxConcurrentTransfers": MaxConcurrentTransfers,
		"MaxDiscoveredDevices":   MaxDiscoveredDevices,
	}
	for nome, valor := range valores {
		if valor <= 0 {
			t.Errorf("%s = %d, esperado valor positivo", nome, valor)
		}
	}
}

func TestLimitesSaoCoerentes(t *testing.T) {
	if MaxDiscoveryDatagram > MaxControlFrameBytes {
		t.Errorf("datagrama UDP (%d) não pode exceder o quadro de controle (%d)", MaxDiscoveryDatagram, MaxControlFrameBytes)
	}
	if MaxControlFrameBytes > math.MaxUint32 {
		t.Errorf("quadro de controle não cabe no cabeçalho de 32 bits")
	}
	if MaxFileNameBytes > 255 {
		t.Errorf("nome de arquivo (%d bytes) excede o limite comum dos sistemas de arquivos", MaxFileNameBytes)
	}
}
