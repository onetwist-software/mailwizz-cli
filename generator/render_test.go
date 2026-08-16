package generator_test

import (
	"testing"

	"github.com/onetwist-software/mailwizz-cli/generator"
)

func TestOperationPathExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		op   generator.Operation
		want string
	}{
		{
			name: "no path params",
			op:   generator.Operation{Path: "/lists"},
			want: `"/lists"`,
		},
		{
			name: "single path param",
			op: generator.Operation{
				Path:       "/lists/{list_uid}",
				PathParams: []*generator.Param{{Name: "list_uid", GoField: "ListUID"}},
			},
			want: `"/lists/" + url.PathEscape(req.ListUID)`,
		},
		{
			name: "two path params with a literal segment between them",
			op: generator.Operation{
				Path: "/lists/{list_uid}/subscribers/{subscriber_uid}",
				PathParams: []*generator.Param{
					{Name: "list_uid", GoField: "ListUID"},
					{Name: "subscriber_uid", GoField: "SubscriberUID"},
				},
			},
			want: `"/lists/" + url.PathEscape(req.ListUID) + "/subscribers/" + url.PathEscape(req.SubscriberUID)`,
		},
		{
			name: "three path params",
			op: generator.Operation{
				Path: "/campaigns/{campaign_uid}/track-url/{subscriber_uid}/{hash}",
				PathParams: []*generator.Param{
					{Name: "campaign_uid", GoField: "CampaignUID"},
					{Name: "subscriber_uid", GoField: "SubscriberUID"},
					{Name: "hash", GoField: "Hash"},
				},
			},
			want: `"/campaigns/" + url.PathEscape(req.CampaignUID) + "/track-url/" + url.PathEscape(req.SubscriberUID) + "/" + url.PathEscape(req.Hash)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.op.PathExpr(); got != tt.want {
				t.Errorf("PathExpr() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestOperationFuncName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		op   generator.Operation
		want string
	}{
		{
			op:   generator.Operation{CommandPath: []string{"lists"}, CommandName: "create"},
			want: "listsCreateCommand",
		},
		{
			op:   generator.Operation{CommandPath: []string{"lists", "subscribers"}, CommandName: "search-by-email"},
			want: "listsSubscribersSearchByEmailCommand",
		},
		{
			op:   generator.Operation{CommandPath: []string{"campaigns"}, CommandName: "pause-unpause"},
			want: "campaignsPauseUnpauseCommand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.op.FuncName(); got != tt.want {
				t.Errorf("FuncName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestResourceFuncName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resource generator.Resource
		want     string
	}{
		{resource: generator.Resource{Path: []string{"lists"}}, want: "listsCommand"},
		{resource: generator.Resource{Path: []string{"lists", "subscribers"}}, want: "listsSubscribersCommand"},
		{resource: generator.Resource{Path: []string{"delivery-servers"}}, want: "deliveryServersCommand"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.resource.FuncName(); got != tt.want {
				t.Errorf("FuncName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestOperationRequestFieldsOrderAndContent(t *testing.T) {
	t.Parallel()

	op := generator.Operation{
		PathParams:  []*generator.Param{{GoField: "ListUID", FlagName: "list-uid", Required: true}},
		QueryParams: []*generator.Param{{GoField: "Status", FlagName: "status", Enum: []string{"a", "b"}}},
		BodyFields:  []*generator.Field{{GoField: "GeneralName", FlagName: "general-name", Required: true}},
	}

	fields := op.RequestFields()
	if len(fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(fields))
	}

	wantOrder := []string{"list-uid", "status", "general-name"}
	for i, want := range wantOrder {
		if fields[i].FlagName != want {
			t.Errorf("fields[%d].FlagName = %q, want %q", i, fields[i].FlagName, want)
		}
	}

	if !fields[0].Required {
		t.Errorf("path param field should preserve Required = true")
	}

	if len(fields[1].Enum) != 2 {
		t.Errorf("query param field should preserve Enum, got %v", fields[1].Enum)
	}
}
