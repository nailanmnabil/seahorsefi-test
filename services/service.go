package services

import (
	"context"
	"log"
	"math/big"
	"seahorsefi-test/entities"
	"seahorsefi-test/pkg"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	ethClient *ethclient.Client
	dbConn    *gorm.DB
	ethAbi    abi.ABI
	usdcAbi   abi.ABI

	poolAndCalculateMutex sync.Mutex
}

func NewService(ethClient *ethclient.Client, dbConn *gorm.DB, ethAbi abi.ABI, usdcAbi abi.ABI) *Service {
	return &Service{
		ethClient: ethClient,
		dbConn:    dbConn,
		ethAbi:    ethAbi,
		usdcAbi:   usdcAbi,
	}
}

func (s *Service) PoolAndCalculate() {
	s.poolAndCalculateMutex.Lock()
	defer s.poolAndCalculateMutex.Unlock()

	now := time.Now().UTC()

	s.PoolEvent()
	s.PointCalculator(now)
}

func (s *Service) GetCurrentBlockIfEmpty() {
	var eventCount int64
	err := s.dbConn.Model(&entities.Event{}).Count(&eventCount).Error
	if err != nil {
		log.Fatalf("failed to count event: %v\n", err)
	}

	if eventCount != 0 {
		return
	}

	query := ethereum.FilterQuery{
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

	ctx := context.Background()

	// get latest block
	latestBlock, err := s.ethClient.BlockNumber(context.Background())
	if err != nil {
		log.Fatalf("failed to get latest block: %v\n", err)
	}

	var blockRange uint64 = 100000
	var logs []types.Log

	// grow block range until we find block
	for len(logs) == 0 {
		query.FromBlock = big.NewInt(int64(latestBlock - blockRange))
		query.ToBlock = big.NewInt(int64(latestBlock))

		logs, err = s.ethClient.FilterLogs(ctx, query)
		if err != nil {
			log.Fatalf("failed to filter logs: %v\n", err)
		}

		blockRange += blockRange
	}

	// find log that have largest number
	var latestLog *types.Log
	for i := range logs {
		if latestLog == nil || logs[i].BlockNumber > latestLog.BlockNumber {
			latestLog = &logs[i]
		}
	}

	event, walletAddr, err := s.parseEvent(ctx, *latestLog)
	if err != nil {
		log.Fatalf("failed to parse event: %v\n", err)
	}

	s.dbConn.Transaction(func(tx *gorm.DB) error {
		walletId := uuid.NewString()
		err = tx.Create(&entities.Wallet{
			ID:      walletId,
			Address: walletAddr,
		}).Error
		if err != nil {
			return err
		}

		event.WalletID = walletId
		err = tx.Create(&event).Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Fatalf("transaction failed: %v\n", err)
	}
}
