package main

import "testing"

func TestClockStyleOnesOnlyInMultiDigitNumbers(t *testing.T) {
	for _, tc := range []struct{ number, advance, width int }{
		{1, 6, 11}, {10, 6, 13}, {11, 6, 9}, {21, 6, 13}, {22, 6, 17},
		{1, 5, 9}, {10, 5, 11}, {11, 5, 8},
	} {
		if got := scratchLabelWidth(tc.number, tc.advance); got != tc.width {
			t.Errorf("number %d advance %d: got %d want %d", tc.number, tc.advance, got, tc.width)
		}
	}
}
