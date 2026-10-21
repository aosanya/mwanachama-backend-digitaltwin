package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	digitaltwin "github.com/aosanya/mwanachama-backend-digitaltwin"
)

type Tool = dispatch.Tool

func BuildTools(tm digitaltwin.TwinManager, m Mount) ([]Tool, error) {
	s, err := dispatch.Parse(digitaltwin.Operations())
	if err != nil {
		return nil, err
	}
	return dispatch.Tools(s, dispatch.Deps{
		Manager: tm, Errors: Sentinels, Authorize: m.Authorize, Caller: m.Caller, Device: m.Device,
	})
}
