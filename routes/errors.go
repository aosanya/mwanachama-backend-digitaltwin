package routes

import (
	"errors"
	"net/http"

	"github.com/aosanya/mwanachama-backend-digitaltwin"
)

// digitaltwinStatusFor maps this package's sentinel errors to a status
// code — the same shape actor's routes package uses for its own errors.
func digitaltwinStatusFor(err error) int {
	switch {
	case errors.Is(err, digitaltwin.ErrAssetNotFound),
		errors.Is(err, digitaltwin.ErrConnectionNotFound),
		errors.Is(err, digitaltwin.ErrMetricDefinitionNotFound),
		errors.Is(err, digitaltwin.ErrRetentionPolicyNotFound),
		errors.Is(err, digitaltwin.ErrReadingNotFound):
		return http.StatusNotFound
	case errors.Is(err, digitaltwin.ErrInvalid),
		errors.Is(err, digitaltwin.ErrInvalidConnection),
		errors.Is(err, digitaltwin.ErrUnknownMetric):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeDigitaltwinErr(w http.ResponseWriter, err error) {
	code := digitaltwinStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}
