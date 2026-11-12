//go:build postgres

package hosts

import "testing"

func TestAccountSnapshotReplacement(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	host := enrollTestHost(t, ctx, store, "account-inventory", nil)
	accounts := []HostAccount{
		{UID: "501", Username: "alex", UUID: "account-uuid", IsAdmin: true, LoggedIn: true, FileVaultUnlock: new(true)},
		{UID: "0", Username: "root"},
	}
	if err := store.ReplaceAccounts(ctx, host.ID, accounts); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListAccounts(ctx, host.ID)
	if err != nil || len(got) != 2 || !got[0].LoggedIn || got[1].LoggedIn {
		t.Fatalf("initial accounts = %+v, %v", got, err)
	}
	accounts[0].LoggedIn = false
	accounts[0].FileVaultUnlock = nil
	if err := store.ReplaceAccounts(ctx, host.ID, accounts[:1]); err != nil {
		t.Fatal(err)
	}
	got, err = store.ListAccounts(ctx, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].FileVaultUnlock != nil || got[0].LoggedIn {
		t.Fatalf("replaced accounts = %+v", got)
	}
	if err := store.ReplaceAccounts(ctx, host.ID, []HostAccount{accounts[0], accounts[0]}); err == nil {
		t.Fatal("duplicate account snapshot succeeded")
	}
	got, err = store.ListAccounts(ctx, host.ID)
	if err != nil || len(got) != 1 {
		t.Fatalf("failed snapshot lost accounts: %+v, %v", got, err)
	}
	if err := store.ReplaceAccounts(ctx, host.ID, nil); err != nil {
		t.Fatal(err)
	}
	got, err = store.ListAccounts(ctx, host.ID)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty snapshot = %+v, %v", got, err)
	}
	loaded, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.InventoryUpdatedAt != nil {
		t.Fatal("partial account collection completed inventory")
	}
}
