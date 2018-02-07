package cf_funcs

import (
	. "github.com/crewjam/go-cloudformation"
)

// Sub represents the Fn::Sub function called over value.
func Sub(value Stringable) *StringExpr {
	return SubFunc{Value: *value.String()}.String()
}

type SubFunc struct {
	Value StringExpr `json:"Fn::Sub"`
}

func (f SubFunc) String() *StringExpr {
	return &StringExpr{Func: f}
}

var _ Stringable = SubFunc{} // SubFunc must implement Stringable
var _ StringFunc = SubFunc{} // SubFunc must implement StringFunc
