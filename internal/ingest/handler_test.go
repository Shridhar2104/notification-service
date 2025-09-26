package ingest

import "testing"

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		request NotifyRequest
		wantErr bool
	}{
		{
			name:    "valid",
			request: NotifyRequest{Channel: ChannelEmail, TemplateID: "tpl", To: map[string]any{"email": "a@b.com"}},
		},
		{
			name:    "missing channel",
			request: NotifyRequest{TemplateID: "tpl", To: map[string]any{"email": "a@b.com"}},
			wantErr: true,
		},
		{
			name:    "missing template",
			request: NotifyRequest{Channel: ChannelEmail, To: map[string]any{"email": "a@b.com"}},
			wantErr: true,
		},
		{
			name:    "missing recipient",
			request: NotifyRequest{Channel: ChannelEmail, TemplateID: "tpl"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRequest(tc.request)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
