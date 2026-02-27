package short

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/item_query/shared"
)

// repoStub satisfies the Repository interface for unit testing.
type repoStub struct {
	niinResp      model.NiinLookup
	niinErr       error
	partResp      []model.NiinLookup
	partErr       error
	cancelledResp []model.Nsn
	cancelledErr  error
	// niinRespByCall allows different responses per sequential ShortItemSearchNiin call.
	// If non-nil, each call pops the first element. Falls back to niinResp/niinErr when exhausted.
	niinRespByCall []niinCall
}

type niinCall struct {
	resp model.NiinLookup
	err  error
}

func (r *repoStub) ShortItemSearchNiin(string) (model.NiinLookup, error) {
	if len(r.niinRespByCall) > 0 {
		call := r.niinRespByCall[0]
		r.niinRespByCall = r.niinRespByCall[1:]
		return call.resp, call.err
	}
	return r.niinResp, r.niinErr
}

func (r *repoStub) ShortItemSearchPart(string) ([]model.NiinLookup, error) {
	return r.partResp, r.partErr
}

func (r *repoStub) ShortItemSearchCancelledNiin(string) ([]model.Nsn, error) {
	return r.cancelledResp, r.cancelledErr
}

type analyticsStub struct {
	calls []string
	fail  bool
}

func (a *analyticsStub) IncrementItemSearchSuccess(niin string, nomenclature string) error {
	a.calls = append(a.calls, niin+":"+nomenclature)
	if a.fail {
		return errors.New("analytics down")
	}
	return nil
}

// ---- Existing tests (unchanged) ----

func TestFindShortByNiinTracksAnalytics(t *testing.T) {
	niin := "123456789"
	name := "Widget"
	stub := &repoStub{niinResp: model.NiinLookup{Niin: &niin, ItemName: &name}}
	analytics := &analyticsStub{}
	svc := NewService(stub, analytics)

	result, err := svc.FindShortByNiin(niin)
	require.NoError(t, err)
	require.Equal(t, niin, deref(result.Niin))

	// Wait for async analytics processing
	time.Sleep(10 * time.Millisecond)
	require.Len(t, analytics.calls, 1)
}

func TestFindShortByPartTracksUniqueNiins(t *testing.T) {
	niin := "A123"
	name := "Widget"
	stub := &repoStub{partResp: []model.NiinLookup{{Niin: &niin, ItemName: &name}, {Niin: &niin, ItemName: &name}}}
	analytics := &analyticsStub{}
	svc := NewService(stub, analytics)

	results, err := svc.FindShortByPart("part")
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Wait for async analytics processing
	time.Sleep(10 * time.Millisecond)
	require.Len(t, analytics.calls, 1)
}

func TestFindShortByNiinReturnsRepoError(t *testing.T) {
	stub := &repoStub{niinErr: shared.ErrNoItemsFound}
	svc := NewService(stub, &analyticsStub{})

	_, err := svc.FindShortByNiin("bad")
	require.ErrorIs(t, err, shared.ErrNoItemsFound)
}

func TestFindShortByNiinAnalyticsFailureDoesNotFail(t *testing.T) {
	niin := "013469317"
	name := "Widget"
	stub := &repoStub{niinResp: model.NiinLookup{Niin: &niin, ItemName: &name}}
	analytics := &analyticsStub{fail: true}
	svc := NewService(stub, analytics)

	result, err := svc.FindShortByNiin(niin)
	require.NoError(t, err)
	require.Equal(t, niin, deref(result.Niin))

	// Wait for async analytics processing
	time.Sleep(10 * time.Millisecond)
	require.Len(t, analytics.calls, 1)
}

// ---- New tests for FindShortByNiinCancelled ----

// TestFindShortByNiinCancelledPrimaryHit verifies that when the primary niin_lookup
// search succeeds, the result is returned as a single-element slice with no fallback.
func TestFindShortByNiinCancelledPrimaryHit(t *testing.T) {
	niin := "123456789"
	name := "Widget"
	stub := &repoStub{niinResp: model.NiinLookup{Niin: &niin, ItemName: &name}}
	analytics := &analyticsStub{}
	svc := NewService(stub, analytics)

	results, err := svc.FindShortByNiinCancelled(niin)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, niin, deref(results[0].Niin))

	time.Sleep(10 * time.Millisecond)
	require.Len(t, analytics.calls, 1)
}

