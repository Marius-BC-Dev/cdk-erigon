package zkchainconfig

import "github.com/ledgerwatch/erigon/params/networkname"

var chainIds = []uint64{
	195,    // x1-testnet
	1101,   // mainnet
	2440,   // cardona internal
	2442,   // cardona
	10010,  //etrog testnet
	999999, // local devnet

	1001, // lumoz private chain layer2
}

var chainIdToName = map[uint64]string{
	195:    networkname.X1TestnetChainName,
	1101:   networkname.HermezMainnetChainName,
	2440:   networkname.HermezBaliChainName,
	2442:   networkname.HermezCardonaChainName,
	10010:  networkname.HermezEtrogChainName,
	999999: networkname.HermezLocalDevnetChainName,
	123:    networkname.HermezESTestChainName,

	9: networkname.LomuzPrivateTestChainChainName,
	//1001:  networkname.LomuzPrivateTestChainLayer2ChainName,
	55555: networkname.SaasL1TestnetChainName,
	5200:  networkname.MerlinV2TestnetChainName,
	1001:  networkname.ZkevmFork58ChainName,
}

func IsZk(chainId uint64) bool {
	for _, validId := range chainIds {
		if chainId == validId {
			return true
		}
	}
	return false
}

func GetChainName(chainId uint64) string {
	return chainIdToName[chainId]
}

func IsTestnet(chainId uint64) bool {
	return chainId == 1442
}

func IsDevnet(chainId uint64) bool {
	return chainId == 1440
}

func CheckForkOrder() error {
	return nil
}
