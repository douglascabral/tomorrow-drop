// Package limits centralizes the resource limits of the protocol and the core.
//
// The values protect against memory, disk and network exhaustion caused by
// unauthenticated or malicious peers.
package limits

const (
	// MaxControlFrameBytes is the maximum size of a control frame (JSON).
	MaxControlFrameBytes = 64 << 10
	// MaxDiscoveryDatagram is the maximum size of a UDP discovery datagram.
	MaxDiscoveryDatagram = 1 << 10
	// MaxFileNameBytes is the maximum size of a received file name.
	MaxFileNameBytes = 255
	// MaxFilesPerOffer is the maximum number of files in one offer.
	MaxFilesPerOffer = 1000
	// MaxTextBytes is the maximum size of a transferred text or link.
	MaxTextBytes = 1 << 20
	// MaxConcurrentTransfers is the number of simultaneous transfers.
	MaxConcurrentTransfers = 4
	// MaxDiscoveredDevices is the maximum size of the ephemeral device list.
	MaxDiscoveredDevices = 256
)
