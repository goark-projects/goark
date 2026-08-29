package mvc

import (
	"errors"
	"net/http"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
)

type conditionDispatchHandler struct {
	registrations []routeRegistration
}

func newConditionDispatchHandler(registrations []routeRegistration) arkweb.Handler {
	return conditionDispatchHandler{registrations: append([]routeRegistration(nil), registrations...)}
}

func (h conditionDispatchHandler) Handle(ctx *arkweb.Context) (arkweb.Result, error) {
	bestIndex := -1
	bestScore := -1
	var selectedErr error
	for i := range h.registrations {
		registration := h.registrations[i]
		if err := registration.conditions.match(ctx); err != nil {
			selectedErr = selectConditionError(selectedErr, err)
			continue
		}
		score := registration.conditions.specificity()
		if bestIndex < 0 || score > bestScore {
			bestIndex = i
			bestScore = score
		}
	}
	if bestIndex < 0 {
		if selectedErr != nil {
			return nil, selectedErr
		}
		return nil, servlet.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound), nil)
	}
	return h.registrations[bestIndex].handler.Handle(ctx)
}

func selectConditionError(current error, candidate error) error {
	if current == nil {
		return candidate
	}
	if conditionErrorRank(candidate) > conditionErrorRank(current) {
		return candidate
	}
	return current
}

func conditionErrorRank(err error) int {
	var statusErr servlet.StatusError
	if !errors.As(err, &statusErr) {
		return 1
	}
	switch statusErr.StatusCode() {
	case http.StatusUnsupportedMediaType:
		return 4
	case http.StatusNotAcceptable:
		return 3
	case http.StatusBadRequest:
		return 2
	default:
		return 1
	}
}
