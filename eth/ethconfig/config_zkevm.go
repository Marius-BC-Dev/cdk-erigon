package ethconfig

import (
	"time"

	"github.com/gateway-fm/cdk-erigon-lib/common"
)

type Zk struct {
	L2ChainId                  uint64
	L2RpcUrl                   string
	L2DataStreamerUrl          string
	L2DataStreamerTimeout      time.Duration
	L1ChainId                  uint64
	L1RpcUrl                   string
	AddressSequencer           common.Address
	AddressAdmin               common.Address
	AddressRollup              common.Address
	AddressZkevm               common.Address
	AddressGerManager          common.Address
	L1RollupId                 uint64
	L1BlockRange               uint64
	L1QueryDelay               uint64
	L1MaticContractAddress     common.Address
	L1FirstBlock               uint64
	RpcRateLimits              int
	DatastreamVersion          int
	SequencerInitialForkId     uint64
	ExecutorUrls               []string
	ExecutorStrictMode         bool
	ExecutorRecordToDisk       bool
	AllowFreeTransactions      bool
	AllowPreEIP155Transactions bool

	RebuildTreeAfter uint64
	WitnessFull      bool
}

var DefaultZkConfig = &Zk{}
