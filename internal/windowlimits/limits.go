package windowlimits

import "errors"

const Maximum = 16384

func Validate(width, height, minWidth, minHeight uint) error {
	if minWidth > Maximum || minHeight > Maximum {
		return errors.New("window minimum dimensions must not exceed 16384 logical units")
	}
	if minWidth > width || minHeight > height {
		return errors.New("window minimum dimensions must not exceed initial dimensions")
	}
	return nil
}
