// Package limits centraliza os limites de recursos do protocolo e do núcleo.
//
// Valores protegem contra exaustão de memória, disco e rede por pares
// não autenticados ou maliciosos.
package limits

const (
	// MaxControlFrameBytes é o tamanho máximo de um quadro de controle (JSON).
	MaxControlFrameBytes = 64 << 10
	// MaxDiscoveryDatagram é o tamanho máximo de um datagrama UDP de descoberta.
	MaxDiscoveryDatagram = 1 << 10
	// MaxFileNameBytes é o tamanho máximo de um nome de arquivo recebido.
	MaxFileNameBytes = 255
	// MaxFilesPerOffer é a quantidade máxima de arquivos em uma oferta.
	MaxFilesPerOffer = 1000
	// MaxTextBytes é o tamanho máximo de texto ou link transferido.
	MaxTextBytes = 1 << 20
	// MaxConcurrentTransfers é o número de transferências simultâneas.
	MaxConcurrentTransfers = 4
	// MaxDiscoveredDevices é o número máximo de dispositivos na lista efêmera.
	MaxDiscoveredDevices = 256
)
