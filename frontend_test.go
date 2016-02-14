package main

import "testing"

func TestPlausibleUserId(t *testing.T) {
	type plausibleTest struct {
		input    string
		expected string
	}
	tests := []plausibleTest{
		{
			"https://plus.google.com/+Some_User1_/posts/hXs4HS4wK6U",
			"+Some_User1_",
		},
		{
			"https://plus.google.com/+Some_User1_/",
			"+Some_User1_",
		},
		{
			"https://plus.google.com/+Some_User1_",
			"+Some_User1_",
		},
		{
			"http://plus.google.com/+PlaintextSome_User1_",
			"+PlaintextSome_User1_",
		},
		{
			"+Some_User1_",
			"+Some_User1_",
		},
		{
			"1111",
			"1111",
		},
	}

	for i, tc := range tests {
		id := PlausibleUserId(tc.input)
		if tc.expected != id {
			t.Errorf("%d, %q: want %q, got %q", i, tc.input, tc.expected, id)
		}
	}
}
