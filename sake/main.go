package sake

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
	"net/http"
	"wwfc/common"
	"wwfc/logging"
)

var (
	ctx  = context.Background()
	pool *pgxpool.Pool
)

func StartServer() {
	// Get config
	config := common.GetConfig()

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

func HandleRequest(w http.ResponseWriter, r *http.Request) {
	logging.Notice("SAKE", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))

	switch r.URL.String() {
	case "/SakeStorageServer/StorageServer.asmx":
		moduleName := "SAKE:Storage:" + r.RemoteAddr
		handleStorageRequest(moduleName, w, r)
		break
	}
}
