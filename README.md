# kubewatch

Kubernetes API event watcher.

##### Install

```
go get -u github.com/softonic/kubewatch
```

##### Help

```
kubewatch --help
usage: kubewatch [<flags>]

Watches Kubernetes resources via its API.

Flags:
  -h, --help                 Show context-sensitive help (also try --help-long and --help-man).
      --kubeconfig           Absolute path to the kubeconfig file.
      --resource="services"  Set the resource type to be watched.
      --namespace=""         Set the namespace to be watched.
      --version              Show application version.
```

##### Shell completion

```
eval "$(kubewatch --completion-script-${0#-})"
```

##### Watch for pods evens:

```
kubewatch --resource pods
```
