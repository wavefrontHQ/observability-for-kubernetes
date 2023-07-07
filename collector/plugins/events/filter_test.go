// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func TestAllowList(t *testing.T) {
	// test the previous field for backwards compat
	ef := newEventFilter(configuration.EventsFilter{
		TagWhitelist: map[string][]string{"foo": {"bar"}},
	})
	if ef.matches(map[string]string{"k": "v", "foo": "bard"}) {
		t.Errorf("error matching event tags")
	}
	if !ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}

	ef = newEventFilter(configuration.EventsFilter{
		TagAllowList: map[string][]string{"foo": {"bar"}},
	})
	if ef.matches(map[string]string{"k": "v", "foo": "bard"}) {
		t.Errorf("error matching event tags")
	}
	if !ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}
}

func TestDenyList(t *testing.T) {
	// test the previous field for backwards compat
	ef := newEventFilter(configuration.EventsFilter{
		TagBlacklist: map[string][]string{"foo": {"bar"}},
	})
	if !ef.matches(map[string]string{"k": "v", "foo": "bard"}) {
		t.Errorf("error matching event tags")
	}
	if ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}

	ef = newEventFilter(configuration.EventsFilter{
		TagDenyList: map[string][]string{"foo": {"bar"}},
	})
	if !ef.matches(map[string]string{"k": "v", "foo": "bard"}) {
		t.Errorf("error matching event tags")
	}
	if ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}
}

func TestAllowListSets(t *testing.T) {
	// test previous field for backwards compat
	ef := newEventFilter(configuration.EventsFilter{
		TagWhitelistSets: []map[string][]string{
			{
				"foo":  {"bar"},
				"food": {"bard"},
			},
		},
	})
	if ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}
	if !ef.matches(map[string]string{"foo": "bar", "food": "bard"}) {
		t.Errorf("error matching event tags")
	}

	ef = newEventFilter(configuration.EventsFilter{
		TagAllowListSets: []map[string][]string{
			{
				"foo":  {"bar"},
				"food": {"bard"},
			},
		},
	})
	if ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}
	if !ef.matches(map[string]string{"foo": "bar", "food": "bard"}) {
		t.Errorf("error matching event tags")
	}
}

func TestAllowListSetsK8sDefaultFiltering(t *testing.T) {
	// test previous field for backwards compat
	ef := newEventFilter(configuration.EventsFilter{
		TagAllowListSets: []map[string][]string{
			{
				"type": {"Warning"},
			},
			{
				"type":   {"Normal"},
				"kind":   {"Pod"},
				"reason": {"Backoff"},
			},
		},
	})

	t.Run("Matches any Warning type event", func(t *testing.T) {
		require.True(t, ef.matches(map[string]string{"type": "Warning", "reason": "failed"}))
		require.True(t, ef.matches(map[string]string{"type": "Warning", "reason": ""}))
		require.True(t, ef.matches(map[string]string{"type": "Warning", "kind": "Service"}))
	})

	t.Run("matches normal type event with kind pod and reason backoff only", func(t *testing.T) {
		require.False(t, ef.matches(map[string]string{"type": "Normal"}))
		require.False(t, ef.matches(map[string]string{"type": "Normal", "kind": "Pod"}))
		require.True(t, ef.matches(map[string]string{"type": "Normal", "kind": "Pod", "reason": "Backoff"}))
	})

}

func TestDenyListSets(t *testing.T) {
	// test previous field for backwards compat
	ef := newEventFilter(configuration.EventsFilter{
		TagBlacklistSets: []map[string][]string{
			{
				"foo":  {"bar"},
				"food": {"bard"},
			},
		},
	})
	if !ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}
	if ef.matches(map[string]string{"foo": "bar", "food": "bard"}) {
		t.Errorf("error matching event tags")
	}

	ef = newEventFilter(configuration.EventsFilter{
		TagDenyListSets: []map[string][]string{
			{
				"foo":  {"bar"},
				"food": {"bard"},
			},
		},
	})
	if !ef.matches(map[string]string{"foo": "bar"}) {
		t.Errorf("error matching event tags")
	}
	if ef.matches(map[string]string{"foo": "bar", "food": "bard"}) {
		t.Errorf("error matching event tags")
	}
}
