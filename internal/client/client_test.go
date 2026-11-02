package client_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/fakeapi"
)

const (
	testToken   = "TEST-TOKEN"
	testProject = "project-1"
)

func newTestClient(t *testing.T) (*client.Client, *fakeapi.Server) {
	t.Helper()
	srv := fakeapi.New(testToken, testProject)
	t.Cleanup(srv.Close)

	c, err := client.New(client.Config{BaseURL: srv.BaseURL(), APIToken: testToken})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, srv
}

func TestNewValidation(t *testing.T) {
	if _, err := client.New(client.Config{}); err == nil {
		t.Error("expected error for empty token")
	}
	if _, err := client.New(client.Config{APIToken: "x", BaseURL: "ftp://example.com"}); err == nil {
		t.Error("expected error for unsupported scheme")
	}
}

func TestLoginInvalidToken(t *testing.T) {
	srv := fakeapi.New(testToken, testProject)
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.BaseURL(), APIToken: "wrong"})
	if err != nil {
		t.Fatal(err)
	}
	err = c.Login(context.Background())
	if err == nil {
		t.Fatal("expected login error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
}

func TestSessionReusedAcrossRequests(t *testing.T) {
	c, srv := newTestClient(t)
	ctx := context.Background()
	srv.AddMachine("web-1")

	for range 3 {
		if _, err := c.ListMachines(ctx, testProject); err != nil {
			t.Fatalf("ListMachines: %v", err)
		}
	}
	if srv.LoginCount() != 1 {
		t.Errorf("expected 1 login, got %d", srv.LoginCount())
	}
}

func TestReauthenticatesOn401(t *testing.T) {
	logins, calls := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			logins++
			w.Header().Add("Set-Cookie", `XSRF-TOKEN="tok"; Path=/`)
			w.WriteHeader(http.StatusOK)
		default:
			calls++
			if calls == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"id":"m1","sourceProperties":{"name":"a"}}`))
		}
	}))
	defer srv.Close()

	c, _ := client.New(client.Config{BaseURL: srv.URL, APIToken: "x"})
	m, err := c.GetMachine(context.Background(), "p", "m1")
	if err != nil {
		t.Fatalf("GetMachine: %v", err)
	}
	if m.ID != "m1" || logins != 2 {
		t.Fatalf("got machine %q after %d logins", m.ID, logins)
	}
}

func TestBlueprintLifecycle(t *testing.T) {
	c, srv := newTestClient(t)
	ctx := context.Background()
	machineID := srv.AddMachine("web-1")

	created, err := c.CreateBlueprint(ctx, testProject, &client.Blueprint{
		MachineID:    machineID,
		InstanceType: "COPY_ORIGIN",
		SubnetIDs:    &[]string{"subnet-1"},
	})
	if err != nil {
		t.Fatalf("CreateBlueprint: %v", err)
	}
	if created.ID == "" || created.Region == "" {
		t.Fatalf("expected computed fields, got %+v", created)
	}

	found, err := c.FindBlueprintByMachineID(ctx, testProject, machineID)
	if err != nil || found == nil || found.ID != created.ID {
		t.Fatalf("FindBlueprintByMachineID = %+v, %v", found, err)
	}

	updated, err := c.UpdateBlueprint(ctx, testProject, created.ID, &client.Blueprint{InstanceType: "m5.large"})
	if err != nil {
		t.Fatalf("UpdateBlueprint: %v", err)
	}
	if updated.InstanceType != "m5.large" || updated.SubnetIDs == nil || len(*updated.SubnetIDs) != 1 {
		t.Fatalf("unexpected update result %+v", updated)
	}

	if _, err := c.GetBlueprint(ctx, testProject, "missing"); !client.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := c.CreateBlueprint(ctx, testProject, &client.Blueprint{}); err == nil {
		t.Fatal("expected error for missing machine id")
	}
}

func TestRecoveryPlanLifecycle(t *testing.T) {
	c, srv := newTestClient(t)
	ctx := context.Background()
	m := srv.AddMachine("db-1")

	rp, err := c.CreateRecoveryPlan(ctx, testProject, &client.RecoveryPlan{
		Name:  "plan",
		Steps: client.RecoveryPlanSteps{Items: []client.RecoveryPlanStep{{Name: "db", MachineIDs: []string{m}}}},
	})
	if err != nil {
		t.Fatalf("CreateRecoveryPlan: %v", err)
	}

	rp.Name = "renamed"
	if _, err := c.UpdateRecoveryPlan(ctx, testProject, rp.ID, rp); err != nil {
		t.Fatalf("UpdateRecoveryPlan: %v", err)
	}
	got, err := c.GetRecoveryPlan(ctx, testProject, rp.ID)
	if err != nil || got.Name != "renamed" || len(got.Steps.Items) != 1 {
		t.Fatalf("GetRecoveryPlan = %+v, %v", got, err)
	}

	if err := c.DeleteRecoveryPlan(ctx, testProject, rp.ID); err != nil {
		t.Fatalf("DeleteRecoveryPlan: %v", err)
	}
	if _, err := c.GetRecoveryPlan(ctx, testProject, rp.ID); !client.IsNotFound(err) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}
