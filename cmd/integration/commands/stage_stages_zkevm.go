package commands

import (
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/spf13/cobra"
	common2 "github.com/gateway-fm/cdk-erigon-lib/common"
	"github.com/ledgerwatch/erigon/core"
	erigoncli "github.com/ledgerwatch/erigon/turbo/cli"
	"github.com/gateway-fm/cdk-erigon-lib/kv"
	"github.com/ledgerwatch/erigon/eth/ethconfig"
	"context"
	"github.com/ledgerwatch/log/v3"
	"errors"
)

var stateStagesZk = &cobra.Command{
	Use: "state_stages_zkevm",
	Short: `Run all StateStages in loop.
Examples:
--unwind-batch-no=10  # unwind so the tip is the highest block in batch number 10
		`,
	Example: "go run ./cmd/integration state_stages_zkevm --config=... --verbosity=3 --unwind-batch-no=100",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, _ := common2.RootContext()
		//cfg := &nodecfg.DefaultConfig
		//utils.SetNodeConfigCobra(cmd, cfg)
		ethConfig := &ethconfig.Defaults
		ethConfig.Genesis = core.GenesisBlockByChainName(chain)
		erigoncli.ApplyFlagsForEthConfigCobra(cmd.Flags(), ethConfig)
		db := openDB(dbCfg(kv.ChainDB, chaindata), true)
		defer db.Close()

		if err := unwindZk(ctx, db); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Error(err.Error())
			}
			return
		}
	},
}

func init() {
	withConfig(stateStagesZk)
	withChain(stateStagesZk)
	withDataDir2(stateStagesZk)
	withUnwind(stateStagesZk)
	withUnwindBatchNo(stateStagesZk) // populates package global flag unwindBatchNo
	rootCmd.AddCommand(stateStagesZk)
}

// unwindZk unwinds to the batch number set in the unwindBatchNo flag (package global)
func unwindZk(ctx context.Context, db kv.RwDB) error {
	_, _, stateStages := newSyncZk(ctx, db)

	tx, err := db.BeginRw(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stateStages.DisableStages(stages.Snapshots)

	err = stateStages.UnwindToBatch(unwindBatchNo, tx)
	if err != nil {
		return err
	}

	err = stateStages.RunUnwind(db, tx)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	tx, err = db.BeginRw(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()

	return nil
}
