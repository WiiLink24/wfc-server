package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/logrusorgru/aurora/v3"
)

type RouteSpec struct {
	AllowedMethods []string
	RequiresSecret bool
	Path           string
	Exec           func(any, bool, *http.Request) (any, int, error)
	ReqType        reflect.Type
	ResType        reflect.Type
}

func MakeRouteSpec[T any, U any](
	requiresSecret bool,
	path string,
	exec func(any, bool, *http.Request) (any, int, error),
	allowedMethods ...string,
) RouteSpec {
	var zeroT [0]T
	var zeroU [0]U
	return RouteSpec{
		allowedMethods,
		requiresSecret,
		path,
		exec,
		reflect.TypeOf(zeroT).Elem(),
		reflect.TypeOf(zeroU).Elem(),
	}
}

type UserActionResponse struct {
	User    database.User
	Success bool
	Error   string
}

var (
	ctx       = context.Background()
	pool      *pgxpool.Pool
	apiSecret string
	routes    []RouteSpec

	ErrPostGetOnly          = errors.New("Incorrect request. POST and GET only.")
	ErrRequestBody          = errors.New("Unable to read request body")
	ErrInvalidSecret        = errors.New("Invalid API secret in request")
	ErrPIDMissing           = errors.New("Profile ID missing or 0 in request")
	ErrReason               = errors.New("Missing reason in request")
	ErrLength               = errors.New("Ban length missing or 0")
	ErrTransaction          = errors.New("Failed to complete database transaction")
	ErrUserQuery            = errors.New("Failed to find user in the database")
	ErrUserQueryTransaction = errors.New("Failed to find user in the database, but the intended transaction may have gone through")
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	apiSecret = config.APISecret

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

	err = database.HashInit(pool, ctx)
	if err != nil {
		panic(err)
	}

	routes = []RouteSpec{
		BanRoute,
		ClearRoute,
		GetHashRoute,
		KickRoute,
		MotdRoute,
		PinfoRoute,
		QueryRoute,
		RemoveHashRoute,
		SetHashRoute,
		UnbanRoute,
	}

	verifyRoutes(routes)
}

func verifyRoutes(routes []RouteSpec) {
	for _, route := range routes {
		if !route.RequiresSecret {
			logging.Info("API", "Registered route", aurora.Cyan(route.Path))
			continue
		}

		typ := route.ReqType
		field, found := typ.FieldByName("Secret")
		if !found || field.Type.Kind() != reflect.String {
			goto die
		}

		typ = route.ResType
		field, found = typ.FieldByName("Success")
		if !found || field.Type.Kind() != reflect.Bool {
			goto die
		}

		field, found = typ.FieldByName("Error")
		if !found || field.Type.Kind() != reflect.String {
			goto die
		}

		logging.Info("API", "Registered route", aurora.Cyan(route.Path))
		continue

	die:
		panic(fmt.Sprintf(
			"The %s in the RouteSpec for path %s is missing a(n) %s field of type %s!",
			typ.Name(),
			route.Path,
			field.Name,
			field.Type.Name(),
		))
	}
}

