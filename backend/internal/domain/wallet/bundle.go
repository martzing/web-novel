package wallet

// ArcBundleDiscountPercent is the platform-wide discount for buying a whole
// arc (`เปิดขายเป็นภาค ผู้อ่านซื้อทั้งภาคได้ในราคาลด 15%`).
//
// It is a constant rather than a per-novel column because the writer settings
// panel offers a checkbox, not a percentage field — the number is product copy,
// not per-novel data, and a column today would let a novel set 100% and give
// chapters away. QuoteBundle still takes the percent as a parameter, so adding
// `novels.arc_discount_percent` later is a one-line change with no rewrite.
const ArcBundleDiscountPercent = 15

// BundleQuote prices a set of chapters bought together.
type BundleQuote struct {
	// Gross is the sum of the individual list prices.
	Gross           int
	DiscountPercent int
	Discount        int
	// Total is what the reader is debited: Gross - Discount.
	Total int
	// PerChapter is each chapter's share of Total, in the order the prices were
	// given. It sums to exactly Total, so the child rows and the ledger row
	// always describe the same money.
	PerChapter []int
}

// QuoteBundle prices `prices` at `discountPercent` off.
//
// Two roundings happen, both resolved in the reader's and the translator's
// favour the same way NetCoins already resolves the platform fee:
//
//   - the discount rounds down, so the reader pays at least the exact
//     percentage and never less;
//   - the discounted total is allocated back across the chapters by largest
//     remainder, so the parts sum to exactly Total. That is what keeps
//     sum(chapter_unlocks.coins_spent) equal to the ledger delta, which the
//     daily stats rollup depends on.
func QuoteBundle(prices []int, discountPercent int) (BundleQuote, error) {
	if len(prices) == 0 {
		return BundleQuote{}, ErrArcAlreadyOwned
	}

	gross := 0
	for _, p := range prices {
		if p < 0 {
			return BundleQuote{}, ErrInvalidAmount
		}
		gross += p
	}
	if gross <= 0 {
		return BundleQuote{}, ErrArcNotForSale
	}

	if discountPercent < 0 {
		discountPercent = 0
	}
	if discountPercent > 100 {
		discountPercent = 100
	}

	discount := gross * discountPercent / 100
	total := gross - discount

	return BundleQuote{
		Gross:           gross,
		DiscountPercent: discountPercent,
		Discount:        discount,
		Total:           total,
		PerChapter:      AllocateProportional(prices, total),
	}, nil
}

// AllocateProportional splits total across weights in proportion, using the
// largest-remainder method so the parts sum to exactly total.
//
// Postconditions: sum(result) == total, and no element exceeds its weight
// (a chapter's share is never more than its list price). A cheap chapter inside
// a large bundle can be allocated 0 — the reader still owns it, the translator
// simply earns nothing on that line, and the total is unaffected.
func AllocateProportional(weights []int, total int) []int {
	out := make([]int, len(weights))
	if len(weights) == 0 || total <= 0 {
		return out
	}

	sum := 0
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		return out
	}

	// Floor pass, remembering each remainder so the leftover can be handed out
	// to whoever was rounded down hardest.
	assigned := 0
	remainders := make([]int, len(weights))
	for i, w := range weights {
		scaled := w * total
		out[i] = scaled / sum
		remainders[i] = scaled % sum
		assigned += out[i]
	}

	// Distribute the leftover one coin at a time, largest remainder first,
	// breaking ties by index so the result is deterministic.
	for range total - assigned {
		best, bestRem := -1, -1
		for i := range weights {
			if out[i] >= weights[i] {
				continue // never allocate more than the list price
			}
			if remainders[i] > bestRem {
				best, bestRem = i, remainders[i]
			}
		}
		if best < 0 {
			break
		}
		out[best]++
		remainders[best] = -1
	}
	return out
}
