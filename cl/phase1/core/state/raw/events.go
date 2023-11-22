package raw

import (
	"github.com/ledgerwatch/erigon/cl/cltypes/solid"
)

type Events struct {
	OnRandaoMixChange                        func(index int, mix [32]byte)
	OnNewValidator                           func(index int, v solid.Validator, balance uint64) error
	OnNewValidatorBalance                    func(index int, balance uint64) error
	OnNewValidatorEffectiveBalance           func(index int, balance uint64) error
	OnNewValidatorActivationEpoch            func(index int, epoch uint64) error
	OnNewValidatorExitEpoch                  func(index int, epoch uint64) error
	OnNewValidatorWithdrawableEpoch          func(index int, epoch uint64) error
	OnNewValidatorSlashed                    func(index int, slashed bool) error
	OnNewValidatorActivationEligibilityEpoch func(index int, epoch uint64) error
	OnNewValidatorWithdrawalCredentials      func(index int, wc []byte) error
}