func HandleRequest(path string, w http.ResponseWriter, r *http.Request) bool {
	ret := false

	for _, route := range routes {
		if path != route.Path {
			continue
		}

		ret = true

		logstr := fmt.Sprintf(
			"%s request to '%s' from '%s':\n",
			aurora.Cyan(r.Method),
			aurora.Cyan(path),
			aurora.Cyan(r.RemoteAddr),
		)

		res, code, err := handleRequestInner(route, w, r, &logstr)
		// Is of type interface{}
		resV := reflect.ValueOf(&res)
		if resV.Kind() == reflect.Ptr {
			if resV.IsNil() {
				resV = reflect.New(route.ResType).Elem()
			} else {
				resV = resV.Elem()
			}
		}

		// Allocate new variable with the response type to operate on. Unable
		// to set fields on the original because it's not addressable
		resC := reflect.New(route.ResType).Elem()
		if resV.Elem().IsValid() {
			resC.Set(resV.Elem())
		}

		logstr += fmt.Sprintf("Response (%d, %s):\n", code, http.StatusText(code))

		// Just reflect the value in there because having each api set the
		// field in their own result would be annoying. Could've made an
		// interface but this was significantly less code than implementing
		// setters on every single result type.
		if err != nil {
			resC.FieldByName("Success").SetBool(false)
			resC.FieldByName("Error").SetString(err.Error())
		} else {
			resC.FieldByName("Success").SetBool(true)
		}

		resLen := route.ResType.NumField()
		if err != nil {
			logstr += fmt.Sprintf("Error: %v\n", err)
		} else if resLen > 2 {
			for i := 0; i < resLen; i++ {
				fieldType := route.ResType.Field(i)

				if fieldType.Name == "Success" || fieldType.Name == "Error" {
					continue
				}

				if fieldType.Type.Kind() == reflect.Ptr {
					logstr += fmt.Sprintf("%s: Unresolved Pointer", fieldType.Name)
					continue
				}

				var fieldInst reflect.Value
				if !resC.IsZero() {
					fieldInst = resC.Field(i)
				} else {
					fieldInst = reflect.Zero(route.ResType.Field(i).Type)
				}

				logstr += fmt.Sprintf("%s: %v\n", fieldType.Name, aurora.Cyan(fieldInst))
			}
		}

		logstr = strings.Trim(logstr, "\n ")

		if err != nil {
			logging.Error("API", logstr)
		} else {
			logging.Info("API", logstr)
		}

		jsonData, err := json.Marshal(resC.Interface())

		if err != nil {
			logging.Error("API", "Failed to marshal json!", aurora.Cyan(err))
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
		w.WriteHeader(code)
		w.Write(jsonData)
	}

	return ret
}

// The log message being a string pointer kinda sucks, but it was the least
// worst option while still keeping everything in one message. If I decided
// to just return the string, then the status code and error feel redundant.
func handleRequestInner(spec RouteSpec, w http.ResponseWriter, r *http.Request, msg *string) (any, int, error) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(spec.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return nil, http.StatusNoContent, nil
	}

	if !slices.Contains(spec.AllowedMethods, r.Method) {
		return nil,
			http.StatusMethodNotAllowed,
			fmt.Errorf("Incorrect request. %s only.", strings.Join(spec.AllowedMethods, " and "))
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Allow", strings.Join(spec.AllowedMethods, ", "))
		return nil, http.StatusBadRequest, ErrRequestBody
	}

	// I don't want to deserialize the request multiple times, so I'll just
	// deserialize it into the expected reqtype and reflect the secret out of
	// it...
	req := reflect.New(spec.ReqType)
	// Pointer needs to be convered to an interface so json.Unmarshal has the
	// correct type info about the request struct
	reqI := req.Interface()

	// Even for functions accepting GET alongside post, the req struct needs to
	// exist, so create the zero value and then only fill it if needed...
	if r.Method == http.MethodPost {
		err = json.Unmarshal(body, reqI)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}

		// Since reqI is copied, set req again
		req = reflect.ValueOf(reqI).Elem()

		for i := 0; i < spec.ReqType.NumField(); i++ {
			fieldInfo := spec.ReqType.Field(i)
			field := req.Field(i)
			*msg += fmt.Sprintf("%s: %v\n", fieldInfo.Name, aurora.Cyan(field))
		}
	}

	validSecret := false

	if spec.RequiresSecret {
		// We know the field exists
		secretField := req.FieldByName("Secret")
		secret := secretField.String()

		if apiSecret != "" && secret == apiSecret {
			validSecret = true
		} else {
			return nil, http.StatusForbidden, ErrInvalidSecret
		}
	}

	// Sometimes it's a pointer, sometimes it's not??
	// Motd is, but pinfo isn't... strange. Whatever
	if req.Kind() == reflect.Ptr {
		req = req.Elem()
	}

	return spec.Exec(req.Interface(), validSecret, r)
}

func handleUserAction[T any](req T, validSecret bool, f func(T, bool) (*database.User, int, error)) (UserActionResponse, int, error) {
	user, code, err := f(req, validSecret)

	if user == nil {
		user = &database.User{}
	}

	res := UserActionResponse{}
	res.User = *user

	return res, code, err
}

func Shutdown() {
}
