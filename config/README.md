# Configuration

Configuration file format is toml

## Base options

**sync (bool)**: Switch between normal and sync mode, in sync mode the program will exit as soon as the utxo db is synced to the last block. (default: false)

**mode (string)**: Mode of operation "full"|"seed"|"loadbalance" (default: "full")

**workdir (string)**: Working directory where utxo DB and configuration files are stored (default: ~/.gobalance)

**utxo_cache_size (int)**: Number of utxo cache before a commit to DB is Required (default: 10000)

**balance_cache_size (int)**: Max address balance cached in memory.

**recent_blocks (int)**: Number of blocks required for a block to be assumed confirmed and elegible to commit to db. (default: 20)


### [peers]

**port (uint16)**: Port used to listen for requests from other peers in the cluster (default:9090)

**allow_local_ips (bool)**: Allow communication with peers using one of the hosts ips, mainly for testing (default: false)

**unreachable_marks (integer)**: Number of failed connection attempts before a peer is marked unreachable. (default: 3)
    
**unreachable_period (integer)**: Time interval where all  (default: 100)

**seeds (string array)**: Cluster seed peer address (i.e: ["seed1.unknown.com:9090", "seed2.unknown.com:9090"])


### [api]

**url_prefix (string)**: Optional url prefix for the api end points (default: "/")

**port (uint16)**: Balance api Listen port(default: 8080)

**bind (string)**: IP address to bind the service to (default: "")


### [bitcoind]

**host (string)**: Bitcoind server hostname or ip address (i.e. "server1.unknown.com:8332")

**user (string)**: Bitcoind service username

**pass (string)**; Bitcoind service password

**chain (string)**: Chain selection "mainnet" or "testnet3" (default: "mainnet")


## Environment variables

All the configuration options can also be set using environment vars, the values provided
this way will shadow the values in the configuration file. The env options name are:


- **GOBALANCE_SYNC**
- **GOBALANCE_MODE**
- **GOBALANCE_WORKDIR**
- **GOBALANCE_UTXO_CACHE_SIZE**
- **GOBALANCE_BALANCE_CACHE_SIZE**
- **GOBALANCE_RECENT_BLOCKS**

- **GOBALANCE_PEERS_PORT**
- **GOBALANCE_PEERS_ALLOW_LOCAL_IPS**
- **GOBALANCE_PEERS_UNREACHABLE_MARKS**
- **GOBALANCE_PEERS_UNREACHABLE_PERIOD**
- **GOBALANCE_PEERS_SEEDS**

- **GOBALANCE_API_URL_PREFIX**
- **GOBALANCE_API_PORT**
- **GOBALANCE_API_BIND**

- **GOBALANCE_BITCOIND_HOST**
- **GOBALANCE_BITCOIND_USER**
- **GOBALANCE_BITCOIND_PASS**
- **GOBALANCE_BITCOIND_CHAIN**
