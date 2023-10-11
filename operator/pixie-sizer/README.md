Pixing Sizing Tool
===

This tool recommends memory settings that ensure [Pixie PEM][pem-docs-link] memory is adequately sized.

## Running

```bash
# Make a working directory
mkdir /tmp/pixie-sizer
cd /tmp/pixie-sizer

# Download the install script
curl -O install.sh https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/operator/pixie-sizer/install.sh
chmod +x install.sh

# Install the sizer into a cluster that has Pixie enabled
PS_TRAFFIC_SCALE_FACTOR=1.5 PS_MEASURE_MINUTES:=480 ./install.sh
```

* `PS_TRAFFIC_SCALE_FACTOR`: multiplier applied to the target retention time for rows in the [data tables][data-tables-link].
    + A value of 1.0 will result in enough memory to satisfy the data retention needs of installed pixie scripts for the median amount of traffic.
    + A value greater than 1.0, will result in extra capacity to handle bursty traffic.
      For example, a value of 2.0 would allocate enough memory to handle twice the median volume of traffic measured in the sample period.
    + A value less than 1.0 will result in less memory than is necessary to satisfy the data retention needs of installed pixie scripts.
      This can be useful for environments that want to conserve memory (like development or test).
* `PS_MEASURE_MINUTES`: minutes of PEM metric data to collect before making a recommendation.

[pem-docs-link]: https://docs.px.dev/about-pixie/what-is-pixie/#architecture
[data-tables-link]: https://docs.px.dev/reference/datatables/