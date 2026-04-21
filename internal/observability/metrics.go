package observability

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    UploadsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "avatars_uploads_total",
        },
        []string{"status"},
    )
)
