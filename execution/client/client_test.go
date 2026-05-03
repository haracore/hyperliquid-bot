package client

import "testing"

func TestExtractPerpPositionsFiltersZeroAndSorts(t *testing.T) {
	state := map[string]any{
		"assetPositions": []any{
			map[string]any{"position": map[string]any{
				"coin":          "ETH",
				"szi":           "0",
				"entryPx":       "3000",
				"positionValue": "0",
				"unrealizedPnl": "0",
			}},
			map[string]any{"position": map[string]any{
				"coin":          "BTC",
				"szi":           "0.01",
				"entryPx":       "70000",
				"positionValue": "700",
				"unrealizedPnl": "12",
				"marginUsed":    "70",
				"leverage":      map[string]any{"type": "cross", "value": 10},
			}},
		},
	}

	positions := ExtractPerpPositions(state, false)
	if len(positions) != 1 {
		t.Fatalf("expected 1 non-zero position, got %d", len(positions))
	}
	if positions[0].Coin != "BTC" {
		t.Fatalf("expected BTC, got %q", positions[0].Coin)
	}
	if positions[0].Leverage != "cross:10" {
		t.Fatalf("expected leverage cross:10, got %q", positions[0].Leverage)
	}
}

func TestExtractPerpPositionsShowAll(t *testing.T) {
	state := map[string]any{
		"assetPositions": []any{
			map[string]any{"position": map[string]any{"coin": "ETH", "szi": "0"}},
			map[string]any{"position": map[string]any{"coin": "BTC", "szi": "0.01"}},
		},
	}

	positions := ExtractPerpPositions(state, true)
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}
	if positions[0].Coin != "BTC" || positions[1].Coin != "ETH" {
		t.Fatalf("expected sorted positions, got %#v", positions)
	}
}
