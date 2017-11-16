package config

import (
	"reflect"
)

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

	//
	DefaultRecentBlocks     = int64(20)
	DefaultBalanceCacheSize = int64(100000)
	DefaultUtxoCacheSize    = int64(200000)
)


type Option struct {

	// Option name
	name string

	// Allowed type for the option value
	typ  reflect.Kind

	// Default value (nil for none)
	def  interface{}
}


var Options = [] Option {
	// bitcoind
	{	name: "bitcoind.host",
		typ:  reflect.String,
		def:  DefaultBitcoindHost,
	},

	{	name: "bitcoind.user",
		typ:  reflect.String,
		def:  "",
	},

	{	name: "bitcoind.pass",
		typ:  reflect.String,
		def:  "",
	},

	{	name: "bitcoind.chain",
		typ:  reflect.String,
		def:  DefaultBitcoindMainnet,
	},

	// Api
	{	name: "api.url_prefix",
		typ:  reflect.String,
		def:  DefaultApiUrlPrefix,
	},

	{	name: "api.port",
		typ:  reflect.Int64,
		def:  DefaultApiPort,
	},

	{	name: "api.bind",
		typ:  reflect.String,
		def:  DefaultApiBind,
	},

	// Base
	{	name: "workdir",
		typ:  reflect.String,
		def:  DefaultConfigPath,
	},

	{	name: "recent_blocks",
		typ:  reflect.Int64,
		def:  DefaultRecentBlocks,
	},

	{	name: "utxo_cache_size",
		typ:  reflect.Int64,
		def:  DefaultUtxoCacheSize,
	},

	{	name: "balance_cache_size",
		typ:  reflect.Int64,
		def:  DefaultBalanceCacheSize,
	},
}


