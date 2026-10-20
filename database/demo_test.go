package database

import "testing"

func TestDemoMarketplaceHasThirtyUniqueSellers(t *testing.T) {
	if len(demoSellers) != 30 {
		t.Fatalf("demo seller count = %d, want 30", len(demoSellers))
	}
	seen := make(map[string]bool, len(demoSellers))
	for _, seller := range demoSellers {
		if seller.username == "" || seller.title == "" || seller.game == "" || seller.price <= 0 {
			t.Errorf("incomplete demo listing for %q", seller.username)
		}
		if seen[seller.username] {
			t.Errorf("duplicate demo username %q", seller.username)
		}
		seen[seller.username] = true
	}
}
