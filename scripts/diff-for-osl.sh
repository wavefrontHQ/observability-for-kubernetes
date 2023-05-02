#!/usr/bin/env bash

git diff v2.5.0..HEAD -- collector/go.mod | grep -e '^+\s' | awk '{print $2}' | sort > additions.txt
git diff v2.5.0..HEAD -- collector/go.mod | grep -e '^-\s' | awk '{print $2}' | sort > removals.txt

