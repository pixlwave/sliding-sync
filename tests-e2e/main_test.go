package syncv3_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matrix-org/sync-v3/sync3"
	"github.com/matrix-org/sync-v3/testutils/m"
)

var (
	proxyBaseURL      = "http://localhost"
	homeserverBaseURL = os.Getenv("SYNCV3_SERVER")
	userCounter       uint64
)

func TestMain(m *testing.M) {
	listenAddr := os.Getenv("SYNCV3_BINDADDR")
	if listenAddr == "" {
		fmt.Println("SYNCV3_BINDADDR must be set")
		os.Exit(1)
	}
	segments := strings.Split(listenAddr, ":")
	proxyBaseURL += ":" + segments[1]
	fmt.Println("proxy located at", proxyBaseURL)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func assertEventsEqual(t *testing.T, wantList []Event, gotList []json.RawMessage) {
	t.Helper()
	err := eventsEqual(wantList, gotList)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func eventsEqual(wantList []Event, gotList []json.RawMessage) error {
	if len(wantList) != len(gotList) {
		return fmt.Errorf("got %d events, want %d", len(gotList), len(wantList))
	}
	for i := 0; i < len(wantList); i++ {
		want := wantList[i]
		var got Event
		if err := json.Unmarshal(gotList[i], &got); err != nil {
			return fmt.Errorf("failed to unmarshal event %d: %s", i, err)
		}
		if want.ID != "" && got.ID != want.ID {
			return fmt.Errorf("event %d ID mismatch: got %v want %v", i, got.ID, want.ID)
		}
		if want.Content != nil && !reflect.DeepEqual(got.Content, want.Content) {
			return fmt.Errorf("event %d content mismatch: got %+v want %+v", i, got.Content, want.Content)
		}
		if want.Type != "" && want.Type != got.Type {
			return fmt.Errorf("event %d Type mismatch: got %v want %v", i, got.Type, want.Type)
		}
		if want.StateKey != nil {
			if got.StateKey == nil {
				return fmt.Errorf("event %d StateKey mismatch: want %v got <nil>", i, *want.StateKey)
			} else if *want.StateKey != *got.StateKey {
				return fmt.Errorf("event %d StateKey mismatch: got %v want %v", i, *got.StateKey, *want.StateKey)
			}
		}
		if want.Sender != "" && want.Sender != got.Sender {
			return fmt.Errorf("event %d Sender mismatch: got %v want %v", i, got.Sender, want.Sender)
		}
	}
	return nil
}

func MatchRoomTimelineMostRecent(n int, events []Event) m.RoomMatcher {
	subset := events[len(events)-n:]
	return func(r sync3.Room) error {
		if len(r.Timeline) < len(subset) {
			return fmt.Errorf("timeline length mismatch: got %d want at least %d", len(r.Timeline), len(subset))
		}
		gotSubset := r.Timeline[len(r.Timeline)-n:]
		return eventsEqual(events, gotSubset)
	}
}

func MatchRoomRequiredState(events []Event) m.RoomMatcher {
	return func(r sync3.Room) error {
		if len(r.RequiredState) != len(events) {
			return fmt.Errorf("required state length mismatch, got %d want %d", len(r.RequiredState), len(events))
		}
		// allow any ordering for required state
		for _, want := range events {
			found := false
			for _, got := range r.RequiredState {
				if err := eventsEqual([]Event{want}, []json.RawMessage{got}); err == nil {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("required state want event %+v but it does not exist", want)
			}
		}
		return nil
	}
}

func registerNewUser(t *testing.T) *CSAPI {
	// create user
	httpClient := NewLoggedClient(t, "localhost", nil)
	client := &CSAPI{
		Client:           httpClient,
		BaseURL:          homeserverBaseURL,
		SyncUntilTimeout: 3 * time.Second,
	}
	localpart := fmt.Sprintf("user-%d-%d", time.Now().Unix(), atomic.AddUint64(&userCounter, 1))

	client.UserID, client.AccessToken, client.DeviceID = client.RegisterUser(t, localpart, "password")
	client.Localpart = strings.Split(client.UserID, ":")[0][1:]
	return client
}

func ptr(s string) *string {
	return &s
}