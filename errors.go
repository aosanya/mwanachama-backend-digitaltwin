package digitaltwin

import "errors"

var ErrInvalid = errors.New("digitaltwin: invalid")

var ErrNodeNotFound = errors.New("digitaltwin: node not found")

var ErrLinkNotFound = errors.New("digitaltwin: link not found")

var ErrInvalidLink = errors.New("digitaltwin: invalid link")

var ErrMetricNotFound = errors.New("digitaltwin: metric not found")

var ErrRetentionNotFound = errors.New("digitaltwin: retention rule not found")

var ErrObservationNotFound = errors.New("digitaltwin: no observation found")

var ErrUnknownMetric = errors.New("digitaltwin: metric not registered")
