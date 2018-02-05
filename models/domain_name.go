package models

import (
	"fmt"
	"strings"
	"github.com/crewjam/go-cloudformation"
)

type DomainName struct {
	BaseDomainName *string
	Stage          *Stage
}

func (d *DomainName) StringExpr() *cloudformation.StringExpr {
	return cloudformation.String(d.string())
}

func (d *DomainName) string() string {
	return fmt.Sprintf(
		"%s-%s.com",
		d.BaseDomainName,
		strings.ToLower(d.Stage.name),
	)
}
