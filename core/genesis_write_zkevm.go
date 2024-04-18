package core

import (
	"math/big"

	libcommon "github.com/gateway-fm/cdk-erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/smt/pkg/smt"
	"github.com/ledgerwatch/erigon/zkevm/hex"
)

func HermezMainnetGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.HermezMainnetChainConfig,
		Timestamp:  1679653163,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/hermez.json"),
	}
}

func HermezMainnetShadowforkGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.HermezMainnetChainConfig,
		Timestamp:  1679653163,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/hermez-shadowfork.json"),
	}
}

func HermezEtrogGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.HermezEtrogChainConfig,
		Timestamp:  1703260380,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/hermez-etrog.json"),
	}
}

func HermezCardonaGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.HermezCardonaChainConfig,
		Timestamp:  1701262224,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/hermez-cardona.json"),
	}
}

func HermezCardonaInternalGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.HermezBaliChainConfig,
		Timestamp:  1701336708,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/hermez-bali.json"),
	}
}

func HermezLocalDevnetGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.HermezLocalDevnetChainConfig,
		Timestamp:  1706732232,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/hermez-dev.json"),
	}
}

func HermezESTestGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.HermezESTestChainConfig,
		Timestamp:  1710763452,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/hermez-estest.json"),
	}
}

func X1TestnetGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.X1TestnetChainConfig,
		Timestamp:  1699369668,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/x1-testnet.json"),
	}
}

func LomuzPrivateChainLayer2GenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.LumozPrivateTestnetLayer2Config,
		Timestamp:  1712317293,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/lumoz-private-chain-layer2.json"),
	}
}

func MerlinV2TestnetGenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.MerlinV2TestnetConfig,
		Timestamp:  1712033416,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/merlin-v2-testnet.json"),
	}
}

func ZkevmFork58GenesisBlock() *types.Genesis {
	return &types.Genesis{
		Config:     params.ZkevmFor58Config,
		Timestamp:  1712475797,
		GasLimit:   0x0,
		Difficulty: big.NewInt(0x0),
		Alloc:      readPrealloc("allocs/zkevm-fork5-8.json"),
	}
}

func processAccount(s *smt.SMT, root *big.Int, a *types.GenesisAccount, addr libcommon.Address) (*big.Int, error) {

	// store the account balance and nonce
	r, err := s.SetAccountState(addr.String(), a.Balance, new(big.Int).SetUint64(a.Nonce))
	if err != nil {
		return nil, err
	}

	if len(a.Code) > 0 {
		xs := hex.EncodeToString(a.Code)
		err = s.SetContractBytecode(addr.String(), xs)
		if err != nil {
			return nil, err
		}
	}

	// parse the storage into map[string]string by splitting the storage hex into two 32 bit values
	sm := make(map[string]string)
	for k, v := range a.Storage {
		sm[k.String()] = v.String()
	}

	// store the account storage
	if len(sm) > 0 {
		r, err = s.SetContractStorage(addr.String(), sm, nil)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}
