package main

import (
	"log"
	"log/slog"
	"net/http"
	"seahorsefi-test/pkg"
	"seahorsefi-test/services"
	"strings"

	_ "seahorsefi-test/docs"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robfig/cron/v3"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title SeahorseFi API Doc
// @version 1.0
// @description This is a documentation of SeahorseFi off-chain point tracker server.
// @termsOfService http://swagger.io/terms/
// @BasePath /
func main() {
	dbStringConn := pkg.GetEnv("DB_STRING_CONN", "")
	dbMinConn := pkg.GetEnv("DB_MIN_CONN", "3")
	dbMaxConn := pkg.GetEnv("DB_MAX_CONN", "10")
	infuraURL := pkg.GetEnv("INFURA_URL", "")

	client, err := ethclient.Dial(infuraURL)
	if err != nil {
		log.Fatalf("failed to connect to the Ethereum client: %v\n", err)
	}

	dbConn := pkg.GetDbConn(dbStringConn, dbMinConn, dbMaxConn)

	cron := cron.New()

	ethABI, err := abi.JSON(strings.NewReader(pkg.ETH_ABI))
	if err != nil {
		log.Fatalf("failed to parse eth ABI: %v\n", err)
	}

	usdcABI, err := abi.JSON(strings.NewReader(pkg.USDC_ABI))
	if err != nil {
		log.Fatalf("failed to parse usdc ABI: %v\n", err)
	}

	svc := services.NewService(client, dbConn, ethABI, usdcABI)
	svc.GetCurrentBlockIfEmpty()

	// start all scheduler
	cron.AddFunc("*/1 * * * *", svc.PoolAndCalculate)
	cron.Start()

	// start rest api
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/docs/doc.json"),
	))
	r.Get("/wallets/points", svc.GetWalletPoint)
	slog.Info("Server running on port 8080")
	http.ListenAndServe(":8080", r)
}
