package routes

import (
	"fmt"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
)

type Route = httpwire.Route

var operations = sync.OnceValues(func() (*dispatch.Spec, error) {
	return dispatch.Parse(digitaltwin.Operations())
})

var sentinels = map[string]error{
	"ErrInvalid":             digitaltwin.ErrInvalid,
	"ErrInvalidLink":         digitaltwin.ErrInvalidLink,
	"ErrUnknownMetric":       digitaltwin.ErrUnknownMetric,
	"ErrNodeNotFound":        digitaltwin.ErrNodeNotFound,
	"ErrLinkNotFound":        digitaltwin.ErrLinkNotFound,
	"ErrMetricNotFound":      digitaltwin.ErrMetricNotFound,
	"ErrRetentionNotFound":   digitaltwin.ErrRetentionNotFound,
	"ErrObservationNotFound": digitaltwin.ErrObservationNotFound,
}

var AnonymousActions = []string{}

type Mount struct {
	Authorize dispatch.Authorizer
	Caller    dispatch.Caller
}

func Build(tm digitaltwin.TwinManager) ([]Route, error) { return BuildWith(tm, nil) }

func BuildWith(tm digitaltwin.TwinManager, authorize dispatch.Authorizer) ([]Route, error) {
	return BuildFor(tm, Mount{Authorize: authorize})
}

func BuildFor(tm digitaltwin.TwinManager, m Mount) ([]Route, error) {
	s, err := operations()
	if err != nil {
		return nil, err
	}
	return dispatch.Dispatch(s, dispatch.Deps{
		Manager: tm, Errors: sentinels, Authorize: m.Authorize, Caller: m.Caller,
	})
}

func Routes(tm digitaltwin.TwinManager) []Route { return RoutesWith(tm, nil) }

func RoutesWith(tm digitaltwin.TwinManager, authorize dispatch.Authorizer) []Route {
	return RoutesFor(tm, Mount{Authorize: authorize})
}

func RoutesFor(tm digitaltwin.TwinManager, m Mount) []Route {
	out, err := BuildFor(tm, m)
	if err != nil {
		panic(fmt.Sprintf("digitaltwin routes: %v", err))
	}
	return out
}

func Split(tm digitaltwin.TwinManager) dispatch.Split { return SplitWith(tm, nil) }

func SplitWith(tm digitaltwin.TwinManager, authorize dispatch.Authorizer) dispatch.Split {
	return SplitFor(tm, Mount{Authorize: authorize})
}

func SplitFor(tm digitaltwin.TwinManager, m Mount) dispatch.Split {
	public := dispatch.Anonymous(Routes(tm), AnonymousActions...)
	gated := dispatch.Anonymous(RoutesFor(tm, m), AnonymousActions...)
	return dispatch.Split{Anonymous: public.Anonymous, Gated: gated.Gated}
}

func PublicRoutes(tm digitaltwin.TwinManager) []Route { return Split(tm).Anonymous }

func OperatorRoutes(tm digitaltwin.TwinManager) []Route { return OperatorRoutesWith(tm, nil) }

func OperatorRoutesWith(tm digitaltwin.TwinManager, authorize dispatch.Authorizer) []Route {
	return OperatorRoutesFor(tm, Mount{Authorize: authorize})
}

func OperatorRoutesFor(tm digitaltwin.TwinManager, m Mount) []Route {
	return SplitFor(tm, m).Gated
}
