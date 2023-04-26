# Usage

Log in to an OpenShift cluster or set the KUBECONFIG env variable, then:

```
$ go run . -network-csv ~/Downloads/export-2023-04-18-01-31.csv -exclude observability,OLM
```

This will create an threatdragon file name `output.json`, which can be imported in a Threat Dragon instance.
