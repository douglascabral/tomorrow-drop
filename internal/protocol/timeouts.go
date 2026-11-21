package protocol

import "time"

const (
	// HandshakeTimeout limita o estabelecimento de TLS e a troca de versões.
	HandshakeTimeout = 10 * time.Second
	// DecisionTimeout limita a espera por aceite ou recusa do destinatário (PRD §13).
	DecisionTimeout = 60 * time.Second
	// IdleTimeout encerra conexões sem tráfego.
	IdleTimeout = 30 * time.Second
	// DiscoveryInterval é o intervalo de DISCOVER enquanto o app estiver visível (PRD §5.2).
	DiscoveryInterval = 30 * time.Second
)
