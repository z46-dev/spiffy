package spiffy

import (
	"fmt"
	"reflect"
)

func validateSliderValueFormat[T Numeric](format string) (err error) {
	var verb byte
	if verb, err = sliderValueFormatVerb(format); err != nil {
		return
	}

	var zero T
	if sliderFormatVerbMatchesKind(verb, reflect.TypeOf(zero).Kind()) {
		return
	}

	err = fmt.Errorf("slider value format verb %%%c is incompatible with %T", verb, zero)
	return
}

func sliderValueFormatVerb(format string) (verb byte, err error) {
	var targets int
	for index := 0; index < len(format); index++ {
		if format[index] != '%' {
			continue
		}

		index++
		if index >= len(format) {
			err = fmt.Errorf("slider value format ends with an unescaped percent sign")
			return
		}

		if format[index] == '%' {
			continue
		}

		if format[index] == '[' {
			index++
			var start int = index
			for index < len(format) && format[index] >= '0' && format[index] <= '9' {
				index++
			}

			if start == index || index >= len(format) || format[index] != ']' {
				err = fmt.Errorf("slider value format contains an invalid argument index")
				return
			}

			if index-start != 1 || format[start] != '1' {
				err = fmt.Errorf("slider value format can only reference argument one")
				return
			}

			index++
		}

		for index < len(format) && isFormatFlag(format[index]) {
			index++
		}

		if index < len(format) && format[index] == '*' {
			err = fmt.Errorf("slider value format cannot use dynamic width")
			return
		}

		for index < len(format) && format[index] >= '0' && format[index] <= '9' {
			index++
		}

		if index < len(format) && format[index] == '.' {
			index++
			if index < len(format) && format[index] == '*' {
				err = fmt.Errorf("slider value format cannot use dynamic precision")
				return
			}

			for index < len(format) && format[index] >= '0' && format[index] <= '9' {
				index++
			}
		}

		if index >= len(format) || !isNumericFormatVerb(format[index]) {
			err = fmt.Errorf("slider value format contains an unsupported numeric target")
			return
		}

		verb = format[index]
		targets++
	}

	if targets != 1 {
		err = fmt.Errorf("slider value format must contain exactly one numeric target")
	}

	return
}

func defaultSliderValueFormat[T Numeric]() (format string) {
	var zero T
	if reflect.TypeOf(zero).Kind() == reflect.Float32 || reflect.TypeOf(zero).Kind() == reflect.Float64 {
		format = "%.0f"
	} else {
		format = "%d"
	}

	return
}

func isFormatFlag(character byte) bool {
	return character == '+' || character == '-' || character == '#' || character == '0' || character == ' '
}

func isNumericFormatVerb(character byte) bool {
	return isIntegerFormatVerb(character) || isFloatFormatVerb(character)
}

func isIntegerFormatVerb(character byte) bool {
	return character == 'b' || character == 'c' || character == 'd' || character == 'o' || character == 'O' || character == 'x' || character == 'X' || character == 'U'
}

func isFloatFormatVerb(character byte) bool {
	return character == 'b' || character == 'e' || character == 'E' || character == 'f' || character == 'F' || character == 'g' || character == 'G' || character == 'x' || character == 'X'
}

func sliderFormatVerbMatchesKind(verb byte, kind reflect.Kind) bool {
	if kind == reflect.Float32 || kind == reflect.Float64 {
		return isFloatFormatVerb(verb)
	}

	return isIntegerFormatVerb(verb)
}
