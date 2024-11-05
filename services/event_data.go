package services

import (
	"math/big"
	"seahorsefi-test/pkg"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type (
	RedeemEvent struct {
		Redeemer     common.Address
		RedeemAmount *big.Int
		RedeemTokens *big.Int
	}
	BorrowEvent struct {
		Borrower       common.Address
		BorrowAmount   *big.Int
		AccountBorrows *big.Int
		TotalBorrows   *big.Int
	}
	RepayBorrowEvent struct {
		Payer          common.Address
		Borrower       common.Address
		RepayAmount    *big.Int
		AccountBorrows *big.Int
		TotalBorrows   *big.Int
	}
	MintEvent struct {
		Minter common.Address
		MintAmount *big.Int
		MintTokens *big.Int
	}
)

func (s *Service) parseRedeemData(vLog types.Log) (RedeemEvent, error) {
	var event RedeemEvent
	switch vLog.Address {
	case common.HexToAddress(pkg.ETH_ADDRESS):
		err := s.ethAbi.UnpackIntoInterface(&event, "Redeem", vLog.Data)
		if err != nil {
			return event, err
		}
	case common.HexToAddress(pkg.USDC_ADDRESS):
		err := s.usdcAbi.UnpackIntoInterface(&event, "Redeem", vLog.Data)
		if err != nil {
			return event, err
		}
	}

	return event, nil
}

func (s *Service) parseBorrowData(vLog types.Log) (BorrowEvent, error) {
	var event BorrowEvent
	switch vLog.Address {
	case common.HexToAddress(pkg.ETH_ADDRESS):
		err := s.ethAbi.UnpackIntoInterface(&event, "Borrow", vLog.Data)
		if err != nil {
			return event, err
		}
	case common.HexToAddress(pkg.USDC_ADDRESS):
		err := s.usdcAbi.UnpackIntoInterface(&event, "Borrow", vLog.Data)
		if err != nil {
			return event, err
		}
	}

	return event, nil
}

func (s *Service) parseRepayBorrowData(vLog types.Log) (RepayBorrowEvent, error) {
	var event RepayBorrowEvent
	switch vLog.Address {
	case common.HexToAddress(pkg.ETH_ADDRESS):
		err := s.ethAbi.UnpackIntoInterface(&event, "RepayBorrow", vLog.Data)
		if err != nil {
			return event, err
		}
	case common.HexToAddress(pkg.USDC_ADDRESS):
		err := s.usdcAbi.UnpackIntoInterface(&event, "RepayBorrow", vLog.Data)
		if err != nil {
			return event, err
		}
	}

	return event, nil
}
func (s *Service) parseMintData(vLog types.Log) (MintEvent, error) {
	var event MintEvent
	switch vLog.Address {
	case common.HexToAddress(pkg.ETH_ADDRESS):
		err := s.ethAbi.UnpackIntoInterface(&event, "Mint", vLog.Data)
		if err != nil {
			return event, err
		}
	case common.HexToAddress(pkg.USDC_ADDRESS):
		err := s.usdcAbi.UnpackIntoInterface(&event, "Mint", vLog.Data)
		if err != nil {
			return event, err
		}
	}

	return event, nil
}
