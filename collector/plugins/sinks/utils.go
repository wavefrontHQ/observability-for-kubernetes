// Copyright 2018-2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package sinks

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	emptyReason       = "they were empty"
	excludeListReason = "they were on an exclude list"
	dedupeReason      = "there were too many tags so we removed tags with duplicate tag values"
	iaasReason        = "there were too many tags so we removed IaaS specific tags"
	alphaBetaReason   = "there were too many tags so we removed alpha and beta tags"
	extraTagsReason   = "there were too many tags so we removed label.* tags"
)

var alphaBetaRegex = regexp.MustCompile("^label.*beta|alpha*")
var iaasNameRegex = regexp.MustCompile("^label.*gke|azure|eks*")
var labelNameRegex = regexp.MustCompile("^label.*")

// cleanTags removes empty, excluded tags, and tags with duplicate values (if there are too many tags) and returns a map
// that lists removed tag names by their reason for removal
func cleanTags(tags map[string]string, tagGuaranteeList []string, maxCapacity int) map[string][]string {
	removedReasons := map[string][]string{}
	removedReasons[emptyReason] = removeEmptyTags(tags)

	// Exclude tags irrespective of annotation count as long as they are not in the guarantee list.
	removedReasons[excludeListReason] = excludeTags(tags)

	// Split include tags and adjust maxCapacity
	tagsToGuarantee, tagsToGuaranteeSize := splitGuaranteedTags(tags, tagGuaranteeList)
	adjustedMaxCapacity := maxCapacity - tagsToGuaranteeSize
	if len(tags) > adjustedMaxCapacity {
		removedReasons[dedupeReason] = dedupeTagValues(tags)
	}

	// remove other tags if we are still over the max capacity
	if len(tags) > adjustedMaxCapacity {
		removedReasons[alphaBetaReason] = removeTagsLabelsMatching(tags, alphaBetaRegex, len(tags)-adjustedMaxCapacity)
		removedReasons[iaasReason] = removeTagsLabelsMatching(tags, iaasNameRegex, len(tags)-adjustedMaxCapacity)
		removedReasons[extraTagsReason] = removeTagsLabelsMatching(tags, labelNameRegex, len(tags)-adjustedMaxCapacity)
	}

	combineTags(tagsToGuarantee, tags)

	return removedReasons
}

func combineTags(include map[string]string, tags map[string]string) {
	for includedTagKey, includedTagVal := range include {
		tags[includedTagKey] = includedTagVal
	}
}

func splitGuaranteedTags(tags map[string]string, tagGuaranteeList []string) (map[string]string, int) {
	var tagsToGuarantee = make(map[string]string)
	for _, tagKey := range tagGuaranteeList {
		if val, ok := tags[tagKey]; ok {
			tagsToGuarantee[tagKey] = val
		}
		delete(tags, tagKey)
	}

	for tagKey, val := range tags {
		if !labelNameRegex.MatchString(tagKey) {
			tagsToGuarantee[tagKey] = val
			delete(tags, tagKey)
		}
	}
	return tagsToGuarantee, len(tagsToGuarantee)
}

func logTagCleaningReasons(metricName string, reasons map[string][]string) {
	for reason, tagNames := range reasons {
		if len(tagNames) == 0 {
			continue
		}
		log.Debugf(
			"the following tags were removed from %s because %s: %s",
			metricName, reason, strings.Join(tagNames, ", "),
		)
	}
}

const minDedupeTagValueLen = 5

func dedupeTagValues(tags map[string]string) []string {
	var removedTags []string
	invertedTags := make(map[string]string) // tag value -> tag name
	tagNames := sortKeys(tags)
	for _, name := range tagNames {
		value := tags[name]
		if len(value) < minDedupeTagValueLen {
			continue
		}

		if _, ok := invertedTags[value]; !ok {
			invertedTags[value] = name
			continue
		}

		nameToKeep, nameToRemove := compareTagNames(name, invertedTags[value])
		removedTags = append(removedTags, nameToRemove)
		delete(tags, nameToRemove)
		invertedTags[value] = nameToKeep
	}
	return removedTags
}

func compareTagNames(name string, prevWinner string) (string, string) {
	if alphaBetaRegex.MatchString(name) {
		return prevWinner, name
	}

	if alphaBetaRegex.MatchString(prevWinner) {
		return name, prevWinner
	}

	if len(name) < len(prevWinner) {
		return name, prevWinner
	} else if len(prevWinner) < len(name) {
		return prevWinner, name
	}

	if name < prevWinner {
		return name, prevWinner
	} else {
		return prevWinner, name
	}
}

func isAnEmptyTag(value string) bool {
	if value == "" || value == "/" || value == "-" {
		return true
	}
	return false
}

func removeEmptyTags(tags map[string]string) []string {
	var removed []string
	for name, value := range tags {
		if isAnEmptyTag(value) {
			removed = append(removed, name)
			delete(tags, name)
		}
	}
	return removed
}

func removeTagsLabelsMatching(tags map[string]string, regexp *regexp.Regexp, numberToRemove int) []string {
	var removed []string
	count := 0
	tagNames := sortKeys(tags)
	for _, name := range tagNames {
		if count >= numberToRemove {
			break
		}
		if regexp.MatchString(name) {
			removed = append(removed, name)
			delete(tags, name)
			count++
		}
	}
	return removed
}

func excludeTags(tags map[string]string) []string {
	var removed []string
	for name := range tags {
		if excludeTag(name) {
			removed = append(removed, name)
			delete(tags, name)
		}
	}
	return removed
}

func excludeTag(name string) bool {
	for _, excludeName := range excludeTagList {
		if excludeName == name {
			return true
		}
	}
	return false
}
