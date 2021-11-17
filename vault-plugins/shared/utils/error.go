package utils

import "fmt"

type ErrorCollector []error

func (c *ErrorCollector) Collect(e error) { *c = append(*c, e) }

func (c *ErrorCollector) Error() string {
	if c.Empty() {
		return ""
	}
	if len(*c) == 1 {
		return (*c)[0].Error()
	}
	err := "collected errors:\n"
	for i, e := range *c {
		err += fmt.Sprintf("\tError %d: %s\n", i, e.Error())
	}

	return err
}

func (c *ErrorCollector) Empty() bool {
	return len(*c) == 0
}

func (c *ErrorCollector) Err() error {
	if c.Empty() {
		return nil
	}
	return fmt.Errorf(c.Error())
}
