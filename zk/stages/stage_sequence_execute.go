package stages

import (
	"context"
	"fmt"
	"time"

	"github.com/gateway-fm/cdk-erigon-lib/common"
	"github.com/gateway-fm/cdk-erigon-lib/kv"
	"github.com/ledgerwatch/log/v3"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ledgerwatch/erigon/common/math"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/core/vm"
	"github.com/ledgerwatch/erigon/eth/ethconfig"
	"github.com/ledgerwatch/erigon/eth/stagedsync"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
)

func SpawnSequencingStage(
	s *stagedsync.StageState,
	u stagedsync.Unwinder,
	tx kv.RwTx,
	toBlock uint64,
	ctx context.Context,
	cfg SequenceBlockCfg,
	initialCycle bool,
	quiet bool,
) (err error) {
	logPrefix := s.LogPrefix()
	log.Info(fmt.Sprintf("[%s] Starting sequencing stage", logPrefix))
	defer log.Info(fmt.Sprintf("[%s] Finished sequencing stage", logPrefix))

	freshTx := tx == nil
	if freshTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	sdb := newStageDb(tx)

	executionAt, err := s.ExecutionAt(tx)
	if err != nil {
		return err
	}

	lastBatch, err := stages.GetStageProgress(tx, stages.HighestSeenBatchNumber)
	if err != nil {
		return err
	}

	forkId, err := prepareForkId(cfg, lastBatch, executionAt, sdb.hermezDb)
	if err != nil {
		return err
	}

	if err := stagedsync.UpdateZkEVMBlockCfg(cfg.chainConfig, sdb.hermezDb, logPrefix); err != nil {
		return err
	}

	// injected batch
	if executionAt == 0 {
		header, parentBlock, err := prepareHeader(tx, executionAt, forkId, cfg.zk.AddressSequencer)
		if err != nil {
			return err
		}

		err = processInjectedInitialBatch(ctx, cfg, s, sdb, forkId, header, parentBlock)
		if err != nil {
			return err
		}
	} else {
		thisBatch := lastBatch + 1
		batchTicker := time.NewTicker(cfg.zk.SequencerBatchSealTime)
		batchCounters := vm.NewBatchCounterCollector(sdb.smt.GetDepth(), uint16(forkId))
		overflow := false
		runLoopBlocks := true
		yielded := mapset.NewSet[[32]byte]()

		log.Info(fmt.Sprintf("[%s] Starting batch %d...", logPrefix, thisBatch))

		for bn := executionAt; runLoopBlocks; bn++ {
			log.Info(fmt.Sprintf("[%s] Starting block %d...", logPrefix, bn+1))
			header, parentBlock, err := prepareHeader(tx, bn, forkId, cfg.zk.AddressSequencer)
			if err != nil {
				return err
			}

			// start waiting for a new transaction to arrive
			log.Info(fmt.Sprintf("[%s] Waiting for txs from the pool...", logPrefix))

			var addedTransactions []types.Transaction
			var addedReceipts []*types.Receipt
			lastTxTime := time.Now()
			ibs := state.New(sdb.stateReader)

			logTicker := time.NewTicker(10 * time.Second)
			blockTicker := time.NewTicker(cfg.zk.SequencerBlockSealTime)

			// whilst in the 1 batch = 1 block = 1 tx flow we can immediately add in the changeL2BlockTx calculation
			// as this is the first tx we can skip the overflow check
			batchCounters.StartNewBlock()

			// calculate and store the l1 info tree index used for this block
			l1TreeUpdateIndex, l1TreeUpdate, err := calculateNextL1TreeUpdateToUse(tx, sdb.hermezDb)
			if err != nil {
				return err
			}

			l1BlockHash := common.Hash{}
			ger := common.Hash{}
			if l1TreeUpdate != nil {
				l1BlockHash = l1TreeUpdate.ParentHash
				ger = l1TreeUpdate.GER
			}

			parentRoot := parentBlock.Root()
			if err = handleStateForNewBlockStarting(cfg.chainConfig, sdb.hermezDb, ibs, bn+1, header.Time, &parentRoot, l1TreeUpdate); err != nil {
				return err
			}

			// start to wait for transactions to come in from the pool and attempt to add them to the current batch.  Once we detect a counter
			// overflow we revert the IBS back to the previous snapshot and don't add the transaction/receipt to the collection that will
			// end up in the finalised block
		LOOP_TRANSACTIONS:
			for {
				select {
				case <-logTicker.C:
					log.Info(fmt.Sprintf("[%s] Waiting some more for txs from the pool...", logPrefix))
				case <-blockTicker.C:
					// runLoopBlocks = false
					break LOOP_TRANSACTIONS
				case <-batchTicker.C:
					runLoopBlocks = false
					break LOOP_TRANSACTIONS
				default:
					cfg.txPool.LockFlusher()
					transactions, err := getNextTransactions(cfg, executionAt, forkId, yielded)
					if err != nil {
						return err
					}
					cfg.txPool.UnlockFlusher()

					for _, transaction := range transactions {
						var receipt *types.Receipt
						receipt, overflow, err = attemptAddTransaction(cfg, sdb, ibs, batchCounters, header, parentBlock.Header(), transaction)
						if err != nil {
							return err
						}
						if overflow {
							log.Info(fmt.Sprintf("[%s] overflowed adding transaction to batch", logPrefix), "batch", thisBatch, "tx-hash", transaction.Hash())
							panic("CURRENT IMPLEMENTATION DOES NOT WORK")
							break LOOP_TRANSACTIONS
						}

						addedTransactions = append(addedTransactions, transaction)
						addedReceipts = append(addedReceipts, receipt)
					}

					// if there were no transactions in this check, and we have some transactions to process, and we've waited long enough for
					// more to arrive then close the batch
					timeNow := time.Now()
					sinceLastTx := timeNow.Sub(lastTxTime)
					if len(transactions) > 0 {
						lastTxTime = timeNow
					} else if len(addedTransactions) > 0 && sinceLastTx > 250*time.Millisecond {
						// log.Info(fmt.Sprintf("[%s] No new transactions, closing block at %v transactions", logPrefix, len(addedTransactions)))
						// break LOOP_TRANSACTIONS
					}
				}
			}

			// todo: can we handle this scenario without needing to re-process the transactions?  We're doing this currently because the IBS can't be reverted once a tx has been
			// finalised within it - it causes a panic
			if overflow {
				panic("CURRENT IMPLEMENTATION DOES NOT WORK")
				// we know now that we have a list of good transactions, so we need to get a fresh intra block state and re-run the known good ones
				// before continuing on
				batchCounters.ClearTransactionCounters()
				ibs = state.New(sdb.stateReader)

				// it was incremented before, so needs resetting here
				header.GasUsed = 0

				for idx, transaction := range addedTransactions {
					receipt, innerOverflow, err := attemptAddTransaction(cfg, sdb, ibs, batchCounters, header, parentBlock.Header(), transaction)
					if err != nil {
						return err
					}
					if innerOverflow {
						// kill the node at this stage to prevent a batch being created that can't be proven
						panic("overflowed twice during execution")
					}
					addedReceipts[idx] = receipt
				}
			}

			if err = sdb.hermezDb.WriteBlockL1InfoTreeIndex(bn+1, l1TreeUpdateIndex); err != nil {
				return err
			}

			if err = finaliseBlock(ctx, cfg, s, sdb, ibs, header, parentBlock, forkId, thisBatch, ger, l1BlockHash, addedTransactions, addedReceipts); err != nil {
				return err
			}

			if err = updateSequencerProgress(tx, bn+1, thisBatch, l1TreeUpdateIndex); err != nil {
				return err
			}

			if cfg.accumulator != nil {
				txs, err := rawdb.RawTransactionsRange(tx, header.Number.Uint64(), header.Number.Uint64())
				if err != nil {
					return err
				}
				cfg.accumulator.StartChange(header.Number.Uint64(), header.Hash(), txs, false)
			}

			log.Info(fmt.Sprintf("[%s] Finish block %d with %d transactions...", logPrefix, bn+1, len(addedTransactions)))
		}

		counters, err := batchCounters.CombineCollectors()
		if err != nil {
			return err
		}

		log.Info("counters consumed", "counts", counters.UsedAsString())
		err = sdb.hermezDb.WriteBatchCounters(thisBatch, counters.UsedAsMap())
		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("[%s] Finish batch %d...", logPrefix, thisBatch))
	}

	if freshTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func PruneSequenceExecutionStage(s *stagedsync.PruneState, tx kv.RwTx, cfg SequenceBlockCfg, ctx context.Context, initialCycle bool) (err error) {
	logPrefix := s.LogPrefix()
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	logEvery := time.NewTicker(logInterval)
	defer logEvery.Stop()

	if cfg.historyV3 {
		cfg.agg.SetTx(tx)
		if initialCycle {
			if err = cfg.agg.Prune(ctx, ethconfig.HistoryV3AggregationStep/10); err != nil { // prune part of retired data, before commit
				return err
			}
		} else {
			if err = cfg.agg.PruneWithTiemout(ctx, 1*time.Second); err != nil { // prune part of retired data, before commit
				return err
			}
		}
	} else {
		if cfg.prune.History.Enabled() {
			if err = rawdb.PruneTableDupSort(tx, kv.AccountChangeSet, logPrefix, cfg.prune.History.PruneTo(s.ForwardProgress), logEvery, ctx); err != nil {
				return err
			}
			if err = rawdb.PruneTableDupSort(tx, kv.StorageChangeSet, logPrefix, cfg.prune.History.PruneTo(s.ForwardProgress), logEvery, ctx); err != nil {
				return err
			}
		}

		if cfg.prune.Receipts.Enabled() {
			if err = rawdb.PruneTable(tx, kv.Receipts, cfg.prune.Receipts.PruneTo(s.ForwardProgress), ctx, math.MaxInt32); err != nil {
				return err
			}
			if err = rawdb.PruneTable(tx, kv.BorReceipts, cfg.prune.Receipts.PruneTo(s.ForwardProgress), ctx, math.MaxUint32); err != nil {
				return err
			}
			// LogIndex.Prune will read everything what not pruned here
			if err = rawdb.PruneTable(tx, kv.Log, cfg.prune.Receipts.PruneTo(s.ForwardProgress), ctx, math.MaxInt32); err != nil {
				return err
			}
		}
		if cfg.prune.CallTraces.Enabled() {
			if err = rawdb.PruneTableDupSort(tx, kv.CallTraceSet, logPrefix, cfg.prune.CallTraces.PruneTo(s.ForwardProgress), logEvery, ctx); err != nil {
				return err
			}
		}
	}

	if err = s.Done(tx); err != nil {
		return err
	}
	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
