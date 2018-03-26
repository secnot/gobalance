package peers

type Peers struct {
	Peers []string `json:peers`
}

type Status struct {
	Mode PeerMode `json:mode`

	// Peer API port
	Port uint16 `json:port`

	// Balance API port
	BalancePort uint16 `json:balanceport`

	// Time passed since the peer was created (in seconds)
	Uptime int64  `json:uptime`

	// Software version being used
	Version string `json:version`
}

