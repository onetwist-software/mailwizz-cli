package generator_test

import (
	"testing"

	"github.com/onetwist-software/mailwizz-cli/generator"
)

func TestFlagName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "list_uid", want: "list-uid"},
		{in: "EMAIL", want: "email"},
		{in: "FNAME", want: "fname"},
		{in: "general[name]", want: "general-name"},
		{in: "general[description]", want: "general-description"},
		{in: "defaults[from_email]", want: "defaults-from-email"},
		{in: "notifications[subscribe_to]", want: "notifications-subscribe-to"},
		{in: "details[ip_address]", want: "details-ip-address"},
		{in: "email_message_id", want: "email-message-id"},
		{in: "campaign_uid", want: "campaign-uid"},
		{in: "page", want: "page"},
		{in: "per_page", want: "per-page"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			if got := generator.FlagName(tt.in); got != tt.want {
				t.Errorf("FlagName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGoFieldName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "list_uid", want: "ListUID"},
		{in: "EMAIL", want: "Email"},
		{in: "general[name]", want: "GeneralName"},
		{in: "defaults[from_email]", want: "DefaultsFromEmail"},
		{in: "details[ip_address]", want: "DetailsIPAddress"},
		{in: "email_message_id", want: "EmailMessageID"},
		{in: "server_id", want: "ServerID"},
		{in: "country_id", want: "CountryID"},
		{in: "per_page", want: "PerPage"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			if got := generator.GoFieldName(tt.in); got != tt.want {
				t.Errorf("GoFieldName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGoOperationName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "viewListSubscriber", want: "ViewListSubscriber"},
		{in: "createList", want: "CreateList"},
		{in: "pauseunpauseCampaign", want: "PauseunpauseCampaign"},
		{in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			if got := generator.GoOperationName(tt.in); got != tt.want {
				t.Errorf("GoOperationName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDefaultCommandName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{method: "GET", path: "/lists", want: "list"},
		{method: "GET", path: "/lists/{list_uid}", want: "view"},
		{method: "GET", path: "/lists/{list_uid}/subscribers", want: "list"},
		{method: "GET", path: "/lists/{list_uid}/subscribers/{subscriber_uid}", want: "view"},
		{method: "POST", path: "/lists", want: "create"},
		{method: "PUT", path: "/lists/{list_uid}", want: "update"},
		{method: "DELETE", path: "/lists/{list_uid}", want: "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			t.Parallel()

			if got := generator.DefaultCommandName(tt.method, tt.path); got != tt.want {
				t.Errorf("DefaultCommandName(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultCommandPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want []string
	}{
		{path: "/lists", want: []string{"lists"}},
		{path: "/lists/{list_uid}", want: []string{"lists"}},
		{path: "/lists/{list_uid}/subscribers", want: []string{"lists", "subscribers"}},
		{path: "/lists/{list_uid}/subscribers/{subscriber_uid}", want: []string{"lists", "subscribers"}},
		{path: "lists/fields/types", want: []string{"lists", "fields", "types"}},
		{path: "/campaigns/{campaign_uid}/track-url/{subscriber_uid}/{hash}", want: []string{"campaigns", "track-url"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got := generator.DefaultCommandPath(tt.path)
			if len(got) != len(tt.want) {
				t.Fatalf("DefaultCommandPath(%q) = %v, want %v", tt.path, got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("DefaultCommandPath(%q) = %v, want %v", tt.path, got, tt.want)
				}
			}
		})
	}
}
