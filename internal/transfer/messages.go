package transfer

// chunkSize is the file data sent per TRANSFER_DATA message. Base64 expands it
// by a third, so the encoded message stays under the control frame limit.
const chunkSize = 32 << 10

// Decline reasons sent to the other side.
const (
	reasonDeclined  = "declined"
	reasonBusy      = "busy"
	reasonInvalid   = "invalid"
	reasonIntegrity = "integrity"
	reasonSize      = "size"
	reasonCanceled  = "canceled"
)

type offerBody struct {
	ID    string     `json:"id"`
	Kind  Kind       `json:"kind"`
	Items []itemBody `json:"items"`
}

type itemBody struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type dataBody struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
	Data  []byte `json:"data"`
}

type endBody struct {
	ID     string `json:"id"`
	Index  int    `json:"index"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type idBody struct {
	ID string `json:"id"`
}

type declineBody struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type resultBody struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}
