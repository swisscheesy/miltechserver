package short

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/item_query/shared"
)

type repoStub struct {
	niinResp model.NiinLookup
	niinErr  error
	partResp []model.NiinLookup
	partErr  error
}

func (r *repoStub) ShortItemSearchNiin(string) (model.NiinLookup, error) {
	return r.niinResp, r.niinErr
}

func (r *repoStub) ShortItemSearchPart(string) ([]model.NiinLookup, error) {
	return r.partResp, r.partErr
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

func TestFindShortByNiinTracksAnalytics(t *testing.T) {
	niin := "123456789"
	name := "Widget"
	stub := &repoStub{niinResp: model.NiinLookup{Niin: &niin, ItemName: &name}}
	analytics := &analyticsStub{}
	svc := NewService(stub, analytics)

	result, err := svc.FindShortByNiin(niin)
	require.NoError(t, err)
	require.Equal(t, niin, deref(result.Niin))
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
	require.Len(t, analytics.calls, 1)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
