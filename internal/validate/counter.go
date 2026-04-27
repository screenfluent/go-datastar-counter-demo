package validate

import (
	"errors"

	z "github.com/Oudwins/zog"
)

var changeSchema = z.Struct(z.Shape{
	"Delta": z.Int().OneOf([]int{-1, 1}, z.Message("dozwolone akcje to tylko -1 albo +1")),
})

type changeInput struct {
	Delta int
}

func Change(delta int) error {
	change := changeInput{Delta: delta}
	if issues := changeSchema.Validate(&change); issues != nil {
		return errors.New("nieprawidlowa zmiana licznika")
	}
	return nil
}
