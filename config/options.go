package config

const (
	// Bitcoind
	DefaultBitcoindMainnet  = "mainnet"
	DefaultBitcoindTestnet3 = "testnet3"

	// API
	DefaultApiUrlPrefix = "/"
	DefaultApiPort      = int64(8080)
	DefaultApiBind      = ""

	// Peers
	DefaultPeersPort              = int64(9090)
	DefaultPeersAllowLocalIps     = false
	DefaultPeersUnreachableMarks  = int64(3)
	DefaultPeersUnreachablePeriod = int64(5)


	//
	DefaultRecentBlocks     = int64(20)
	DefaultBalanceCacheSize = int64(100000)
	DefaultUtxoCacheSize    = int64(200000)
	DefaultSync				= false
	DefaultMode             = "full"
)

var DefaultPeersSeeds    = [...]string {}
var DefaultBitcoindHosts = [...]string {"localhost:8332"}
var AllowedPeerModes     = [...]string {"full", "seed", "loadbalance"}


type Option struct {

	// Option name
	name string

	// Validator usde to verify value
	val ValidatorFunc

	// Default value (nil for none)
	def  interface{}

	// Cast values to correct type, required for env vars
	cast CastFunc
}

var Options = [] Option {
	// bitcoind
	{	name: "bitcoind.hosts",
		val:  StringSliceValidator(),
		def:  DefaultBitcoindHosts[:],
		cast: CastStringSlice,
	},

	{	name: "bitcoind.user",
		val: StringValidator(),
		def:  "",
		cast: CastString,
	},

	{	name: "bitcoind.pass",
		val:  StringValidator(),
		def:  "",
		cast: CastString,
	},

	{	name: "bitcoind.chain",
		val:  StringValidator(),
		def:  DefaultBitcoindMainnet,
		cast: CastString,
	},

	// Api
	{	name: "api.url_prefix",
		val:  StringValidator(),
		def:  DefaultApiUrlPrefix,
		cast: CastString,
	},

	{	name: "api.port",
		val:  Uint16Validator(),
		def:  DefaultApiPort,
		cast: CastInt64,
	},

	{	name: "api.bind",
		val:  StringValidator(),
		def:  DefaultApiBind,
		cast: CastString,
	},

	// Peers
	{	name: "peers.port",
		val:  Uint16Validator(),
		def:  DefaultPeersPort,
		cast: CastInt64,
	},

	{	name: "peers.allow_local_ips",
		val:  BoolValidator(),
		def:  DefaultPeersAllowLocalIps,
		cast: CastBool,
	},
	
	{	name: "peers.unreachable_marks",
		val:  IntegerMinMaxValidator(1, 100000),
		def:  DefaultPeersUnreachableMarks,
		cast: CastInt64,
	},
	
	{	name: "peers.unreachable_period",
		val:  IntegerMinMaxValidator(1, 999999),
		def:  DefaultPeersUnreachablePeriod,
		cast: CastInt64,
	},

	{	name: "peers.seeds",
		val:  StringSliceValidator(),
		def:  DefaultPeersSeeds[:],
		cast: CastStringSlice,
	},

	// Base
	{	name: "workdir",
		val:  StringValidator(),
		def:  DefaultConfigPath,
		cast: CastString,
	},

	{	name: "recent_blocks",
		val:  IntegerMinValidator(1),
		def:  DefaultRecentBlocks,
		cast: CastInt64,
	},

	{	name: "utxo_cache_size",
		val:  IntegerMinValidator(1),
		def:  DefaultUtxoCacheSize,
		cast: CastInt64,
	},

	{	name: "balance_cache_size",
		val:  IntegerMinValidator(1),
		def:  DefaultBalanceCacheSize,
		cast: CastInt64,
	},
	
	{   name: "mode",
		val:  StringChoiceValidator(AllowedPeerModes[:]...),
		def:  DefaultMode,
		cast: CastString,
	},

	{	name: "sync",
		val:  BoolValidator(),
		def:  DefaultSync,
		cast: CastBool,
	},
}

