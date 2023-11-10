package nas

import (
	"encoding/base64"
	"github.com/logrusorgru/aurora/v3"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"wwfc/logging"
	"wwfc/sake"
)

type Route struct {
	Actions []Action
}

// Action contains information about how a specified action should be handled.
type Action struct {
	ActionName  string
	Callback    func(*Response, map[string]string) map[string]string
	ServiceType string
}

func NewRoute() Route {
	return Route{}
}

// RoutingGroup defines a group of actions for a given service type.
type RoutingGroup struct {
	Route       *Route
	ServiceType string
}

// HandleGroup returns a routing group type for the given service type.
func (route *Route) HandleGroup(serviceType string) RoutingGroup {
	return RoutingGroup{
		Route:       route,
		ServiceType: serviceType,
	}
}

func (r *RoutingGroup) HandleAction(action string, function func(*Response, map[string]string) map[string]string) {
	r.Route.Actions = append(r.Route.Actions, Action{
		ActionName:  action,
		Callback:    function,
		ServiceType: r.ServiceType,
	})
}

var (
	regexSakeURL = regexp.MustCompile(`^([a-z\-]+\.)?sake\.gs\.`)
)

func (route *Route) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Move this to its own server
		// Check for *.sake.gs.* or sake.gs.*
		if regexSakeURL.MatchString(r.Host) {
			// Redirect to the sake server
			sake.HandleRequest(w, r)
			return
		}

		logging.Notice("NAS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
		moduleName := "NAS:" + r.RemoteAddr

		// TODO: Move this to its own server
		// Check for /payload
		if strings.HasPrefix(r.URL.String(), "/payload") {
			handlePayloadRequest(w, r)
			return
		}

		err := r.ParseForm()
		if err != nil {
			logging.Error(moduleName, "Failed to parse form")
			return
		}

		if !strings.HasPrefix(r.URL.Path, "/") {
			logging.Error(moduleName, "Invalid URL")
			return
		}

		path := r.URL.Path[1:]

		fields := map[string]string{}
		for key, values := range r.PostForm {
			if len(values) != 1 {
				logging.Warn(moduleName, "Ignoring multiple POST form values:", aurora.Cyan(key).String()+":", aurora.Cyan(values))
				continue
			}

			parsed, err := base64.StdEncoding.DecodeString(strings.Replace(values[0], "*", "=", -1))
			if err != nil {
				logging.Error(moduleName, "Invalid POST form value:", aurora.Cyan(key).String()+":", aurora.Cyan(values[0]))
				return
			}
			logging.Info(moduleName, aurora.Cyan(key).String()+":", aurora.Cyan(string(parsed)))
			fields[key] = string(parsed)
		}

		actionName, ok := fields["action"]
		if !ok || actionName == "" {
			logging.Error(moduleName, "No action in form")
			return
		}

		var action Action
		for _, _action := range route.Actions {
			if path == _action.ServiceType && actionName == _action.ActionName {
				action = _action
			}
		}

		// Make sure we found an action
		if action.ActionName == "" && action.ServiceType == "" {
			logging.Error(moduleName, "No action for", aurora.Cyan(actionName))
			return
		}

		response := NewResponse(&w, r)
		reply := action.Callback(response, fields)

		if len(reply) != 0 {
			param := url.Values{}
			for key, value := range reply {
				param.Set(key, strings.Replace(base64.StdEncoding.EncodeToString([]byte(value)), "=", "*", -1))
			}
			response.payload = []byte(param.Encode())
			response.payload = []byte(strings.Replace(string(response.payload), "%2A", "*", -1))
			// DWC treats the response like a null terminated string
			response.payload = append(response.payload, 0x00)
		}

		w.Header().Set("NODE", "wifiappe1")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(len(response.payload)))
		w.Write(response.payload)
	})
}

func NewResponse(w *http.ResponseWriter, r *http.Request) *Response {
	return &Response{
		request: r,
		writer:  w,
	}
}