// TestFindShortByNiinCancelledFallbackSucceeds verifies the two-step fallback:
// primary miss → cancelled_niin hit → re-query niin_lookup → results returned.
func TestFindShortByNiinCancelledFallbackSucceeds(t *testing.T) {
	canonicalNiin := "987654321"
	name := "Replacement Widget"
	cancelledNiinVal := "OLD123 OLD456"

	stub := &repoStub{
		// First call (primary) misses, second call (re-lookup with canonical) hits.
		niinRespByCall: []niinCall{
			{err: shared.ErrNoItemsFound},
			{resp: model.NiinLookup{Niin: &canonicalNiin, ItemName: &name}},
		},
		cancelledResp: []model.Nsn{
			{Niin: canonicalNiin, CancelledNiin: &cancelledNiinVal},
		},
	}
	analytics := &analyticsStub{}
	svc := NewService(stub, analytics)

	results, err := svc.FindShortByNiinCancelled("OLD123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, canonicalNiin, deref(results[0].Niin))

	time.Sleep(10 * time.Millisecond)
	require.Len(t, analytics.calls, 1)
}

// TestFindShortByNiinCancelledBothMiss verifies that ErrNoItemsFound is returned
// when neither the primary nor the cancelled_niin search finds anything.
func TestFindShortByNiinCancelledBothMiss(t *testing.T) {
	stub := &repoStub{
		niinErr:      shared.ErrNoItemsFound,
		cancelledErr: shared.ErrNoItemsFound,
	}
	svc := NewService(stub, &analyticsStub{})

	_, err := svc.FindShortByNiinCancelled("UNKNOWN")
	require.ErrorIs(t, err, shared.ErrNoItemsFound)
}

// TestFindShortByNiinCancelledAllCanonicalSkipped verifies that if cancelled matches
// are found but none of the canonical NIINs resolve in niin_lookup, ErrNoItemsFound
// is returned rather than an empty slice.
func TestFindShortByNiinCancelledAllCanonicalSkipped(t *testing.T) {
	cancelledNiinVal := "GHOST"
	stub := &repoStub{
		// Primary miss, then canonical re-lookup also misses.
		niinRespByCall: []niinCall{
			{err: shared.ErrNoItemsFound},
			{err: shared.ErrNoItemsFound},
		},
		cancelledResp: []model.Nsn{
			{Niin: "CANONICAL1", CancelledNiin: &cancelledNiinVal},
		},
	}
	svc := NewService(stub, &analyticsStub{})

	_, err := svc.FindShortByNiinCancelled("GHOST")
	require.ErrorIs(t, err, shared.ErrNoItemsFound)
}

// TestFindShortByNiinCancelledDeduplicatesCanonical verifies that if multiple nsn rows
// share the same canonical NIIN, only one niin_lookup call is made for that NIIN.
func TestFindShortByNiinCancelledDeduplicatesCanonical(t *testing.T) {
	canonicalNiin := "CANONICAL"
	name := "Widget"
	cancelledVal := "OLD"

	stub := &repoStub{
		niinRespByCall: []niinCall{
			{err: shared.ErrNoItemsFound},                                   // primary miss
			{resp: model.NiinLookup{Niin: &canonicalNiin, ItemName: &name}}, // re-lookup
		},
		cancelledResp: []model.Nsn{
			{Niin: canonicalNiin, CancelledNiin: &cancelledVal},
			{Niin: canonicalNiin, CancelledNiin: &cancelledVal}, // duplicate
		},
	}
	analytics := &analyticsStub{}
	svc := NewService(stub, analytics)

	results, err := svc.FindShortByNiinCancelled("OLD")
	require.NoError(t, err)
	require.Len(t, results, 1)

	time.Sleep(10 * time.Millisecond)
	require.Len(t, analytics.calls, 1)
}

// TestFindShortByNiinCancelledPrimaryUnexpectedError verifies that a non-ErrNoItemsFound
// error from the primary search is propagated immediately without attempting the fallback.
func TestFindShortByNiinCancelledPrimaryUnexpectedError(t *testing.T) {
	unexpected := errors.New("db connection failed")
	stub := &repoStub{niinErr: unexpected}
	svc := NewService(stub, &analyticsStub{})

	_, err := svc.FindShortByNiinCancelled("any")
	require.ErrorIs(t, err, unexpected)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
