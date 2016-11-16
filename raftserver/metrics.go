// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raftserver

import "github.com/prometheus/client_golang/prometheus"

var (
	hasLeader = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "csf",
		Subsystem: "raftserver",
		Name:      "has_leader",
		Help:      "Whether or not a leader exists. 1 is existence, 0 is not.",
	})
	leaderChanges = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "csf",
		Subsystem: "raftserver",
		Name:      "leader_changes_seen_total",
		Help:      "The number of leader changes seen.",
	})
	proposalsCommitted = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "csf",
		Subsystem: "raftserver",
		Name:      "proposals_committed_total",
		Help:      "The total number of consensus proposals committed.",
	})
	proposalsApplied = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "csf",
		Subsystem: "raftserver",
		Name:      "proposals_applied_total",
		Help:      "The total number of consensus proposals applied.",
	})
	proposalsPending = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "csf",
		Subsystem: "raftserver",
		Name:      "proposals_pending",
		Help:      "The current number of pending proposals to commit.",
	})
	proposalsFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "csf",
		Subsystem: "raftserver",
		Name:      "proposals_failed_total",
		Help:      "The total number of failed proposals seen.",
	})
)

func init() {
	prometheus.MustRegister(hasLeader)
	prometheus.MustRegister(leaderChanges)
	prometheus.MustRegister(proposalsCommitted)
	prometheus.MustRegister(proposalsApplied)
	prometheus.MustRegister(proposalsPending)
	prometheus.MustRegister(proposalsFailed)
}
