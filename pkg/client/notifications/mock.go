package notifications

import "context"

type MockTracker struct {
	Payload NotificationPayload
}

type ClientMock struct {
	Tracker *MockTracker
}

func NewClientMock(ctx context.Context) (*ClientMock, *MockTracker) {
	tracker := &MockTracker{}
	return &ClientMock{
		Tracker: tracker,
	}, tracker
}

func (n *ClientMock) Send(ctx context.Context, payload *NotificationPayload) error {
	n.Tracker.Payload = *payload
	return nil
}
