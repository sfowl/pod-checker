# Work in progress

Reads openshift/kubernetes cluster data, to automatically produce security reports.

## Usage

### Gather data

Log in to an OpenShift cluster or set the KUBECONFIG env variable, then:

```
$ go run . -network-csv ./example/input/network-traffic.csv -exclude observability,OLM
```

A network-traffic.csv file is a CSV of network traffic data, exported by the [network observability operator](https://docs.openshift.com/container-platform/4.12/networking/network_observability/network-observability-overview.html). Network traffic data like this is necessary to create the links between components in the final report data.

This will:
* print detailed component data in csv form to stdout
* create a threatdragon file `output.json`, which can be imported in a [Threat Dragon](https://github.com/OWASP/threat-dragon) instance.

### Threagile

Using the data harvested in the previous step, produce threagile reports:

```
$ podman run --user root --rm -it -v ./example:/app/work:Z threagile  -model /app/work/output/threagile_input.yaml -output /app/work/output/threagile
```

This will produce a variety of threagile outputs in the `example/output/threagile` directory.

## TODO

* augment data with user supplied component info (e.g. in yaml format)
  - qualitative data (e.g. from survey)
  - data not retrievable from API (e.g. systemd services like kubelet, cri-o, sshd)
  - custom groupings of namespaces/components (currently hardcoded)
* de-openshift-ify, support vanilla kube
* support other network traffic formats
* read more security info from clusters
* add more security info to threatdragon diagrams
* improve layout of threatdragon diagrams

## Examples

Threat Dragon output diagram:

![screenshot](/example/screenshot_threatdragon.png)

Threagile report pdf:

![pdf](/example/screenshot_threagile.png)
