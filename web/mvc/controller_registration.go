package mvc

import (
	"fmt"
	"sort"
	"strings"

	arkweb "goark.dev/arkarta/web"
	goweb "goark.dev/goark/web"
)

type routeRegistrationKey struct {
	method  string
	pattern string
}

type routeRegistration struct {
	handler    arkweb.Handler
	conditions Conditions
}

func registerControllers(registry *goweb.Registry, controllers []Controller) error {
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	groups := make(map[routeRegistrationKey][]routeRegistration)
	keys := make([]routeRegistrationKey, 0)
	for _, controller := range controllers {
		if err := appendControllerRegistrations(registry, controller, groups, &keys); err != nil {
			return err
		}
	}
	for _, key := range keys {
		handler, err := buildRouteRegistrationHandler(key, groups[key])
		if err != nil {
			return err
		}
		if err := registry.Handle(key.method, key.pattern, handler); err != nil {
			return err
		}
	}
	return nil
}

func appendControllerRegistrations(
	registry *goweb.Registry,
	controller Controller,
	groups map[routeRegistrationKey][]routeRegistration,
	keys *[]routeRegistrationKey,
) error {
	var implicitGroups map[uint64]struct{}
	for _, route := range controller.routes {
		if controller.skipImplicitRoute(route, implicitGroups) {
			continue
		}
		if route.implicitMethods && len(controller.methods) > 0 && route.methodGroupID != 0 {
			if implicitGroups == nil {
				implicitGroups = make(map[uint64]struct{})
			}
			implicitGroups[route.methodGroupID] = struct{}{}
		}
		for _, method := range controller.routeMethods(route) {
			resolvedRoute := route
			resolvedRoute.Method = method
			if err := registerRouteCORS(registry, controller, resolvedRoute); err != nil {
				return err
			}
			key := routeRegistrationKey{method: resolvedRoute.Method, pattern: resolvedRoute.Pattern}
			if _, exists := groups[key]; !exists {
				*keys = append(*keys, key)
			}
			groups[key] = append(groups[key], routeRegistration{
				handler:    bindControllerKind(controller.kind, resolvedRoute.Handler),
				conditions: mergeControllerRouteConditions(controller.conditions, resolvedRoute.Conditions),
			})
		}
	}
	return nil
}

func registerRouteCORS(registry *goweb.Registry, controller Controller, route Route) error {
	if config, ok := controller.crossOriginFor(route); ok {
		return registry.AddCORSMapping(route.Pattern, crossOriginMethods(route.Method), *config)
	}
	return nil
}

func buildRouteRegistrationHandler(key routeRegistrationKey, registrations []routeRegistration) (arkweb.Handler, error) {
	if len(registrations) == 0 {
		return nil, fmt.Errorf("goark/web/mvc: empty route registration %s %s", key.method, key.pattern)
	}
	if len(registrations) == 1 {
		registration := registrations[0]
		return registration.conditions.wrap(registration.handler), nil
	}
	if err := rejectAmbiguousRouteConditions(key, registrations); err != nil {
		return nil, err
	}
	return newConditionDispatchHandler(registrations), nil
}

func rejectAmbiguousRouteConditions(key routeRegistrationKey, registrations []routeRegistration) error {
	seen := make(map[string]struct{}, len(registrations))
	for _, registration := range registrations {
		signature := conditionSignature(registration.conditions)
		if _, exists := seen[signature]; exists {
			return fmt.Errorf("goark/web/mvc: ambiguous route conditions for %s %s", key.method, key.pattern)
		}
		seen[signature] = struct{}{}
	}
	return nil
}

func conditionSignature(conditions Conditions) string {
	return strings.Join([]string{
		conditionSignaturePart(conditions.Consumes),
		conditionSignaturePart(conditions.Produces),
		conditionSignaturePart(conditions.Params),
		conditionSignaturePart(conditions.Headers),
	}, "\x01")
}

func conditionSignaturePart(values []string) string {
	if len(values) == 0 {
		return ""
	}
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return strings.Join(copied, "\x00")
}

func (c Controller) skipImplicitRoute(route Route, groups map[uint64]struct{}) bool {
	if !route.implicitMethods || len(c.methods) == 0 || route.methodGroupID == 0 {
		return false
	}
	_, exists := groups[route.methodGroupID]
	return exists
}

func (c Controller) routeMethods(route Route) []string {
	if len(c.methods) == 0 {
		return []string{route.Method}
	}
	if route.implicitMethods {
		return c.methods
	}
	methods := make([]string, 0, len(c.methods)+1)
	if method := normalizeSingleRouteMethod(route.Method); method != "" {
		methods = append(methods, method)
	}
	for _, method := range c.methods {
		if !hasRequestMethod(methods, method) {
			methods = append(methods, method)
		}
	}
	return methods
}
