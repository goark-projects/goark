package mvc

import (
	"reflect"

	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/goark/web/message"
)

func bindAndValidateJSON(ctx *arkweb.Context, target any, groups []string) error {
	if len(groups) == 0 {
		if err := ctx.BindJSON(target); err != nil {
			return err
		}
		if !supportsValidation(target) {
			return nil
		}
		return validateBound(ctx, target, nil)
	}
	if err := ctx.BindJSON(target); err != nil {
		return err
	}
	if !supportsValidation(target) {
		return nil
	}
	return validateBound(ctx, target, groups)
}

func bindAndValidateBody(ctx *arkweb.Context, target any, groups []string, mediaTypes []string) error {
	if err := message.NewReader().Read(ctx, target, mediaTypes...); err != nil {
		return err
	}
	if !supportsValidation(target) {
		return nil
	}
	return validateBound(ctx, target, groups)
}

func validateBound(ctx *arkweb.Context, target any, groups []string) error {
	var (
		result validation.Result
		err    error
	)
	if len(groups) == 0 {
		result, err = ctx.Validate(target)
	} else {
		result, err = ctx.ValidateGroups(target, groups...)
	}
	if err != nil {
		return err
	}
	return result.Error()
}

func cloneValidationGroups(groups []string) []string {
	if len(groups) == 0 {
		return nil
	}
	cloned := make([]string, len(groups))
	copy(cloned, groups)
	return cloned
}

func supportsValidation(target any) bool {
	typ := reflect.TypeOf(target)
	if typ == nil {
		return false
	}
	typ = indirectValidationType(typ)
	switch typ.Kind() {
	case reflect.Struct:
		return true
	case reflect.Array, reflect.Slice:
		return indirectValidationType(typ.Elem()).Kind() == reflect.Struct
	default:
		return false
	}
}

func indirectValidationType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}
