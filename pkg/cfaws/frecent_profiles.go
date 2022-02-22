package cfaws

import (
	"os"
	"sort"

	"github.com/common-fate/granted/pkg/debug"
	"github.com/common-fate/granted/pkg/frecency"
	"github.com/pkg/errors"
)

var frecencyStoreKey = "aws_profiles_frecency"

type FrecentProfiles struct {
	store    *frecency.FrecencyStore
	toRemove []string
}

// should be called after selecting a profile to update frecency cache
// wrap this method in a go routine to avoid blocking the user
func (f *FrecentProfiles) Update(selectedProfile string) {
	s := make([]interface{}, len(f.toRemove))
	for i, v := range f.toRemove {
		s[i] = v
	}
	err := f.store.DeleteAll(s)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "removing entries from frecency").Error())
	}
	err = f.store.Upsert(selectedProfile)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "upserting entry to frecency").Error())
	}
}

// use this to update frecency cache when the profile is supplied by the commandline
func UpdateFrecencyCache(selectedProfile string) {
	fr, err := frecency.Load(frecencyStoreKey)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "loading aws_profiles_frecency frecency store").Error())
	} else {
		err = fr.Upsert(selectedProfile)
		if err != nil {
			debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "upserting entry to frecency").Error())
		}
	}
}

// loads the frecency cache and generates a list of profiles with frecently used profiles first, followed by alphabetically sorted profiles that have not been used with assume
// this method returns a FrecentProfiles pointer which should be used after selecting a profile to update the cache, it will also remove any entries which no longer exist in the aws config
func (c CFSharedConfigs) GetFrecentProfiles() (*FrecentProfiles, []string) {
	names := []string{}
	namesMap := make(map[string]string)
	profileNames := c.ProfileNames()
	pnMap := make(map[string]string)
	for _, pn := range profileNames {
		pnMap[pn] = pn
	}
	fr, err := frecency.Load(frecencyStoreKey)
	if err != nil {
		debug.Fprintf(debug.VerbosityDebug, os.Stderr, errors.Wrap(err, "loading aws_profiles_frecency frecency store").Error())
	}
	namesToRemoveFromFrecency := []string{}

	// add all frecent profile names in order if they are still present in the profileNames slice
	for _, entry := range fr.Entries {
		e := entry.Entry.(string)
		if _, ok := pnMap[e]; ok {
			names = append(names, e)
			namesMap[e] = e
		} else {
			namesToRemoveFromFrecency = append(namesToRemoveFromFrecency, e)
		}
	}

	// add all other entries from profileNames, sort them alphabetically first
	namesToSort := []string{}
	for _, n := range profileNames {
		if _, ok := namesMap[n]; !ok {
			namesToSort = append(namesToSort, n)
		}
	}
	sort.Strings(namesToSort)
	names = append(names, namesToSort...)

	frPr := &FrecentProfiles{store: fr, toRemove: namesToRemoveFromFrecency}

	return frPr, names
}
