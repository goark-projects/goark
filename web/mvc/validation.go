package mvc

import (
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
)

func bindAndValidateJSON(ctx *arkweb.Context, target any, groups []string) error {
	if len(groups) == 0 {
		return ctx.BindAndValidateJSON(target)
	}
	if err := ctx.BindJSON(target); err != nil {
		return err
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
