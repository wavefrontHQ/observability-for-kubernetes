package result

import "k8s.io/apimachinery/pkg/util/errors"

var Valid = Result{}

type Result struct {
	isWarning bool
	error     error
}

func New(isWarning bool, errs ...error) Result {
	if len(errs) == 0 {
		return Valid
	}
	if len(errs) == 1 {
		return Result{isWarning, errs[0]}
	}
	return Result{isWarning, errors.NewAggregate(errs)}
}

func NewError(errs ...error) Result {
	return New(false, errs...)
}

func (r Result) Message() string {
	if r.IsValid() {
		return ""
	} else {
		return r.error.Error()
	}
}

func (r Result) IsValid() bool {
	return r.error == nil
}

func (r Result) IsError() bool {
	return !r.IsValid() && !r.isWarning
}

func (r Result) IsWarning() bool {
	return !r.IsValid() && r.isWarning
}
