package services

import (
	"log/slog"
	"math"
	"runtime/debug"
	"seahorsefi-test/entities"
	"seahorsefi-test/pkg"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ActiveEvent struct {
	Event           entities.Event
	AllEventID      []int
	CloseEvent      entities.Event
	AllCloseEventID []int
	Multiplier      int
}

func (s *Service) PointCalculator(now time.Time) {
	schedulerID := uuid.NewString()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in PointCalculator scheduler", "scheduler-id", schedulerID, "recover", r, "stack", debug.Stack())
		}
	}()

	slog.Info("starting PointCalculator scheduler", "scheduler-id", schedulerID)

	err := s.dbConn.Transaction(func(tx *gorm.DB) error {
		// get all active MINT and BORROW events that need point calculation
		activeEvents, err := s.getActiveEvent(now)
		if err != nil {
			return err
		}

		for _, activeEvent := range activeEvents {
			var closedAt *time.Time
			calculateUntil := now

			// if closing event exists, calculate points only until that time
			if activeEvent.CloseEvent.ID != 0 {
				calculateUntil = activeEvent.CloseEvent.CreatedAt
				closedAt = &activeEvent.CloseEvent.CreatedAt
			}

			// calculate time difference in 10-minute intervals
			timeDiff := calculateUntil.Sub(activeEvent.Event.LastCalculatedAt).Minutes()
			intervals := int(math.Floor(timeDiff / 10))

			if intervals > 0 {
				// calculate and update wallet points
				pointsToAdd := intervals * activeEvent.Multiplier
				err = tx.Model(&entities.Wallet{}).Where("id = ?", activeEvent.Event.WalletID).
					Update("points", gorm.Expr("points + ?", pointsToAdd)).Error
				if err != nil {
					return err
				}

				// update event's last calculation time
				lastCalculatedAt := activeEvent.Event.LastCalculatedAt.Add(time.Duration(intervals) * 10 * time.Minute)
				err = tx.Model(&entities.Event{}).Where("id IN ?", activeEvent.AllEventID).
					Update("last_calculated_at", lastCalculatedAt).Error
				if err != nil {
					return err
				}
			}

			if closedAt != nil {
				// update active event's closed status
				err = tx.Model(&entities.Event{}).Where("id in (?)", activeEvent.AllEventID).
					Update("closed_at", *closedAt).Error
				if err != nil {
					return err
				}

				// update closed event's closed status
				err = tx.Model(&entities.Event{}).Where("id in (?)", activeEvent.AllCloseEventID).
					Update("closed_at", gorm.Expr("created_at")).Error
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("transaction failed in events processing", "scheduler-id", schedulerID, "error", err)
		return
	}

	slog.Info("successfully completed PointCalculator scheduler", "scheduler-id", schedulerID)
}

func (s *Service) getActiveEvent(now time.Time) ([]ActiveEvent, error) {
	var allEvents []entities.Event
	var activeEvents []ActiveEvent

	err := s.dbConn.Clauses(clause.Locking{Strength: "UPDATE"}).Where("last_calculated_at < ? AND event_type IN ('MINT', 'BORROW') AND closed_at IS NULL", now).
		Find(&allEvents).Error
	if err != nil {
		return nil, err
	}

	// close event -> active block number -> active event
	closeEventToEvents := make(map[entities.Event]map[uint64]entities.Event)
	// wallet id -> event type -> event address -> active block number -> active event
	unclosedEvents := make(map[string]map[string]map[string]map[uint64]entities.Event)

	// looking for already closed or unclosed event
	for _, event := range allEvents {
		closedEventType := pkg.REDEEM
		if event.EventType == pkg.BORROW {
			closedEventType = pkg.REPAY_BORROW
		}

		var closeEvent entities.Event
		err = s.dbConn.
			Where(`wallet_id = ? AND address = ? AND event_type = ? AND created_at > ? AND closed_at IS NULL`,
				event.WalletID, event.Address, closedEventType, event.CreatedAt,
			).Order("block_number ASC").First(&closeEvent).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}

		if err == gorm.ErrRecordNotFound {
			s.addUnclosedEvent(unclosedEvents, event)
			continue
		}

		if _, exists := closeEventToEvents[closeEvent]; !exists {
			closeEventToEvents[closeEvent] = map[uint64]entities.Event{event.BlockNumber: event}
		} else {
			closeEventToEvents[closeEvent][event.BlockNumber] = event
		}
	}

	// process unclosed event
	for _, typeAddress := range unclosedEvents {
		for _, addressEvents := range typeAddress {
			for _, blockNumberToEvent := range addressEvents {
				allEventID := make([]int, 0)

				var lowestBlockNumber uint64 = math.MaxUint64
				for _, event := range blockNumberToEvent {
					lowestBlockNumber = min(lowestBlockNumber, event.BlockNumber)
					allEventID = append(allEventID, event.ID)
				}

				activeEvents = append(activeEvents, ActiveEvent{
					Event:      blockNumberToEvent[lowestBlockNumber],
					AllEventID: allEventID,
				})
			}
		}
	}

	// process already closed event
	for closeEvent, blockNumberToEvent := range closeEventToEvents {
		allEventID := make([]int, 0)

		var lowestBlockNumber uint64 = math.MaxUint64
		for _, event := range blockNumberToEvent {
			lowestBlockNumber = min(lowestBlockNumber, event.BlockNumber)
			allEventID = append(allEventID, event.ID)
		}

		activeEvents = append(activeEvents, ActiveEvent{
			Event:      blockNumberToEvent[lowestBlockNumber],
			AllEventID: allEventID,
			CloseEvent: closeEvent,
		})
	}
	for i, activeEvent := range activeEvents {
		activeEvents[i].Multiplier = 1
		if activeEvents[i].Event.EventType == pkg.BORROW {
			activeEvents[i].Multiplier = 2
		}

		if activeEvent.CloseEvent.ID == 0 {
			continue
		}

		var allCloseEventID []int
		var eventsAfterClosing []entities.Event

		event := activeEvent.Event
		closeEvent := activeEvent.CloseEvent

		err = s.dbConn.Model(&entities.Event{}).
			Where("address = ? AND wallet_id = ? AND created_at >= ? AND (event_type = ? OR event_type = ?)",
				closeEvent.Address, closeEvent.WalletID, closeEvent.CreatedAt,
				closeEvent.EventType, event.EventType).
			Find(&eventsAfterClosing).Error
		if err != nil {
			return nil, err
		}

		for _, eventAfterClosing := range eventsAfterClosing {
			if eventAfterClosing.EventType == event.EventType {
				break
			}
			if eventAfterClosing.EventType == closeEvent.EventType {
				allCloseEventID = append(allCloseEventID, eventAfterClosing.ID)
			}
		}

		activeEvents[i].AllCloseEventID = make([]int, 0)
		activeEvents[i].AllCloseEventID = append(activeEvents[i].AllCloseEventID, allCloseEventID...)
	}

	return activeEvents, nil
}

func (s *Service) addUnclosedEvent(unclosedEvents map[string]map[string]map[string]map[uint64]entities.Event, event entities.Event) {
	if _, exists := unclosedEvents[event.WalletID]; !exists {
		unclosedEvents[event.WalletID] = make(map[string]map[string]map[uint64]entities.Event)
	}
	if _, exists := unclosedEvents[event.WalletID][event.EventType]; !exists {
		unclosedEvents[event.WalletID][event.EventType] = make(map[string]map[uint64]entities.Event)
	}
	if _, exists := unclosedEvents[event.WalletID][event.EventType][event.Address]; !exists {
		unclosedEvents[event.WalletID][event.EventType][event.Address] = make(map[uint64]entities.Event)
	}
	unclosedEvents[event.WalletID][event.EventType][event.Address][event.BlockNumber] = event
}
