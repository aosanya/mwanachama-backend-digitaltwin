package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
)

type Route = httpwire.Route

type Mount = dispatch.Mount

var Sentinels = map[string]error{
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

var Table = dispatch.NewTable(digitaltwin.Operations(), Sentinels, AnonymousActions...)

func Build(tm digitaltwin.TwinManager) ([]Route, error) { return Table.Build(tm, Mount{}) }

func BuildFor(tm digitaltwin.TwinManager, m Mount) ([]Route, error) { return Table.Build(tm, m) }

func Routes(tm digitaltwin.TwinManager) []Route { return Table.Routes(tm, Mount{}) }

func RoutesFor(tm digitaltwin.TwinManager, m Mount) []Route { return Table.Routes(tm, m) }

func Split(tm digitaltwin.TwinManager, m Mount) dispatch.Split { return Table.Split(tm, m) }

func PublicRoutes(tm digitaltwin.TwinManager) []Route {
	return Table.Split(tm, Mount{}).Anonymous
}

func OperatorRoutes(tm digitaltwin.TwinManager, m Mount) []Route {
	return Table.Split(tm, m).Gated
}
