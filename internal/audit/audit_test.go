package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSubscriber struct {
	called bool
	event  *Event
}

func (m *mockSubscriber) Notify(ctx context.Context, e *Event) error {
	m.called = true
	m.event = e
	return nil
}

func Test_Publisher_Subscribe(t *testing.T) {
	p := NewPublisher()
	mockSub := &mockSubscriber{}
	p.Subscribe(mockSub)

	require.Equal(t, 1, len(p.subs))
}

func Test_Publisher_Publish(t *testing.T) {
	p := NewPublisher()

	sub1 := &mockSubscriber{}
	sub2 := &mockSubscriber{}

	p.Subscribe(sub1)
	p.Subscribe(sub2)

	event := &Event{
		TS:     1234567890,
		Action: "test_action",
		UserID: "user123",
		URL:    "http://example.com",
	}

	err := p.Publish(context.Background(), event)
	require.NoError(t, err)

	require.True(t, sub1.called)
	require.True(t, sub2.called)
	require.Equal(t, event, sub1.event)
	require.Equal(t, event, sub2.event)
}

func Test_FileSubscriber_Notify(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := dir + "/audit.log"

	sub, err := NewFileSubscriber(file)
	require.NoError(t, err)
	defer sub.Close()

	event := &Event{
		TS:     1234567890,
		Action: "test_action",
		UserID: "user123",
		URL:    "http://example.com",
	}

	err = sub.Notify(context.Background(), event)
	require.NoError(t, err)

	data, err := os.ReadFile(file)
	require.NoError(t, err)

	expected := `{"ts":1234567890,"action":"test_action","user_id":"user123","url":"http://example.com"}`
	assert.Equal(t, expected+"\n", string(data))
}

func Test_HTTPSubscriber_Notify(t *testing.T) {
	t.Parallel()

	var received Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&received)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sub := NewHTTPSubscriber(server.URL)

	event := &Event{
		TS:     1234567890,
		Action: "test_follow",
		UserID: "user123",
		URL:    "http://example.com",
	}

	err := sub.Notify(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, *event, received)
}

func Test_HTTPSubscriber_Retry(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)

		if n < 4 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sub := NewHTTPSubscriber(server.URL)

	event := &Event{
		TS:     1234567890,
		Action: "test_retry",
		UserID: "user123",
		URL:    "http://example.com",
	}

	err := sub.Notify(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, int32(4), attempts.Load())
}
