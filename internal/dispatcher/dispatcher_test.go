package dispatcher

import "testing"

func TestTopicForChannel(t *testing.T) {
	cases := map[string]string{
		"email":    "dispatch.email",
		"sms":      "dispatch.sms",
		"push":     "dispatch.push",
		"whatsapp": "dispatch.wa",
		"unknown":  "",
	}

	for input, expected := range cases {
		if got := topicForChannel(input); got != expected {
			t.Fatalf("topicForChannel(%s)=%s, expected %s", input, got, expected)
		}
	}
}
