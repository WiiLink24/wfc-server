package nas

import (
	"encoding/base64"
	"github.com/logrusorgru/aurora/v3"
	"net/http"
	"strconv"
	"strings"
	"wwfc/logging"
)

type Route struct {
	Actions []Action
}

// Action contains information about how a specified action should be handled.
type Action struct {
	ActionName  string
	Callback    func(*Response)
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

func (r *RoutingGroup) HandleAction(action string, function func(*Response)) {
	r.Route.Actions = append(r.Route.Actions, Action{
		ActionName:  action,
		Callback:    function,
		ServiceType: r.ServiceType,
	})
}

func (route *Route) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logging.Notice("NAS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host))
		err := r.ParseForm()
		if err != nil {
			logging.Error("NAS", "Failed to parse form")
			return
		}

		if !strings.HasPrefix(r.URL.Path, "/") {
			logging.Error("NAS", "Invalid URL")
			return
		}

		path := r.URL.Path[1:]
		actionName, _ := base64.StdEncoding.DecodeString(strings.Replace(r.PostForm.Get("action"), "*", "=", -1))

		if string(actionName) == "" {
			logging.Error("NAS", "No action in form")
			return
		}

		var action Action
		for _, _action := range route.Actions {
			if path == _action.ServiceType && string(actionName) == _action.ActionName {
				action = _action
			}
		}

		// Make sure we found an action
		if action.ActionName == "" && action.ServiceType == "" {
			logging.Error("NAS", "No action for", aurora.Cyan(string(actionName)))
			return
		}

		response := NewResponse(&w, r)
		action.Callback(response)

		// Our callback function will already have formulated the needed response.
		// We will write common headers then the data.
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
