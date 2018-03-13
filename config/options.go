package config

/*
import (
	"reflect"
)
*/

const (
	// Configuration file default path
	DefaultConfigPath     = "$HOME/.gobalance/"
	DefaultConfigFilename = "conf"

	// Bitcoind
	DefaultBitcoindHost     = "localhost:8332"
	DefaultBitcoindMainnet  = "mainnet"
	DefaultBitcoindTestnet3 = "testnet3"

	// API
	DefaultApiUrlPrefix = "/"
	DefaultApiPort      = int64(8080)
	DefaultApiBind      = ""

	// Peers
	DefaultPeersPort              = int64(9090)
	DefaultPeersAllowLocalIps     = false
	DefaultPeersMode              = "full"
	DefaultPeersUnreachableMarks  = int64(3)
	DefaultPeersUnreachablePeriod = int64(5)


	//
	DefaultRecentBlocks     = int64(20)
	DefaultBalanceCacheSize = int64(100000)
	DefaultUtxoCacheSize    = int64(200000)
	DefaultSync				= false
	
)

var DefaultPeersSeeds  = [...]interface{} {}
var AllowedPeerModes = [...]string {"full", "seed", "loadbalance"}


type Option struct {

	// Option name
	name string

	// Validator usde to verify value
	val ValidatorFunc

	// Default value (nil for none)
	def  interface{}
}


var Options = [] Option {
	// bitcoind
	{	name: "bitcoind.host",
		val:  StringValidator(),
		def:  DefaultBitcoindHost,
	},

	{	name: "bitcoind.user",
		val: StringValidator(),
		def:  "",
	},

	{	name: "bitcoind.pass",
		val:  StringValidator(),
		def:  "",
	},

	{	name: "bitcoind.chain",
		val:  StringValidator(),
		def:  DefaultBitcoindMainnet,
	},

	// Api
	{	name: "api.url_prefix",
		val:  StringValidator(),
		def:  DefaultApiUrlPrefix,
	},

	{	name: "api.port",
		val:  Uint16Validator(),
		def:  DefaultApiPort,
	},

	{	name: "api.bind",
		val:  StringValidator(),
		def:  DefaultApiBind,
	},

	// Peers
	{	name: "peers.port",
		val:  Uint16Validator(),
		def:  DefaultPeersPort,
	},

	{	name: "peers.allow_local_ips",
		val:  BoolValidator(),
		def:  DefaultPeersAllowLocalIps,
	},
	
	{   name: "peers.mode",
		val:  StringChoiceValidator(AllowedPeerModes[:]...),
		def:  DefaultPeersMode,
	},

	{	name: "peers.unreachable_marks",
		val:  IntegerMinMaxValidator(1, 100000),
		def:  DefaultPeersUnreachableMarks,
	},
	
	{	name: "peers.unreachable_period",
		val:  IntegerMinMaxValidator(1, 999999),
		def:  DefaultPeersUnreachablePeriod,
	},

	{	name: "peers.seeds",
		val:  SliceElemValidator(StringValidator()),
		def:  DefaultPeersSeeds[:],
	},

	// Base
	{	name: "workdir",
		val:  StringValidator(),
		def:  DefaultConfigPath,
	},

	{	name: "recent_blocks",
		val:  IntegerMinValidator(1),
		def:  DefaultRecentBlocks,
	},

	{	name: "utxo_cache_size",
		val:  IntegerMinValidator(1),
		def:  DefaultUtxoCacheSize,
	},

	{	name: "balance_cache_size",
		val:  IntegerMinValidator(1),
		def:  DefaultBalanceCacheSize,
	},

	{	name: "sync",
		val:  BoolValidator(),
		def:  DefaultSync,
	},
}

