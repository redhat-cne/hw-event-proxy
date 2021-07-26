// Copyright 2020 The Cloud Native Events Authors
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

package localmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricStatus metrics status
type MetricStatus string

const (
	// ACTIVE ...
	ACTIVE MetricStatus = "active"
	// SUCCESS ...
	SUCCESS MetricStatus = "success"
	// FAILED ...
	FAILED MetricStatus = "failed"
	// CONNECTION_RESET ...
	CONNECTION_RESET MetricStatus = "connection reset"
)

var (

	//amqpEventReceivedCount ...  Total no of events received by the transport
	amqpEventReceivedCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_amqp_events_received",
			Help: "Metric to get number of events received  by the transport",
		}, []string{"address", "status"})
	//amqpEventPublishedCount ...  Total no of events published by the transport
	amqpEventPublishedCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_amqp_events_published",
			Help: "Metric to get number of events published by the transport",
		}, []string{"address", "status"})

	//amqpConnectionResetCount ...  Total no of connection resets
	amqpConnectionResetCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_amqp_connection_reset",
			Help: "Metric to get number of connection resets",
		}, []string{})

	//amqpSenderCount ...  Total no of events published by the transport
	amqpSenderCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_amqp_sender",
			Help: "Metric to get number of sender created",
		}, []string{"address", "status"})

	//amqpReceiverCount ...  Total no of events published by the transport
	amqpReceiverCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_amqp_receiver",
			Help: "Metric to get number of receiver created",
		}, []string{"address", "status"})

	//amqpStatusCheckCount ...  Total no of status check received by the transport
	amqpStatusCheckCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cne_amqp_status_check_published",
			Help: "Metric to get number of status check published by the transport",
		}, []string{"address", "status"})
)

// RegisterMetrics ...
func RegisterMetrics() {
	prometheus.MustRegister(amqpEventReceivedCount)
	prometheus.MustRegister(amqpEventPublishedCount)
	prometheus.MustRegister(amqpConnectionResetCount)
	prometheus.MustRegister(amqpSenderCount)
	prometheus.MustRegister(amqpReceiverCount)
	prometheus.MustRegister(amqpStatusCheckCount)
}

// UpdateTransportConnectionResetCount ...
func UpdateTransportConnectionResetCount(val int) {
	amqpConnectionResetCount.With(prometheus.Labels{}).Add(float64(val))
}

// UpdateEventReceivedCount ...
func UpdateEventReceivedCount(address string, status MetricStatus, val int) {
	amqpEventReceivedCount.With(
		prometheus.Labels{"address": address, "status": string(status)}).Add(float64(val))
}

// UpdateEventCreatedCount ...
func UpdateEventCreatedCount(address string, status MetricStatus, val int) {
	amqpEventPublishedCount.With(
		prometheus.Labels{"address": address, "status": string(status)}).Add(float64(val))
}

// UpdateStatusCheckCount ...
func UpdateStatusCheckCount(address string, status MetricStatus, val int) {
	amqpEventPublishedCount.With(
		prometheus.Labels{"address": address, "status": string(status)}).Add(float64(val))
}

// UpdateSenderCreatedCount ...
func UpdateSenderCreatedCount(address string, status MetricStatus, val int) {
	amqpSenderCount.With(
		prometheus.Labels{"address": address, "status": string(status)}).Add(float64(val))
}

// UpdateReceiverCreatedCount ...
func UpdateReceiverCreatedCount(address string, status MetricStatus, val int) {
	amqpReceiverCount.With(
		prometheus.Labels{"address": address, "status": string(status)}).Add(float64(val))
}
