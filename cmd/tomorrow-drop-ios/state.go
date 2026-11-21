package main

// The state the interface reads. Every field is plain data, so the interface
// only shows it and sends answers back through the bridge.

type deviceJSON struct {
	Name        string `json:"name"`
	Addr        string `json:"addr"`
	Paired      bool   `json:"paired"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type pairedJSON struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	Hex         string `json:"hex"`
	Addr        string `json:"addr,omitempty"`
}

type pairJSON struct {
	Local     string `json:"local"`
	Remote    string `json:"remote"`
	Stage     string `json:"stage"` // "confirm" or "name"
	Suggested string `json:"suggested"`
	Incoming  bool   `json:"incoming"`
}

type offerJSON struct {
	From  string   `json:"from"`
	Count int      `json:"count"`
	Bytes int64    `json:"bytes"`
	Names []string `json:"names"`
	Text  bool     `json:"text"`
}

type jobJSON struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Fraction float64 `json:"fraction"`
	State    string  `json:"state"` // "running", "done" or "failed"
	Message  string  `json:"message"`
}

type stateJSON struct {
	Fingerprint string       `json:"fingerprint"`
	Address     string       `json:"address"`
	Listening   bool         `json:"listening"`
	Devices     []deviceJSON `json:"devices"`
	Paired      []pairedJSON `json:"paired"`
	Pair        *pairJSON    `json:"pair,omitempty"`
	Offer       *offerJSON   `json:"offer,omitempty"`
	Jobs        []jobJSON    `json:"jobs"`
	Notice      string       `json:"notice,omitempty"`
	Result      *resultJSON  `json:"result,omitempty"`
	Received    *string      `json:"received,omitempty"`
}

// resultJSON is the outcome of an operation, shown once.
type resultJSON struct {
	OK      bool   `json:"ok"`
	Title   string `json:"title"`
	Message string `json:"message"`
}
