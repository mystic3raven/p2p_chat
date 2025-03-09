package chat

import "testing"

func TestChat(t *testing.T) {
    expected := "Chat started!"
    result := StartChat()
    if result != expected {
        t.Errorf("Expected %s but got %s", expected, result)
    }
}

