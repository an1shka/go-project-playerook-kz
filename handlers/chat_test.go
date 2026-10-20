package handlers

import "testing"

func TestConversationIDIsSymmetric(t *testing.T) {
	first := conversationID("user-b", "user-a")
	second := conversationID("user-a", "user-b")
	if first != second {
		t.Fatalf("conversation IDs differ: %q != %q", first, second)
	}
	if first != "user-a:user-b" {
		t.Fatalf("unexpected conversation ID: %q", first)
	}
}

func TestValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{id: "3f4a2c10-5d6e-4a8b-9c0d-1e2f3a4b5c6d", want: true},
		{id: "not-a-uuid", want: false},
		{id: "3f4a2c10-5d6e-4a8b-9c0d-1e2f3a4b5c6z", want: false},
	}
	for _, test := range tests {
		if got := validID(test.id); got != test.want {
			t.Errorf("validID(%q) = %v, want %v", test.id, got, test.want)
		}
	}
}

func TestValidUsername(t *testing.T) {
	tests := []struct {
		username string
		want     bool
	}{
		{username: "gamer_kz", want: true},
		{username: "abc", want: true},
		{username: "ab", want: false},
		{username: "Player One", want: false},
		{username: "nickname!", want: false},
		{username: "эта_игра", want: false},
	}
	for _, test := range tests {
		if got := validUsername(test.username); got != test.want {
			t.Errorf("validUsername(%q) = %v, want %v", test.username, got, test.want)
		}
	}
}
