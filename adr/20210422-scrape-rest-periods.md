# 20210422 Scrape Rest Periods

## Context
Scrapes that take longer to process than their collection interval started their next scrape immediately after finishing the previous. This left no time for garbage collection, ultimately causing our memory usage to spike.
This happens more frequently when the collector pod has too low of a CPU limit.

## Decision
Rather than try to "catch up" on a missed scrape interval, the collector will skip the interval. It will also leave a buffer if a scape finishes working too close to the next interval's start.
It computes the remaining interval each scrape has before the next is supposed to trigger and drops the next scrape if it's within 3.33% of the next one to allow the garbage collector more time to collect.

## Status: Implemented
[PR](https://github.com/wavefrontHQ/observability-for-kubernetes/commit/042eea52c37c9595e8e033cd7eed2ffdb00d1310)

## Consequences
If processing scrape takes longer than this threshold, we will skip collection intervals which will lead to missing data.
Let the CPU catch up to cut down on memory bloat; it stopped masking a CPU starvation issue as a memory issue.
