package main

import (
	"testing"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    " Hello World ",
			expected: []string{"hello", "world"},
		},

		{
			input:    " Hello doug",
			expected: []string{"hello", "doug"},
		},
		{
			input:    "Hey Kamau ",
			expected: []string{"hey", "kamau"},
		},
	}

	for _, c := range cases {
		actual := CleanInput(c.input)
		if len(actual) != len(c.expected) {
			t.Errorf("the lengths don't match")
			return
		}

		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]

			if word != expectedWord {
				t.Errorf("words don't match")
				return
			}
		}
	}

}
