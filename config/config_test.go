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
	if data["bitcoind.chain"].(string) != "testnet99" {
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

