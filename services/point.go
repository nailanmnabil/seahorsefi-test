package services

import (
	"encoding/json"
	"net/http"
	"seahorsefi-test/entities"
)

// GetWalletPoint handles the get wallet point request
// @Summary Get all wallets point
// @Description Get all wallets point
// @Tags Wallet
// @Accept json
// @Produce json
// @Success 200 {array} entities.Wallet "Successful response with list of wallets"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /wallets/points [get]
func (s *Service) GetWalletPoint(w http.ResponseWriter, r *http.Request) {
	wallets := make([]entities.Wallet, 0)
	err := s.dbConn.Preload("Events").Find(&wallets).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(wallets)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}
