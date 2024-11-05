package services

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"runtime/debug"
	"seahorsefi-test/entities"
	"seahorsefi-test/pkg"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	BATCH_SIZE = 500
)

func (s *Service) PoolEvent() {
	ctx := context.Background()
	schedulerID := uuid.NewString()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in PoolEvent scheduler", "scheduler-id", schedulerID, "recover", r, "stack", debug.Stack())
		}
	}()

	slog.Info("starting PoolEvent scheduler", "scheduler-id", schedulerID)

	// get latest block in the eth net
	latestBlock, err := s.ethClient.BlockNumber(ctx)
	if err != nil {
		slog.Error("failed to get latest block", "scheduler-id", schedulerID, "error", err)
	}

	var latestEvent entities.Event
	err = s.dbConn.Order("block_number desc").Limit(1).First(&latestEvent).Error
	if err != nil {
		slog.Error("failed to get latest event we have", "scheduler-id", schedulerID, "error", err)
	}

	currentBlock := latestEvent.BlockNumber + 1

	// start fetch and populate event until we reach the latest block
	for currentBlock < latestBlock {
		endBlock := currentBlock + BATCH_SIZE
		if endBlock > latestBlock {
			endBlock = latestBlock
		}

		slog.Info("start to fetch event", "start", currentBlock, "end", endBlock)

		// fetch new event batches
		logs, err := s.fetchEvents(ctx,
			big.NewInt(int64(currentBlock)),
			big.NewInt(int64(endBlock)),
		)
		if err != nil {
			slog.Error("failed to fetch event", "scheduler-id", schedulerID, "error", err)
		}

		if len(logs) == 0 {
			currentBlock = endBlock + 1
			time.Sleep(100 * time.Millisecond)
			continue
		}

		newWalletCheck := make(map[string]string) // to check if there is new wallet already append to newWallets slice, map for O(1)
		newWallets := make([]entities.Wallet, 0)
		newEvents := make([]entities.Event, 0)

		// process the logs that we get
		for _, vLog := range logs {
			event, walletAddr, err := s.parseEvent(ctx, vLog)
			if err != nil {
				slog.Error("failed to parse event", "scheduler-id", schedulerID, "error", err)
			}

			// find wallet by wallet address
			var wallet entities.Wallet
			err = s.dbConn.Where("address = ?", walletAddr).First(&wallet).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				slog.Error("failed to get wallet", "scheduler-id", schedulerID, "error", err)
			}

			walletID := wallet.ID

			// if its now found then create new one
			if err == gorm.ErrRecordNotFound {
				if _, ok := newWalletCheck[walletAddr]; !ok {
					walletID = uuid.NewString()
					wallet = entities.Wallet{
						ID:      walletID,
						Address: walletAddr,
					}
					newWallets = append(newWallets, wallet)
					newWalletCheck[walletAddr] = walletID
				} else {
					walletID = newWalletCheck[walletAddr]
				}
			}

			event.WalletID = walletID
			newEvents = append(newEvents, event)
		}

		// insert all new events and new wallets
		err = s.dbConn.Transaction(func(tx *gorm.DB) error {
			if len(newWallets) != 0 {
				err := tx.Create(&newWallets).Error
				if err != nil {
					return err
				}
			}

			if len(newEvents) != 0 {
				err = tx.Create(&newEvents).Error
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			slog.Error("failed to create new event and wallet", "scheduler-id", schedulerID, "error", err)
		}

		currentBlock = endBlock + 1
		time.Sleep(100 * time.Millisecond)
	}

	slog.Info("successfully run PoolEvent scheduler", "scheduler-id", schedulerID, "start block", latestEvent.BlockNumber, "final block", latestBlock)
}

func (s *Service) fetchEvents(ctx context.Context, fromBlock, toBlock *big.Int) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []common.Address{
			common.HexToAddress(pkg.ETH_ADDRESS),
			common.HexToAddress(pkg.USDC_ADDRESS),
		},
		Topics: [][]common.Hash{
			{
				common.HexToHash(pkg.MINT_EVENT_SIGNATURE),
				common.HexToHash(pkg.BORROW_EVENT_SIGNATURE),
				common.HexToHash(pkg.REPAY_BORROW_EVENT_SIGNATURE),
				common.HexToHash(pkg.REDEEM_EVENT_SIGNATURE),
			},
		},
	}

	logs, err := s.ethClient.FilterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to filter logs: %v", err)
	}

	return logs, nil
}

func (s *Service) parseEvent(ctx context.Context, vLog types.Log) (entities.Event, string, error) {
	if len(vLog.Topics) == 0 {
		return entities.Event{}, "", fmt.Errorf("no topics in log")
	}

	var event *abi.Event
	var err error
	var eventType string

	switch vLog.Address {
	case common.HexToAddress(pkg.ETH_ADDRESS):
		event, err = s.ethAbi.EventByID(vLog.Topics[0])
		if err != nil {
			return entities.Event{}, "", fmt.Errorf("failed to get event by ID: %v", err)
		}
	case common.HexToAddress(pkg.USDC_ADDRESS):
		event, err = s.ethAbi.EventByID(vLog.Topics[0])
		if err != nil {
			return entities.Event{}, "", fmt.Errorf("failed to get event by ID: %v", err)
		}
	}

	var walletAddr common.Address

	switch event.Name {
	case "Mint":
		eventType = pkg.MINT
		mintEvent, err := s.parseMintData(vLog)
		if err != nil {
			return entities.Event{}, "", err
		}
		walletAddr = mintEvent.Minter
	case "Redeem":
		eventType = pkg.REDEEM
		redeemEvent, err := s.parseRedeemData(vLog)
		if err != nil {
			return entities.Event{}, "", err
		}
		walletAddr = redeemEvent.Redeemer
	case "Borrow":
		eventType = pkg.BORROW
		borrowEvent, err := s.parseBorrowData(vLog)
		if err != nil {
			return entities.Event{}, "", err
		}
		walletAddr = borrowEvent.Borrower
	case "RepayBorrow":
		eventType = pkg.REPAY_BORROW
		repayBorrowEvent, err := s.parseRepayBorrowData(vLog)
		if err != nil {
			return entities.Event{}, "", err
		}
		walletAddr = repayBorrowEvent.Borrower
	}

	block, err := s.ethClient.BlockByNumber(ctx, big.NewInt(int64(vLog.BlockNumber)))
	if err != nil {
		return entities.Event{}, "", fmt.Errorf("failed to get block by number: %v", err)
	}

	return entities.Event{
		Address:          vLog.Address.String(),
		TransactionHash:  vLog.TxHash.String(),
		BlockNumber:      vLog.BlockNumber,
		EventType:        eventType,
		CreatedAt:        time.Unix(int64(block.Time()), 0),
		LastCalculatedAt: time.Unix(int64(block.Time()), 0),
	}, walletAddr.String(), nil
}
