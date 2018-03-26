package config

import(
	"testing"
	"runtime"
	"path/filepath"
	"fmt"
)

const (
	TestDataRelativePath = "test_data/"
)


func GetTestDataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return fmt.Sprintf("%v/%v", filepath.Dir(filename), TestDataRelativePath)
}




// Test item in storage
func TestValidConfigFile(t *testing.T) {

	data, err := ReadConfig(GetTestDataPath(), "valid_conf")
	if err != nil {
		t.Errorf("ReadConfig(): %v", err)
	}

	// Check option values
	if data["utxo_cache_size"].(int64) != 55555 {
		t.Errorf("utxo_cache_size: Unexpected value")
	}

	if data["balance_cache_size"].(int64) != 111111 {
		t.Errorf("balance_cache_size: Unexpected value")
	}
	
	if data["recent_blocks"].(int64) != 22 {
		t.Errorf("recent_blocks: Unexpected value")
	}

	if data["workdir"].(string) != "thatdir" {
		t.Errorf("workdir: Unexpected value")
	}

	if data["sync"].(bool) != true {
		t.Errorf("sync: Unexpected value")
	}	
	
	if data["mode"].(string) != "seed" {
		t.Errorf("peers.mode: Unexpected value")
	}

	
	// Test api option values
	if data["api.url_prefix"].(string) != "/api/v1/" {
		t.Errorf("api.url_prefix: Unexpected value")
	}
	if data["api.port"].(int64) != 1000 {
		t.Errorf("api.port: Unexpected value")
	}
	if data["api.bind"].(string) != "127.0.0.1" {
		t.Errorf("api.bind: Unexpected value")
	}

	// Test peers option values
	if data["peers.port"].(int64) != 4000 {
		t.Errorf("peers.port: Unexpected value")
	}
	if data["peers.allow_local_ips"].(bool) != true {
		t.Errorf("peers.allow_local_ips: Unexpected value")
	}
	if data["peers.unreachable_marks"].(int64) != int64(10) {
		t.Errorf("peers.ureachable_marks: Unexpected value")
	}
	if data["peers.unreachable_period"].(int64) != int64(100) {
		t.Errorf("peers.ureachable_period: Unexpected value")
	}

	// Test peers seeds
	seeds := data["peers.seeds"].([]interface{})
	if len(seeds) != 2 {
		t.Errorf("peers.seeds: Expection two seed string returned %v", len(seeds))
	}

	seedsMap := make(map[string]bool)
	for _, seed := range seeds {
		seedsMap[seed.(string)] = true
	}

	if _, ok := seedsMap["seed1.unknown.com"]; !ok {
		t.Errorf("peers.seeds: missing seed seed1.unknown.com")
	}
	if _, ok := seedsMap["seed2.unknown.com"]; !ok {
		t.Errorf("peers.seeds: missing seed seed2.unknown.com")
	}

	// Test bitcoind option values
	if data["bitcoind.host"].(string) != "localhost:8000" {
		t.Errorf("bitcoind.host: Unexpected value")
	}
	if data["bitcoind.user"].(string) != "gobalance" {
		t.Errorf("bitcoind.user: Unexpected value")
	}
	if data["bitcoind.pass"].(string) != "12345" {
		t.Errorf("bitcoind.pass: Unexpected value")
	}
	if data["bitcoind.chain"].(string) != "testnet3" {
		t.Errorf("bitcoind.chain: Unexpected value")
	}
}


// Test default values for missing options
func TestDefaultValuesConfigFile(t *testing.T) {

	data, err := ReadConfig(GetTestDataPath(), "default_conf")
	if err != nil {
		t.Errorf("ReadConfig(): %v", err)
		return
	}

	// Check option values
	if data["utxo_cache_size"].(int64) != DefaultUtxoCacheSize {
		t.Errorf("utxo_cache_size: Unexpected default value")
	}

	if data["balance_cache_size"].(int64) != DefaultBalanceCacheSize {
		t.Errorf("balance_cache_size: Unexpected default value")
	}
	
	if data["recent_blocks"].(int64) != DefaultRecentBlocks {
		t.Errorf("recent_blocks: Unexpected default value")
	}

	if data["workdir"].(string) != DefaultConfigPath {
		t.Errorf("workdir: Unexpected default value")
	}
	
	if data["sync"].(bool) != false {
		t.Errorf("sync: Unexpected default value")
	}	
	
	if data["mode"].(string) != DefaultMode {
		t.Errorf("peers.mode: Unexpected default value")
	}
	
	// Test api option values
	if data["api.url_prefix"].(string) != DefaultApiUrlPrefix {
		t.Errorf("api.url_prefix: Unexpected default value")
	}
	if data["api.port"].(int64) != DefaultApiPort {
		t.Errorf("api.port: Unexpected defalut value")
	}
	if data["api.bind"].(string) != DefaultApiBind {
		t.Errorf("api.bind: Unexpected default value")
	}

	// Test peers option values
	if data["peers.port"].(int64) != DefaultPeersPort {
		t.Errorf("peers.port: Unexpected default value")
	}
	if data["peers.allow_local_ips"].(bool) != DefaultPeersAllowLocalIps {
		t.Errorf("peers.allow_local_ips: Unexpected default value")
	}
	if data["peers.unreachable_marks"].(int64) != DefaultPeersUnreachableMarks {
		t.Errorf("peers.unreachable_marks: Unexpected default value")
	}
	if data["peers.unreachable_period"].(int64) != DefaultPeersUnreachablePeriod {
		t.Errorf("peers.unreachable_period: Unexpected default value")
	}
	if len(data["peers.seeds"].([]interface{})) != 0 {
		t.Errorf("peers.seeds: Unexpected default value")
	}

	// Test bitcoind option values
	if data["bitcoind.host"].(string) != DefaultBitcoindHost {
		t.Errorf("bitcoind.host: Unexpected default value")
	}
	if data["bitcoind.user"] != "" {
		t.Errorf("bitcoind.user: Unexpected defalut value")
	}
	if data["bitcoind.pass"] != "" {
		t.Errorf("bitcoind.pass: Unexpected default value")
	}
	if data["bitcoind.chain"] != DefaultBitcoindMainnet {
		t.Errorf("bitcoind.chain: Unexpected default value")
	}
}

// Test ReadConfig returns an error when reading config file containing unsupported options
func TestUnknownOptionsConfigFile(t *testing.T) {

	_, err := ReadConfig(GetTestDataPath(), "unsupported_conf")
	if err == nil {
		t.Errorf("ReadConfig(): Should have returned an error")
	}
}

// Test ReadConfig returns an error 
func TestInvalidTypeOptionsConfigFile(t *testing.T) {
	
	_, err := ReadConfig(GetTestDataPath(), "type_string_conf")
	if err == nil {
		t.Errorf("ReadConfig(): Should have returned an error")
	}

	_, err = ReadConfig(GetTestDataPath(), "type_int_conf")
	if err == nil {
		t.Errorf("ReadConfig(): Should have returned an error")
	}
}

