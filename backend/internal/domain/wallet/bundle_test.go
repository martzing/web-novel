package wallet

import (
	"errors"
	"reflect"
	"testing"
)

// The reader must never pay less than the stated percentage, so the discount
// rounds down — the same direction NetCoins rounds the platform fee.
func TestQuoteBundle_RoundsTheDiscountDownSoTheTranslatorIsNotShortchanged(t *testing.T) {
	tests := []struct {
		name         string
		prices       []int
		percent      int
		wantGross    int
		wantDiscount int
		wantTotal    int
	}{
		{"exact division", []int{5, 5, 5, 5}, 15, 20, 3, 17},
		{"rounds the discount down", []int{5, 5, 5}, 15, 15, 2, 13},
		{"one coin, discount rounds to nothing", []int{1}, 15, 1, 0, 1},
		{"no discount", []int{5, 5}, 0, 10, 0, 10},
		{"full discount", []int{5, 5}, 100, 10, 10, 0},
		{"large bundle", []int{5, 5, 5, 5, 5, 5, 5, 5, 5, 5}, 15, 50, 7, 43},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := QuoteBundle(tc.prices, tc.percent)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Gross != tc.wantGross {
				t.Fatalf("gross = %d, want %d", got.Gross, tc.wantGross)
			}
			if got.Discount != tc.wantDiscount {
				t.Fatalf("discount = %d, want %d", got.Discount, tc.wantDiscount)
			}
			if got.Total != tc.wantTotal {
				t.Fatalf("total = %d, want %d", got.Total, tc.wantTotal)
			}
			// The reader always pays at least the exact percentage.
			if got.Total*100 < got.Gross*(100-tc.percent) {
				t.Fatalf("total %d is below the exact %d%% of %d", got.Total, 100-tc.percent, got.Gross)
			}
		})
	}
}

// This is the invariant the whole design leans on: the child rows and the
// ledger row must describe the same money, or chapter_daily_stats drifts from
// coin_ledger.
func TestQuoteBundle_PerChapterAllocationSumsExactlyToTheTotal(t *testing.T) {
	tests := []struct {
		name   string
		prices []int
	}{
		{"all equal", []int{5, 5, 5, 5}},
		{"uneven", []int{3, 7, 11, 2}},
		{"prime-ish total", []int{7, 13, 17}},
		{"one chapter", []int{5}},
		{"a one-coin chapter among big ones", []int{1, 50, 50}},
		{"many small", []int{1, 1, 1, 1, 1, 1, 1}},
		{"wide spread", []int{1, 2, 3, 100}},
		{"two chapters", []int{5, 5}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, percent := range []int{0, 15, 33, 50, 99} {
				quote, err := QuoteBundle(tc.prices, percent)
				if err != nil {
					t.Fatalf("percent %d: unexpected error: %v", percent, err)
				}

				sum := 0
				for _, share := range quote.PerChapter {
					sum += share
				}
				if sum != quote.Total {
					t.Fatalf("percent %d: allocation sums to %d, want the total %d (%v)",
						percent, sum, quote.Total, quote.PerChapter)
				}
				if len(quote.PerChapter) != len(tc.prices) {
					t.Fatalf("allocation has %d entries, want %d", len(quote.PerChapter), len(tc.prices))
				}
			}
		})
	}
}

// A chapter's share must never exceed its list price, or a reader could be
// charged more for a chapter inside a bundle than outside one.
func TestAllocateProportional_NeverExceedsTheListPrice(t *testing.T) {
	weights := []int{1, 2, 3, 100}

	for total := 0; total <= 106; total++ {
		got := AllocateProportional(weights, total)

		sum := 0
		for i, share := range got {
			if share > weights[i] {
				t.Fatalf("total %d: share %d exceeds the list price %d (%v)",
					total, share, weights[i], got)
			}
			if share < 0 {
				t.Fatalf("total %d: negative share %d", total, share)
			}
			sum += share
		}
		// Above the sum of weights there is nothing left to allocate to.
		want := min(total, 106)
		if sum != want {
			t.Fatalf("total %d: allocated %d, want %d (%v)", total, sum, want, got)
		}
	}
}

func TestAllocateProportional_EdgeCases(t *testing.T) {
	if got := AllocateProportional(nil, 10); len(got) != 0 {
		t.Fatalf("nil weights = %v, want empty", got)
	}
	if got := AllocateProportional([]int{5, 5}, 0); !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatalf("zero total = %v, want [0 0]", got)
	}
	if got := AllocateProportional([]int{0, 0}, 10); !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatalf("zero weights = %v, want [0 0]", got)
	}
	if got := AllocateProportional([]int{5, 5}, -3); !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatalf("negative total = %v, want [0 0]", got)
	}
}

func TestQuoteBundle_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		prices  []int
		wantErr error
	}{
		// Nothing left to buy: the caller filters out owned chapters, so an
		// empty set means the reader already owns the arc.
		{"no chapters", nil, ErrArcAlreadyOwned},
		{"only free chapters", []int{0, 0}, ErrArcNotForSale},
		{"a negative price", []int{5, -1}, ErrInvalidAmount},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := QuoteBundle(tc.prices, 15); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestQuoteBundle_ClampsTheDiscountPercent(t *testing.T) {
	over, err := QuoteBundle([]int{10}, 150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if over.DiscountPercent != 100 || over.Total != 0 {
		t.Fatalf("quote = %+v, want the percent clamped to 100", over)
	}

	under, err := QuoteBundle([]int{10}, -20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if under.DiscountPercent != 0 || under.Total != 10 {
		t.Fatalf("quote = %+v, want the percent clamped to 0", under)
	}
}

func TestValidateTip(t *testing.T) {
	for _, coins := range []int{MinTipCoins, 5, 100, MaxTipCoins} {
		if err := ValidateTip(coins); err != nil {
			t.Fatalf("tip of %d must be accepted, got %v", coins, err)
		}
	}
	for _, coins := range []int{0, -1, MaxTipCoins + 1, 100000} {
		if err := ValidateTip(coins); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("tip of %d: error = %v, want ErrInvalidAmount", coins, err)
		}
	}
}
