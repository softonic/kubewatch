# kubewatch

[![Version Widget]][Version] [![License Widget]][License] [![GoReportCard Widget]][GoReportCard] [![Travis Widget]][Travis] [![DockerHub Widget]][DockerHub]

[Version]: https://github.com/softonic/kubewatch/releases
[Version Widget]: https://img.shields.io/github/release/softonic/kubewatch.svg?maxAge=60
[License]: http://www.apache.org/licenses/LICENSE-2.0.txt
[License Widget]: https://img.shields.io/badge/license-APACHE2-1eb0fc.svg
[GoReportCard]: https://goreportcard.com/report/softonic/kubewatch
[GoReportCard Widget]: https://goreportcard.com/badge/softonic/kubewatch
[Travis]: https://travis-ci.org/softonic/kubewatch
[Travis Widget]: https://travis-ci.org/softonic/kubewatch.svg?branch=master
[DockerHub]: https://hub.docker.com/r/softonic/kubewatch
[DockerHub Widget]: https://img.shields.io/docker/pulls/softonic/kubewatch.svg

Kubernetes API event watcher.

##### Install

```
go get -u github.com/softonic/kubewatch
```

##### Shell completion

```
eval "$(kubewatch --completion-script-${0#-})"
```

##### Help

```
kubewatch --help
usage: kubewatch [<flags>] <resources>...

Watches Kubernetes resources via its API.

Flags:
  -h, --help          Show context-sensitive help (also try --help-long and --help-man).
      --kubeconfig    Absolute path to the kubeconfig file.
      --namespace     Set the namespace to be watched.
      --version       Show application version.

Args:
  <resources>  Space delimited list of resources to be watched.
```

##### Out-of-cluster examples:

Watch for `pods` and `events` in all `namespaces`:
```
kubewatch pods events | jq '.'
```

Same thing with docker:
```
docker run -it --rm \
-v ~/.kube/config:/root/.kube/config \
softonic/kubewatch pods events | jq '.'
```

Watch for `services` events in namespace `foo`:
```
kubewatch --namespace foo services | jq '.'
```

Same thing with docker:
```
docker run -it --rm \
-v ~/.kube/config:/root/.kube/config \
softonic/kubewatch --namespace foo services | jq '.'
```

##### In-cluster examples:

Run `kubewatch` in the `monitoring` namespace and watch for `pods` in all namespaces:
```
kubectl --namespace monitoring run kubewatch --image softonic/kubewatch -- pods
```

Run `kubewatch` in the `monitoring` namespace and watch for `pods`, `deployments` and `events` objects in all namespaces. Also flatten the `json` output:
```
kubectl --namespace monitoring \
run kubewatch --image softonic/kubewatch \
-- --flatten pods deployments events
```
