package race

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"wwfc/common"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
)

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	common.ReadGameList()

	// Start SQL
	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	if err != nil {
		panic(err)
	}

	pool, err = pgxpool.ConnectConfig(ctx, dbConf)
	if err != nil {
		panic(err)
	}
}

func Shutdown() {
}

func HandleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	logging.Info("RACE", aurora.Yellow(request.Method), aurora.Cyan(request.URL), "via", aurora.Cyan(request.Host), "from", aurora.BrightCyan(request.RemoteAddr))

	switch {
	case strings.HasSuffix(request.URL.Path, "NintendoRacingService.asmx"):
		moduleName := "RACE:RacingService:" + request.RemoteAddr
		handleNintendoRacingServiceRequest(moduleName, responseWriter, request)
	}
}
